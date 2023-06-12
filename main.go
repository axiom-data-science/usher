package main

import (
	"fmt"
	"github.com/alecthomas/kong"
)

type Globals struct {
	Debug bool `help:"Enable debug mode."`
	Copy  bool `help:"Copy files to destination instead of using hard links."`
}

func (g *Globals) Print() {
	fmt.Println("debug", g.Debug)
	fmt.Println("copy", g.Copy)
}

type WatchCmd struct {
	Type     string `help:"File type to process."`
	SrcPath  string `arg:"" help:"Source path to watch." type:"path"`
	DestPath string `arg:"" help:"Destination path." type:"path"`
}

func (c *WatchCmd) Run(globals *Globals) error {
	if globals.Debug {
		fmt.Println("watch")
		fmt.Println("source", c.SrcPath)
		fmt.Println("dest", c.DestPath)
		globals.Print()
	}
	fileMapper, err := GetFileMapper(c.Type)
	if err != nil {
		return err
	}
	Watch(globals.Debug, globals.Copy, fileMapper, c.SrcPath, c.DestPath)
	return nil
}

type ProcessCmd struct {
	Type     string `help:"File type to process."`
	SrcPath  string `arg:"" help:"Source path to process." type:"path"`
	DestPath string `arg:"" help:"Destination path." type:"path"`
}

func (c *ProcessCmd) Run(globals *Globals) error {
	if globals.Debug {
		fmt.Println("process")
		fmt.Println("source", c.SrcPath)
		fmt.Println("dest", c.DestPath)
		globals.Print()
	}
	fileMapper, err := GetFileMapper(c.Type)
	if err != nil {
		return err
	}
	Process(globals.Debug, globals.Copy, fileMapper, c.SrcPath, c.DestPath)
	return nil
}

var cli struct {
	Globals

	Watch   WatchCmd   `cmd:"" help:"Recursively watch a directory for new files and process when they are detected."`
	Process ProcessCmd `cmd:"" help:"Recursively process all files in a directory and exit."`
}

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}))
	err := ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
