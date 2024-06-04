package usher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rjeczalik/notify"
)

type FileMapper interface {
	GetFileDestPath(relSrcFile string, absSrcFile string, baseSrcFile string,
		mappedRootSrcPath string, mappedRootDestPath string) (string, error)
}

type DelegatingFileMapper struct {
	GetFileDestPathFunc func(relSrcFile string, absSrcFile string, baseSrcFile string,
		mappedRootSrcPath string, mappedRootDestPath string) (string, error)
}

func (fm *DelegatingFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	baseSrcFile string, mappedRootSrcPath string, mappedRootDestPath string) (string, error) {
	return fm.GetFileDestPathFunc(relSrcFile, absSrcFile, baseSrcFile, mappedRootSrcPath, mappedRootDestPath)
}

func NewFileMapper(getFileDestPathFunc func(relSrcFile string, absSrcFile string,
	baseSrcFile string, mappedRootSrcPath string, mappedRootDestPath string) (string, error)) *DelegatingFileMapper {
	return &DelegatingFileMapper{getFileDestPathFunc}
}

var fileMappers map[string]FileMapper = make(map[string]FileMapper)

func SetFileMappers(newFileMappers map[string]FileMapper) {
	fileMappers = newFileMappers
}

func RegisterFileMapper(name string, fileMapper FileMapper) {
	fileMappers[name] = fileMapper
}

func GetFileMapper(fileMapperRef string, debug bool) (FileMapper, error) {
	if debug {
		fmt.Println("known mappers:", getFileMapperRefs())
	}

	if len(fileMapperRef) == 0 {
		if len(fileMappers) == 1 {
			for k := range fileMappers {
				fileMapperRef = k
				break
			}
			fmt.Println("using file mapper", fileMapperRef)
		} else {
			return nil, errors.New("FileMapper type was not provided, " + getFileMapperRefs())
		}
	}

	fileMapper, ok := fileMappers[fileMapperRef]
	if !ok {
		//if we didn't find a known filen mapper, look for a matching exeternal executable
		_, err := exec.LookPath(fileMapperRef)
		if err == nil {
			//create a fileMapper which will call the external executable as the mapper
			fileMapper = NewExternalFileMapper(fileMapperRef)
			ok = true
		} else if strings.Contains(err.Error(), "permission denied") {
			log.Fatal(err)
		}
	}
	if !ok {
		return nil, errors.New("mapper " + fileMapperRef + " does not exist, valid mappers: " + getFileMapperRefs())
	}
	return fileMapper, nil
}

func getFileMapperRefs() string {
	refs := make([]string, 0)
	for k := range fileMappers {
		refs = append(refs, k)
	}
	sort.Strings(refs)
	return strings.Join(refs, ", ")
}

func processFile(config Config, absSrcFile string) {
	srcStat, err := os.Stat(absSrcFile)
	if err != nil {
		//if source file doesn't exist and not in debug mode, don't
		//bother logging (file was deleted before we processed it)
		if config.Debug || !errors.Is(err, os.ErrNotExist) {
			log.Println(err)
		}
		return
	}
	srcDirAbs, _ := filepath.Abs(config.SrcDir)
	relSrcFile := strings.TrimPrefix(absSrcFile, srcDirAbs+"/")

	if srcStat.IsDir() {
		if config.Debug {
			log.Println("Ignoring directory", relSrcFile)
			return
		}
	}

	baseSrcFile := path.Base(absSrcFile)
	//skip files starting with .  //rsync prepends . to files currently being transferred
	if baseSrcFile[0] == '.' {
		if config.Debug {
			log.Println("file", relSrcFile, "starts with ., ignoring")
		}
		return
	}

	var relToRelevantRootSrcFile, mappedRootSrcPath, mappedRootDestPath string
	if len(config.RootPathMappings) == 0 {
		//if root path mappings are not used, we set relToRelevantRootSrcFile
		//to relSrcFile
		relToRelevantRootSrcFile = relSrcFile
	} else {
		//if srcFile begins with a map key in config.RootPathMappings,
		//prefix the destination with the map value
		//to place all unmatched files into a directory, provide a root path mapping
		//with a zero length string for a key (e.g. "":unmatched)
		//otherwise files with unmatched root paths are ignored

		//reverse sort config.RootPathMapping keys to find more specific matches first
		rootPathMappingKeys := make([]string, 0, len(config.RootPathMappings))
		for rootPath := range config.RootPathMappings {
			rootPathMappingKeys = append(rootPathMappingKeys, rootPath)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(rootPathMappingKeys)))

		var mappedRootPathFound bool = false
		for _, rootPath := range rootPathMappingKeys {
			if strings.HasPrefix(relSrcFile, rootPath) {
				mappedRootPathFound = true
				relToRelevantRootSrcFile = strings.TrimPrefix(relSrcFile, rootPath+"/")
				mappedRootSrcPath = rootPath
				mappedRootDestPath = config.RootPathMappings[rootPath]
				break
			}
		}

		if !mappedRootPathFound {
			log.Println("No root path mapping found for", relSrcFile, "(skipping)")
			return
		}
	}

	relDestFile, err := config.FileMapper.GetFileDestPath(
		relToRelevantRootSrcFile, absSrcFile, baseSrcFile, mappedRootSrcPath, mappedRootDestPath)

	if err != nil {
		//TODO copy to unhandled directory?
		if config.Debug {
			log.Println(err)
		}
		return
	}

	//if relDestFile is multiline, just use the first (could be an artifact of external executables)
	relDestFile = strings.Split(relDestFile, "\n")[0]

	//append the mapped root to dest dir if mapped roots are configured
	destDir := config.DestDir
	if len(mappedRootDestPath) > 0 {
		destDir += "/" + mappedRootDestPath
	}

	destFile, _ := filepath.Abs(destDir + "/" + relDestFile)

	//make sure destFile is inside config root dest dir and mapped root (if applicable),
	//aka relative paths weren't used to climb out of it
	if !strings.HasPrefix(destFile, config.DestDir) {
		log.Println(destFile, "is not contained by", config.DestDir, "(skipping)")
		return
	}
	if len(mappedRootDestPath) > 0 && !strings.HasPrefix(destFile, destDir) {
		log.Println(destFile, "is not contained by mapped root", destDir, "(skipping)")
		return
	}

	destParentDir := path.Dir(destFile)
	if !config.DryRun {
		os.MkdirAll(destParentDir, os.ModePerm)
	}

	//check for existing file and delete if it's not the same
	if destStat, err := os.Stat(destFile); err == nil {
		if os.SameFile(srcStat, destStat) {
			if config.Debug {
				log.Println("file", destFile, "exists and is the same as src file, skipping")
			}
			return
		} else {
			if config.Debug {
				log.Println("file", destFile, "exists and is not the same as src file, deleting")
			}
			if !config.DryRun {
				if err := os.Remove(destFile); err != nil {
					log.Println("failed to delete file", destFile, err)
					return
				}
			}
		}
	}

	if config.Copy {
		log.Println(absSrcFile, "--copy-->", destFile)
		if !config.DryRun {
			err = copyFile(absSrcFile, destFile)
		}
	} else {
		log.Println(absSrcFile, "--link-->", destFile)
		if !config.DryRun {
			err = os.Link(absSrcFile, destFile)
		}
	}

	if err != nil {
		log.Println(err)
		return
	}
}

func copyFile(srcPath string, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

func Watch(config Config) {
	c := make(chan notify.EventInfo, config.Watch.EventBufferSize)

	if !config.DryRun {
		os.MkdirAll(config.SrcDir, os.ModePerm)
	}

	if err := notify.Watch(config.SrcDir+"/...", c, notify.All); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	for eventInfo := range c {
		if config.Debug {
			log.Println("Detected event", eventInfo.Event(), "for file", eventInfo.Path())
		}
		if eventInfo.Event() == notify.Create || eventInfo.Event() == notify.Write {
			if config.Debug {
				log.Println("Processing event", eventInfo.Event(), "for file", eventInfo.Path())
			}
			processFile(config, eventInfo.Path())
		}
	}
}

func Process(config Config) {
	if _, err := os.Stat(config.SrcDir); err != nil {
		log.Fatalf("Source directory " + config.SrcDir + " does not exist")
	}

	err := filepath.WalkDir(config.SrcDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			//skip directories
			return nil
		}
		processFile(config, path)
		return nil
	})
	if err != nil {
		log.Fatalf("Could not walk source directory " + config.SrcDir + ", " + err.Error())
	}
}
