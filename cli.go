package usher

import (
	"github.com/alecthomas/kong"
)

type WatchCmd struct {
	WatchConfig
	DirArgs
}

func (c *WatchCmd) Run(globals *Globals) error {
	config := GetConfig(globals, &c.DirArgs)

	if c.EventBufferSize > 0 {
		config.Watch.EventBufferSize = c.EventBufferSize
	} else if config.Watch.EventBufferSize == 0 {
		config.Watch.EventBufferSize = 1000
	}

	Watch(config)
	return nil
}

type ProcessCmd struct {
	DirArgs
}

func (c *ProcessCmd) Run(globals *Globals) error {
	config := GetConfig(globals, &c.DirArgs)
	Process(config)
	return nil
}

var cli struct {
	Globals

	Watch   WatchCmd   `cmd:"" help:"Recursively watch a directory for new files and process when they are detected."`
	Process ProcessCmd `cmd:"" help:"Recursively process all files in a directory and exit."`
}

func Run(fileMappers map[string]FileMapper) {
	SetFileMappers(fileMappers)

	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}))
	err := ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
