package operations

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/logger"
)

type Client struct {
}

func New() *Client {
	return &Client{}
}

// FindFiles returns a slice containing paths to all files found in a directory
// NOTE: it will ignore .git folders and their contents
func (c *Client) FindFiles(searchPath string) []string {
	// regex patterns to exclude in git repo
	// TODO: allow extending exclude pattern from config
	skippedPatterns := []string{
		".git/.*",
	}

	collectFiles := func(dir string, excludeList []string) (fileList []string, err error) {
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if regexp.MustCompile(strings.Join(excludeList, "|")).Match([]byte(path)) {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			fileList = append(fileList, path)
			return nil
		})
		if err != nil {
			logger.Fatalw("walk error",
				"err", err)
			return nil, err
		}
		return fileList, nil
	}
	targetFiles, err := collectFiles(searchPath, skippedPatterns)
	logger.Debugw("FindFiles",
		"matched", targetFiles) // TODO: probably not great logging ALL the files
	// maybe truncate this to be friendlier, print the len()
	// TODO: check incase len is 0

	if err != nil {
		logger.Debugw("file error",
			"path", searchPath,
			"error", err,
		)
	}

	return targetFiles
}

// Search returns a slice of hits that match a string inside a operations
// The assumption is that this is only used against YAML files
// it should work on other types but YMMV
func (c *Client) Search(file string, searchString string) (matches []string) {
	pat := []byte(searchString)
	fp := common.GetFileAbsPath(file)
	f, err := os.Open(fp)
	if err != nil {
		logger.Fatal(err)
	}
	defer f.Close()

	// start a scanner to search the operations
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			// if this matches we know the string is somewhere **within a line of text**
			// we should split that line of text (strings.Fields) and range over those to ensure that we
			// don't count the entire line as the actual hit
			// This should be enough for yaml (although I imagine it would also detect stuff in comments)
			// but it would be madness for a json operations for example..
			for _, field := range strings.Fields(scanner.Text()) {
				if bytes.Contains([]byte(field), pat) {
					// val := strings.Fields(scanner.Text())[1]
					matches = append(matches, field)
					// log.Debug(scanner.Text())
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error(err)
	}
	// if len(matches) > 0 {
	//	log.Debugw("Search found some matches",
	//		"searchString", searchString,
	//		"operations", file,
	//		"matches", matches,
	//	)
	// } else {
	//	log.Debugw("Search found no matches",
	//		"searchString", searchString,
	//		"operations", file,
	//		"matches", matches,
	//	)
	// }
	return matches
}
