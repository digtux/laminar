package cmd

import (
	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/operations"
	"go.uber.org/zap"
	"strings"
)

// FindDockerImages returns sorted and unique list of all docker images
// Tags are stripped
// the docker registries must be configured
func FindDockerImages(fileList []string, name string, log *zap.SugaredLogger) (result []string) {

	for _, file := range fileList {
		imageHit := operations.Search(file, name, log)
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
