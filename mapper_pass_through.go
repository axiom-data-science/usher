package usher

var PassThroughFileMapper = &DelegatingFileMapper{
	GetFileDestPathFunc: func(relSrcFile string, absSrcFile string, baseSrcFile string,
		mappedRootSrcPath string, mappedRootDestPath string) (string, error) {
		return relSrcFile, nil
	},
}
