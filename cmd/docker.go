package cmd

import (
	"github.com/digtux/laminar/pkg/common"
	"strings"
)

// FindDockerImages returns sorted and unique list of all docker images
// Tags are stripped
// the docker registries must be configured
func (d *Daemon) FindDockerImages(fileList []string, name string) (result []string) {
	for _, file := range fileList {
		imageHit := d.opsClient.Search(file, name)
		for _, img := range imageHit {
			// run a split ":" and only take the first field (we don't care about tags here)
			imgWithoutTag := strings.Split(img, ":")[0]
			result = append(result, imgWithoutTag)
		}
	}

	// remove duplicates
	result = common.UniqueStrings(result)
	return result
}
