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
	basercFile string, mappedRootSrcPath string, mappedRootDestPath string,
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
