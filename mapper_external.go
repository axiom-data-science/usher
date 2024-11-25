package usher

import (
	"errors"
	"log"
	"os/exec"
)

type ExternalFileMapper struct {
	Executable string
}

func NewExternalFileMapper(executable string) *ExternalFileMapper {
	return &ExternalFileMapper{executable}
}

func (fm *ExternalFileMapper) GetFileDestPath(relSrcFile string, absSrcFile string,
	baseSrcFile string, mappedRootSrcPath string, mappedRootDestPath string) (string, error) {
	out, err := exec.Command(fm.Executable, relSrcFile, absSrcFile, baseSrcFile,
		mappedRootSrcPath, mappedRootDestPath).Output()
	if err != nil {
		log.Println(err)
		return "", errors.New("error when processing " + relSrcFile +
			"using external mapper " + fm.Executable + ": " + err.Error())
	}
	return string(out), nil
}
