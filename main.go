package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Globals          `yaml:",inline"`
	RootPathMappings map[string]string
	FileMapper       FileMapper `yaml:"-"`
	Watch            WatchConfig
}

func (c *Config) Print() {
	yamlBytes, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(yamlBytes))
}

func (c *Config) UnmarshalConfigFile(configPath string) {
	if len(configPath) == 0 {
		return
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(configBytes, &c)
	if err != nil {
		panic(err)
	}

	return
}

func (c *Config) ApplyCliConfig(globals *Globals, dirArgs *DirArgs) {
	if globals.Debug {
		c.Debug = globals.Debug
	}

	if globals.Copy {
		c.Copy = globals.Copy
	}

	if len(globals.FileMapperRef) > 0 {
		c.FileMapperRef = globals.FileMapperRef
	}

	if len(dirArgs.SrcDir) > 0 {
		c.SrcDir = dirArgs.SrcDir
	}

	if len(dirArgs.DestDir) > 0 {
		c.DestDir = dirArgs.DestDir
	}
}

func GetConfig(globals *Globals, dirArgs *DirArgs) Config {
	config := Config{}
	config.UnmarshalConfigFile(globals.ConfigPath)
	config.ApplyCliConfig(globals, dirArgs)
	fileMapper, err := GetFileMapper(config.FileMapperRef)
	if err != nil {
		panic(err)
	}
	config.FileMapper = fileMapper

	if config.Debug {
		config.Print()
	}

	return config
}

type Globals struct {
	ConfigPath    string `help:"Path to configuration yaml file." name:"config" type:"existingfile" yaml:"-"`
	Debug         bool   `help:"Enable debug mode." yaml:"debug"`
	Copy          bool   `help:"Copy files to destination instead of using hard links."`
	FileMapperRef string `help:"FileMapper type or external executable to map files from src to dest." name:"mapper"`
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

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}))
	err := ctx.Run(&cli.Globals)
	ctx.FatalIfErrorf(err)
}
