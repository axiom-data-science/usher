package usher

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Globals struct {
	ConfigPath    string `help:"Path to configuration yaml file." name:"config" type:"existingfile" yaml:"-"`
	Debug         bool   `help:"Enable debug mode."`
	DryRun        bool   `help:"Show actions that would be performed without doing anything."`
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

	configBytes, err := os.ReadFile(configPath)
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
	if len(globals.ConfigPath) == 0 {
		//use default config.yml if it exists
		if _, err := os.Stat("./config.yml"); err == nil {
			globals.ConfigPath = "./config.yml"
		}
	}
	config.UnmarshalConfigFile(globals.ConfigPath)
	config.ApplyCliConfig(globals, dirArgs)

	fileMapper, err := GetFileMapper(config.FileMapperRef, config.Debug)
	if err != nil {
		log.Fatal(err)
	}
	config.FileMapper = fileMapper

	if config.Debug {
		config.Print()
	}

	return config
}
