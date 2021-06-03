package operations

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/digtux/laminar/pkg/common"
	"go.uber.org/zap"
)

// FindFiles returns a slice containing paths to all files found in a directory
func FindFiles(searchPath string, log *zap.SugaredLogger) []string {

	var result []string

	// this function will handle each object inside the Walk()
	var searchFunc = func(pathX string, infoX os.FileInfo, errX error) error {

		// check for errors
		if errX != nil {
			//log.Warnw("FindFiles error",
			//	"path", pathX,
			//	"err", errX,
			//)
			return errX
		}

		if common.IsFile(pathX, log) {
			log.Debugw("FindFiles found file",
				"fileName", infoX.Name(),
			)

			// TODO more expressive way to ignore certain files (in git) that users may want.. eg helm charts
			ext := filepath.Ext(pathX)
			switch ext {
			case ".yml":
				result = append(result, pathX)
			case ".yaml":
				result = append(result, pathX)
			default:
				log.Warnw("file not yaml, ignoring",
					"laminar.path", pathX)
			}
		}

		return nil
	}

	realPath := common.GetFileAbsPath(searchPath, log)
	err := filepath.Walk(realPath, searchFunc)

	if err != nil {
		log.Debugw("file error",
			"path", searchPath,
			"error", err,
		)
	}

	return result
}

// Search returns a slice of hits that match a string inside a operations
// The assumption is that this is only used against YAML files
// it should work on other types but YMMV
func Search(file string, searchString string, log *zap.SugaredLogger) (matches []string) {
	pat := []byte(searchString)
	fp := common.GetFileAbsPath(file, log)
	f, err := os.Open(fp)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// start a scanner to search the operations
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			// if this matches we know the string is somewhere **within a line of text**
			// we should split that line of text (strings.Fields) and range over those to ensure that we
			// don't count the entire line as the actual hit
			// This should be enough for yaml (althoug I imagine it would also detect stuff in comments)
			// but it would be madness for a json operations for example..
			for _, field := range strings.Fields(scanner.Text()) {
				if bytes.Contains([]byte(field), pat) {
					// val := strings.Fields(scanner.Text())[1]
					matches = append(matches, field)
					//log.Debug(scanner.Text())
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}
	//if len(matches) > 0 {
	//	log.Debugw("Search found some matches",
	//		"searchString", searchString,
	//		"operations", file,
	//		"matches", matches,
	//	)
	//} else {
	//	log.Debugw("Search found no matches",
	//		"searchString", searchString,
	//		"operations", file,
	//		"matches", matches,
	//	)
	//}
	return matches
}
