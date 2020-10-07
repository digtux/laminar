package cfg

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v1"
	"io/ioutil"
)

//LoadFile will return a Config from a file (string)
func LoadFile(fileName string, log *zap.SugaredLogger) (bytes []byte, err error) {
	log.Debugw("reading file",
		"fileName", fileName)
	rawYaml, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Warnw("couldn't read file",
			"fileName", fileName,
		)
		return nil, err
	}
	return rawYaml, err
}

// ParseConfig will read a config and infer some defaults if they're omitted (one day)
func ParseConfig(data []byte, log *zap.SugaredLogger) (Config, error) {

	var yamlConfig Config

	err := yaml.Unmarshal(data, &yamlConfig)
	if err != nil {
		log.Warnw("yaml.Unmarshal error",
			"error", err,
		)
		return Config{}, err
	}

	return yamlConfig, err
}

// GetUpdatesFromGit will check for a .laminar.yaml in the top level of a git repo
// and attempt to return []Updates from there
func GetUpdatesFromGit(path string, log *zap.SugaredLogger) (x RemoteUpdates, err error) {
	// construct what we would expect the .laminar.yaml file to be in the git repo
	file := fmt.Sprintf(path + "/" + ".laminar.yaml")

	// try to read the file
	rawFile, err := LoadFile(file, log)

	if err != nil {
		log.Errorw("Error loading file",
			"file", path,
			"error", err,
		)
	}

	// try to extract what we expect from the file
	x, err = ParseUpdates(rawFile, log)
	if err != nil {
		log.Warnw("Reading updates from remote Repo failed",
			"file", path,
			"error", err,
		)
		return RemoteUpdates{}, err
	}
	return x, err
}

// ParseUpdates will read the .laminar.yaml from a repo and return its RemoteUpdates
func ParseUpdates(data []byte, log *zap.SugaredLogger) (RemoteUpdates, error) {

	var yamlUpdates RemoteUpdates

	err := yaml.Unmarshal(data, &yamlUpdates)
	if err != nil {
		log.Warnw("yaml.Unmarshal error",
			"error", err,
		)
		return RemoteUpdates{}, err
	}
	return yamlUpdates, err
}
