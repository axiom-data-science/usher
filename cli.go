package usher

import (
	"github.com/alecthomas/kong"
)

type Globals struct {
	ConfigPath    string `help:"Path to configuration yaml file." name:"config" type:"existingfile" yaml:"-"`
	Debug         bool   `help:"Enable debug mode." yaml:"debug"`
	Copy          bool   `help:"Copy files to destination instead of using hard links."`
	FileMapperRef string `help:"FileMapper type or external executable to map files from src to dest." name:"mapper" yaml:"mapper"`
	SrcDir        string `yaml:"src" kong:"-"`
	DestDir       string `yaml:"dest" kong:"-"`
}

type DirArgs struct {
	SrcDir  string `arg:"" help:"Source directory to watch/process." type:"path"`
	DestDir string `arg:"" help:"Destination directory." type:"path"`
}

type WatchConfig struct {
	EventBufferSize int `help:"Size of file event buffer (if buffer is full events are dropped). Default 1000."`
}

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
