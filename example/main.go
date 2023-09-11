package main

import (
	"axds.co/usher"
)

func main() {
	usher.Run(map[string]usher.FileMapper{
		"mtime_yyyy_mm_dd": usher.NewMtimeFileMapper("2006/01/02/"),
	})
}
