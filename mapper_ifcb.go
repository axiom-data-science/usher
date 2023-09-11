package usher

import (
	"errors"
	"log"
	"time"
)

type IfcbFileMapper struct{}

func NewIfcbFileMapper() *IfcbFileMapper {
	return &IfcbFileMapper{}
}

func (fm *IfcbFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	baseSrcFile string, mappedRootSrcPath string, mappedRootDestPath string,
	relToMappedRootSrcFile string) (string, error) {
	//D20230525T192231_IFCB162.adc
	if baseSrcFile[0] != 'D' {
		return "", errors.New("file " + relSrcFile + " does not start with D prefix, ignoring")
	} else if len(baseSrcFile) < 16 {
		return "", errors.New("file " + relSrcFile + " has a base filename less than 16 characters, ignoring")
	}

	fileTime, err := time.Parse("20060102T150405", baseSrcFile[1:16])
	if err != nil {
		log.Println(err)
		return "", errors.New("couldn't parse date from file " + relSrcFile + ", ignoring")
	}
	destPath := fileTime.Format("2006/D20060102/") + baseSrcFile
	return destPath, nil
}
