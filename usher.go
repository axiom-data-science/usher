package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rjeczalik/notify"
)

type FileMapper interface {
	GetFileDestPath(file string) (string, error)
}

type IfcbFileMapper struct{}

func (fm IfcbFileMapper) GetFileDestPath(file string) (string, error) {
	//D20230525T192231_IFCB162.adc
	base := path.Base(file)
	if base[0] != 'D' {
		return "", errors.New("file " + file + " does not start with D prefix, ignoring")
	} else if len(base) < 16 {
		return "", errors.New("file " + file + " has a base filename less than 16 characters, ignoring")
	}

	fileTime, err := time.Parse("20060102T150405", base[1:16])
	if err != nil {
		log.Println(err)
		return "", errors.New("couldn't parse date from file " + file + ", ignoring")
	}
	destPath := fileTime.Format("2006/D20060102/") + base
	return destPath, nil
}

var FileMappers map[string]FileMapper

func init() {
	FileMappers = make(map[string]FileMapper)
	FileMappers["ifcb"] = IfcbFileMapper{}
}

func GetFileMapper(fileMapperType string) (FileMapper, error) {
	if len(fileMapperType) == 0 {
		if len(FileMappers) == 1 {
			for k := range FileMappers {
				fileMapperType = k
				break
			}
			fmt.Println("Using type", fileMapperType)
		} else {
			return nil, errors.New("File mapper type was not provided, " + getFileMapperTypes())
		}
	}
	fileMapper, ok := FileMappers[fileMapperType]
	if !ok {
		return nil, errors.New("FileMapper " + fileMapperType + " does not exist, " + getFileMapperTypes())
	}
	return fileMapper, nil
}

func getFileMapperTypes() string {
	types := make([]string, 0)
	for k := range FileMappers {
		types = append(types, k)
	}
	sort.Strings(types)
	return "valid types: " + strings.Join(types, ", ")
}

func processFile(config Config, fileMapper FileMapper, srcFile string) {
	srcDirAbs, _ := filepath.Abs(config.SrcDir)
	relativeSrcFile := strings.TrimPrefix(srcFile, srcDirAbs+"/")
	base := path.Base(srcFile)
	//skip files starting with .  //rsync prepends . to files currently being transferred
	if base[0] == '.' {
		if config.Debug {
			log.Println("file", relativeSrcFile, "starts with ., ignoring")
		}
		return
	}

	relativeDestFile, err := fileMapper.GetFileDestPath(relativeSrcFile)
	if err != nil {
		//TODO copy to unhandled directory?
		if config.Debug {
			log.Println(err)
		}
		return
	}
	destFile := config.DestDir + "/" + relativeDestFile
	destParentDir := path.Dir(destFile)
	os.MkdirAll(destParentDir, os.ModePerm)

	//check for existing file and delete if found
	if _, err := os.Stat(destFile); err == nil {
		if config.Debug {
			log.Println("file", destFile, "exists, deleting")
		}
		if err := os.Remove(destFile); err != nil {
			log.Println("failed to delete file", destFile, err)
			return
		}
	}

	if config.Copy {
		log.Println(srcFile, "--copy-->", destFile)
		err = copyFile(srcFile, destFile)
	} else {
		log.Println(srcFile, "--link-->", destFile)
		err = os.Link(srcFile, destFile)
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

func Watch(config Config, fileMapper FileMapper) {
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
			processFile(config, fileMapper, eventInfo.Path())
		}
	}
}

func Process(config Config, fileMapper FileMapper) {
	if _, err := os.Stat(config.SrcDir); err != nil {
		log.Fatalf("Source directory " + config.SrcDir + " does not exist")
	}

	err := filepath.WalkDir(config.SrcDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			//skip directories
			return nil
		}
		processFile(config, fileMapper, path)
		return nil
	})
	if err != nil {
		log.Fatalf("Could not walk source directory " + config.SrcDir + ", " + err.Error())
	}
}
