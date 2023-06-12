package main

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
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

func processFile(debug bool, copy bool, fileMapper FileMapper, srcDir string, destDir string, srcFile string) {
	srcDirAbs, _ := filepath.Abs(srcDir)
	relativeSrcFile := strings.TrimPrefix(srcFile, srcDirAbs+"/")
	base := path.Base(srcFile)
	//skip files starting with .  //rsync prepends . to files currently being transferred
	if base[0] == '.' {
		if debug {
			log.Println("file", relativeSrcFile, "starts with ., ignoring")
		}
		return
	}

	relativeDestFile, err := fileMapper.GetFileDestPath(relativeSrcFile)
	if err != nil {
		//TODO copy to unhandled directory?
		if debug {
			log.Println(err)
		}
		return
	}
	destFile := destDir + "/" + relativeDestFile
	destParentDir := path.Dir(destFile)
	os.MkdirAll(destParentDir, os.ModePerm)

	//check for existing file and delete if found
	if _, err := os.Stat(destFile); err == nil {
		if debug {
			log.Println("file", destFile, "exists, deleting")
		}
		if err := os.Remove(destFile); err != nil {
			log.Println("failed to delete file", destFile, err)
			return
		}
	}

	if copy {
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

func Watch(debug bool, copy bool, fileMapper FileMapper, srcDir string, destDir string) {
	c := make(chan notify.EventInfo, 1)

	os.MkdirAll(srcDir, os.ModePerm)

	if err := notify.Watch(srcDir+"/...", c, notify.All); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	for eventInfo := range c {
		if eventInfo.Event() == notify.Write || eventInfo.Event() == notify.Rename {
			processFile(debug, copy, fileMapper, srcDir, destDir, eventInfo.Path())
		}
	}
}

func Process(debug bool, copy bool, fileMapper FileMapper, srcDir string, destDir string) {
	if _, err := os.Stat(srcDir); err != nil {
		log.Fatalf("Source directory " + srcDir + " does not exist")
	}

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			//skip directories
			return nil
		}
		processFile(debug, copy, fileMapper, srcDir, destDir, path)
		return nil
	})
	if err != nil {
		log.Fatalf("Could not walk source directory " + srcDir + ", " + err.Error())
	}
}
