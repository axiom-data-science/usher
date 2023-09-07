package main

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
	"time"

	"github.com/rjeczalik/notify"
)

type FileMapper interface {
	GetFileDestPath(relSrcFile string, absSrcFile string,
		mappedRootSrcPath string, mappedRootDestPath string,
		relToMappedRootSrcFile string) (string, error)
}

type ExternalFileMapper struct {
	Executable string
}

func (fm ExternalFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	mappedRootSrcPath string, mappedRootDestPath string,
	relToMappedRootSrcFile string) (string, error) {
	out, err := exec.Command(fm.Executable, relSrcFile, absSrcFile,
		mappedRootSrcPath, mappedRootDestPath, relToMappedRootSrcFile).Output()
	if err != nil {
		log.Println(err)
		return "", errors.New("error when processing " + relSrcFile +
			"using external mapper " + fm.Executable + ": " + err.Error())
	}
	return string(out), nil
}

type IfcbFileMapper struct{}

func (fm IfcbFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	mappedRootSrcPath string, mappedRootDestPath string,
	relToMappedRootSrcFile string) (string, error) {
	//D20230525T192231_IFCB162.adc
	base := path.Base(relSrcFile)
	if base[0] != 'D' {
		return "", errors.New("file " + relSrcFile + " does not start with D prefix, ignoring")
	} else if len(base) < 16 {
		return "", errors.New("file " + relSrcFile + " has a base filename less than 16 characters, ignoring")
	}

	fileTime, err := time.Parse("20060102T150405", base[1:16])
	if err != nil {
		log.Println(err)
		return "", errors.New("couldn't parse date from file " + relSrcFile + ", ignoring")
	}
	destPath := fileTime.Format("2006/D20060102/") + base
	return destPath, nil
}

var FileMappers map[string]FileMapper

func init() {
	FileMappers = make(map[string]FileMapper)
	FileMappers["ifcb"] = IfcbFileMapper{}
}

func GetFileMapper(fileMapperRef string) (FileMapper, error) {
	if len(fileMapperRef) == 0 {
		if len(FileMappers) == 1 {
			for k := range FileMappers {
				fileMapperRef = k
				break
			}
			fmt.Println("Using file mapper", fileMapperRef)
		} else {
			return nil, errors.New("FileMapper type was not provided, " + getFileMapperRefs())
		}
	}

	fileMapper, ok := FileMappers[fileMapperRef]
	if !ok {
		//if we didn't find a known filen mapper, look for a matching exeternal executable
		_, err := exec.LookPath(fileMapperRef)
		if err == nil {
			//create a fileMapper which will call the external executable as the mapper
			fileMapper = ExternalFileMapper{
				Executable: fileMapperRef,
			}
			ok = true
		}
	}
	if !ok {
		return nil, errors.New("FileMapper " + fileMapperRef + " does not exist, " + getFileMapperRefs())
	}
	return fileMapper, nil
}

func getFileMapperRefs() string {
	refs := make([]string, 0)
	for k := range FileMappers {
		refs = append(refs, k)
	}
	sort.Strings(refs)
	return "valid FileMappers: " + strings.Join(refs, ", ")
}

func processFile(config Config, absSrcFile string) {
	srcDirAbs, _ := filepath.Abs(config.SrcDir)
	relSrcFile := strings.TrimPrefix(absSrcFile, srcDirAbs+"/")
	base := path.Base(absSrcFile)
	//skip files starting with .  //rsync prepends . to files currently being transferred
	if base[0] == '.' {
		if config.Debug {
			log.Println("file", relSrcFile, "starts with ., ignoring")
		}
		return
	}

	var mappedRootSrcPath, mappedRootDestPath, relToMappedRootSrcFile string
	if len(config.RootPathMappings) > 0 {
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
				mappedRootSrcPath = rootPath
				mappedRootDestPath = config.RootPathMappings[rootPath]
				relToMappedRootSrcFile = strings.TrimPrefix(relSrcFile, rootPath+"/")
				break
			}
		}

		if !mappedRootPathFound {
			log.Println("No root path mapping found for", relSrcFile, "(skipping)")
			return
		}
	}

	relDestFile, err := config.FileMapper.GetFileDestPath(
		relSrcFile, absSrcFile, mappedRootSrcPath, mappedRootDestPath, relToMappedRootSrcFile)

	if err != nil {
		//TODO copy to unhandled directory?
		if config.Debug {
			log.Println(err)
		}
		return
	}

	//if relDestFile is multiline, just use the first (could be an artifact of external executables)
	relDestFile = strings.Split(relDestFile, "\n")[0]

	//prepend the mapped root if mapped roots are configured
	if len(mappedRootDestPath) > 0 {
		relDestFile = mappedRootDestPath + "/" + relDestFile
	}

	destFile := config.DestDir + "/" + relDestFile
	destParentDir := path.Dir(destFile)
	os.MkdirAll(destParentDir, os.ModePerm)

	//check for existing file and delete if it's not the same
	srcStat, err := os.Stat(absSrcFile)
	if err != nil {
		log.Println(err)
		return
	}
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
			if err := os.Remove(destFile); err != nil {
				log.Println("failed to delete file", destFile, err)
				return
			}
		}
	}

	if config.Copy {
		log.Println(absSrcFile, "--copy-->", destFile)
		err = copyFile(absSrcFile, destFile)
	} else {
		log.Println(absSrcFile, "--link-->", destFile)
		err = os.Link(absSrcFile, destFile)
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

	os.MkdirAll(config.SrcDir, os.ModePerm)

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
