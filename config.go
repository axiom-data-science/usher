package usher

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
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
