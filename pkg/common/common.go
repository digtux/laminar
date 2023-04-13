package common

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/digtux/laminar/pkg/logger"
)

// Misc helper functions

// GetFileAbsPath will expand on something such as ~/.ssh/my_id_rsa and return a string like /home/user/.ssh/my_id_rsa
func GetFileAbsPath(fileName string) (result string) {
	if strings.HasPrefix(fileName, "~/") {
		usr, _ := user.Current()
		dir := usr.HomeDir
		fileName = filepath.Join(dir, fileName[2:])
	}

	result, err := filepath.Abs(fileName)
	if err != nil {
		logger.Fatalw("unable to determine path to a operations",
			"fileName", fileName,
			"error", err,
		)
	}

	return result
}

// IsDir will return true if the path is a directory
func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		logger.Debugw("couldn't reading path",
			"path", path,
			"error", err,
		)
		return false
	}
	return fileInfo.IsDir()
}

// UniqueStrings takes an array of strings in, returns only the unique ones
func UniqueStrings(input []string) []string {
	// credit : https://kylewbanks.com/blog/creating-unique-slices-in-go
	u := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u
}
