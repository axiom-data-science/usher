package usher

import (
	"errors"
	"log"
	"os"
	"strings"
)

type MtimeFileMapper struct {
	DatePathFormat string
}

func NewMtimeFileMapper(datePathFormat string) *MtimeFileMapper {
	if len(datePathFormat) > 0 && !strings.HasSuffix(datePathFormat, "/") {
		datePathFormat += "/"
	}
	return &MtimeFileMapper{datePathFormat}
}

func (fm *MtimeFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	baseSrcFile string, mappedRootSrcPath string, mappedRootDestPath string,
	relToMappedRootSrcFile string) (string, error) {
	//stat source file for last modified time
	srcStat, err := os.Stat(absSrcFile)
	if err != nil {
		log.Println(err)
		return "", errors.New("couldn't stat source file " + relSrcFile + ", skipping")
	}

	//set default date pattern if none was set
	if len(fm.DatePathFormat) == 0 {
		fm.DatePathFormat = "2006/01/02/"
	}
	return srcStat.ModTime().Format(fm.DatePathFormat) + baseSrcFile, nil
}
