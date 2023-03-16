package cmd

import (
	"github.com/digtux/laminar/pkg/common"
	"strings"
)

// FindDockerImages returns sorted and unique list of all docker images
// Tags are stripped
// the docker registries must be configured
//
//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) FindDockerImages(fileList []string, name string) (result []string) {
	for _, file := range fileList {
		imageHit := d.opsClient.Search(file, name)
		for _, img := range imageHit {
			// we don't want trailing @sha256 fields or :tag values, just the image name
			// run a split ":" and only take the first field (we don't care about tags here)
			i := strings.Split(img, "@")[0]
			j := strings.Split(i, ":")[0]
			result = append(result, j)
		}
	}

	// remove duplicates
	result = common.UniqueStrings(result)
	return result
}
