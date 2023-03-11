package cfg

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"go.uber.org/zap"
	"gopkg.in/yaml.v1"
)

// LoadFile will return a Config from a file (string)
func LoadFile(fileName string, log *zap.SugaredLogger) (bytes []byte, err error) {
	log.Debugw("reading file",
		"fileName", fileName)
	rawYaml, err := os.ReadFile(fileName)
	if err != nil {
		log.Warnw("couldn't read file",
			"laminar.fileName", fileName,
		)
		return nil, err
	}
	return rawYaml, err
}

// ParseConfig will read a config and infer some defaults if they're omitted (one day)
func ParseConfig(data []byte) (Config, error) {
	var yamlConfig Config
	var empty Config

	_ = yaml.Unmarshal(data, &yamlConfig)

	// lets return an error if an yaml.Unmarshal returned no new data
	if reflect.DeepEqual(yamlConfig, empty) {
		err := errors.New("no data was loaded")
		return yamlConfig, err
	}
	return yamlConfig, nil
}

// GetUpdatesFromGit will check for a .laminar.yaml in the top level of a git repo
// and attempt to return []Updates from there
func GetUpdatesFromGit(path string, log *zap.SugaredLogger) (updates RemoteUpdates, err error) {
	// construct what we would expect the .laminar.yaml file to be in the git repo
	file := fmt.Sprintf(path + "/" + ".laminar.yaml")

	// try to read the file
	rawFile, err := LoadFile(file, log)

	if err != nil {
		log.Errorw("Error loading file",
			"laminar.file", path,
			"laminar.error", err,
		)
	}

	// try to extract what we expect from the file
	updates, err = ParseUpdates(rawFile, log)
	if err != nil {
		log.Warnw("Reading updates from remote Repo failed",
			"laminar.file", path,
			"laminar.error", err,
		)
		return RemoteUpdates{}, err
	}
	return updates, err
}

// ParseUpdates will read the .laminar.yaml from a repo and return its RemoteUpdates
func ParseUpdates(data []byte, log *zap.SugaredLogger) (RemoteUpdates, error) {
	var yamlUpdates RemoteUpdates

	err := yaml.Unmarshal(data, &yamlUpdates)
	if err != nil {
		log.Warnw("yaml.Unmarshal error",
			"laminar.error", err,
		)
		return RemoteUpdates{}, err
	}
	return yamlUpdates, err
}
