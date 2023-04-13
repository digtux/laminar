package cfg

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/creasty/defaults"
	"github.com/digtux/laminar/pkg/logger"
	"gopkg.in/yaml.v1"
)

// LoadFile will return a Config from a file (string)
func LoadFile(fileName string) (bytes []byte, err error) {
	logger.Debugw("reading file",
		"fileName", fileName)
	rawYaml, err := os.ReadFile(fileName)
	if err != nil {
		logger.Warnw("couldn't read file",
			"laminar.fileName", fileName,
		)
		return nil, err
	}
	return rawYaml, err
}

// ParseConfig will read a config and infer some defaults if they're omitted (one day)
func ParseConfig(data []byte) (Config, error) {
	var empty Config
	// we will shove data into the "yamlConfig" obj from disk
	yamlConfig := Config{}

	err := yaml.Unmarshal(data, &yamlConfig)
	if err != nil {
		return empty, err
	}
	// defaults (https://github.com/creasty/defaults)
	// TODO: yes I know we should use viper/cobra
	if err := defaults.Set(&yamlConfig); err != nil {
		return empty, err
	}
	if err := defaults.Set(&empty); err != nil {
		return empty, err
	}

	// return an error if an yaml.Unmarshal returned no new data
	if reflect.DeepEqual(yamlConfig, empty) {
		err := errors.New("no data was loaded")
		return yamlConfig, err
	}
	return yamlConfig, nil
}

// GetUpdatesFromGit will check for a .laminar.yaml in the top level of a git repo
// and attempt to return []Updates from there
func GetUpdatesFromGit(path string) (updates RemoteUpdates, err error) {
	// construct what we would expect the .laminar.yaml file to be in the git repo
	file := fmt.Sprintf(path + "/" + ".laminar.yaml")

	// try to read the file
	rawFile, err := LoadFile(file)

	if err != nil {
		logger.Errorw("Error loading file",
			"laminar.file", path,
			"laminar.error", err,
		)
	}

	// try to extract what we expect from the file
	updates, err = ParseUpdates(rawFile)
	if err != nil {
		logger.Warnw("Reading updates from remote Repo failed",
			"laminar.file", path,
			"laminar.error", err,
		)
		return RemoteUpdates{}, err
	}
	return updates, err
}

// ParseUpdates will read the .laminar.yaml from a repo and return its RemoteUpdates
func ParseUpdates(data []byte) (RemoteUpdates, error) {
	var yamlUpdates RemoteUpdates

	err := yaml.Unmarshal(data, &yamlUpdates)
	if err != nil {
		logger.Warnw("yaml.Unmarshal error",
			"laminar.error", err,
		)
		return RemoteUpdates{}, err
	}
	return yamlUpdates, err
}
