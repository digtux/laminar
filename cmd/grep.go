package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/logger"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/gobwas/glob"
)

// ChangeRequest is an object recording what changed and when
type ChangeRequest struct {
	Old          string    `json:"old"`
	New          string    `json:"new"`
	Time         time.Time `json:"time"`
	PatternValue string    `json:"patternValue"`
	PatternType  string    `json:"patternType"`
	Image        string    `json:"image"`
	File         string    `json:"file"`
}

func (d *Daemon) doUpdate(filePath string, updates cfg.Updates, registryStrings []string) (changesDone []ChangeRequest) {
	// split the "PatternString" (eg:  `glob:develop-*`) and determine the style
	// TODO: do this check when loading config file
	if len(strings.Split(updates.PatternString, ":")) != 2 {
		logger.Fatalw("pattern string misconfigured.. ",
			"got", updates.PatternString,
			"expected", "'glob:foo-*'   or  'semver:~1.1'   (EG)",
		)
	}

	// slice of potential image strings to operate on
	var potentialUpdatesAll []string
	for _, regString := range registryStrings {
		potentialUpdatesAll = grepFile(filePath, regString)
	}

	// run unique on that string, we'll replace all occurrences
	// TODO: this may not be desirable, check with real-world results
	potentialUpdatesAll = common.UniqueStrings(potentialUpdatesAll)

	patternType := strings.Split(updates.PatternString, ":")[0]
	patternValue := strings.Split(updates.PatternString, ":")[1]

	switch patternType {
	case "glob":
		return d.compileChanges(filePath, potentialUpdatesAll, patternType, patternValue, matchGlob)
	case "regex":
		return d.compileChanges(filePath, potentialUpdatesAll, patternType, patternValue, matchRegex)
	case "semver":
		return d.compileChanges(filePath, potentialUpdatesAll, patternType, patternValue, matchSemver)
	default:
		logger.Fatalf("Support for this pattern type (%s) does not exist yet (sorry)", patternType)
		var changeList []ChangeRequest
		return changeList
	}
}

func (d *Daemon) compileChanges(filePath string, potentialUpdatesAll []string, patternType string, patternValue string, matcher func(string, string) bool) []ChangeRequest {
	var changeList []ChangeRequest
	for _, candidateString := range potentialUpdatesAll {
		// TODO brute force splitting by ":", this will be a problem with registries with additional :123 ports
		// EG.. then the split(":") + len() egg.. if the url is localhost:1234/image:tag
		candidateStringSplit := strings.Split(candidateString, ":")
		if len(candidateStringSplit) < 2 {
			logger.Warnw("Refusing to update image",
				"image", candidateString,
				"file", filePath,
				"info", "expected the format: '<registry>:<tag>",
				"len", len(strings.Split(candidateString, ":")),
			)
		}
		candidateImage := candidateStringSplit[0]

		// this trick will grab the last slice
		candidateTag := candidateStringSplit[len(candidateStringSplit)-1]
		if matcher(candidateTag, patternValue) {
			index := "created"
			tagListFromDB := d.registryClient.CachedImagesToTagInfoListSpecificImage(
				candidateImage,
				index,
			)

			// shouldChange is a bool to assist with logic later
			// changeRequest will go into a []changeList, so we can record it to db one day
			shouldChange, changeRequest := evaluateIfImageShouldChange(
				candidateTag,
				tagListFromDB,
				patternType,
				patternValue,
				candidateImage,
				filePath,
				matcher,
			)
			if shouldChange {
				logger.Infow("newer tag detected",
					"image", changeRequest.File,
					"old", changeRequest.Old,
					"new", changeRequest.New,
				)

				changeHappened := DoChange(changeRequest)
				if changeHappened {
					logger.Debugw("changeList updated with changeRequest",
						"changeRequest", changeRequest)
					changeList = append(changeList, changeRequest)
				}
			}
		} else {
			logger.Debugw("Failed to Match",
				"candidateImage", candidateImage,
				"candidateTag", candidateTag,
				"patternType", patternType,
				"pattern", patternValue,
			)
		}
	}

	// All changes that occurred will be in this slice
	// TODO later: record to DB/cache
	if len(changeList) == 0 {
		logger.Debugw("no changes done",
			"changeList", changeList,
			"filePath", filePath,
		)
	} else {
		logger.Infow("changes to git have happened",
			"changeList", changeList,
			"filePath", filePath,
		)
	}
	return changeList
}

// evaluateIfImageShouldChange checks if a currentTag should be updated
// required:
// - currentTag (string)
// - []TagInfo list of tags from cache (candidates to be promoted)
// - glob-string (all tags must match)
// returns (intent bool, struct ChangeRecord{})
// ChangeRecord? we can record the ChangeRecord in the DB for potential "undo" button
// this function will evaluate "created" timestamps to ensure it has the latest match glob tag
// NOTE: if the currentTag is not indexed in the cache (eg the tag disappeared from docker registry)
//
//	it will assume that **every tag is more recent than the current one**
func evaluateIfImageShouldChange(
	currentTag string,
	cachedTagList []registry.TagInfo,
	patternType string,
	patternValue string,
	image string,
	file string,
	matcher func(string, string) bool,
) (
	intent bool,
	cr ChangeRequest,
) {
	// first lets just be 100% that the currentTag matches
	if matcher(currentTag, patternValue) {
		// This list assumes descending by time, so the first glob match (if any) will be correct
		for _, potentialTag := range cachedTagList {
			// check tag matches. also "latest" is forbidden
			if matcher(
				potentialTag.Tag,
				patternValue) && potentialTag.Tag != "latest" { // exclude identical tags from git+registry
				if potentialTag.Tag != currentTag {
					cr = ChangeRequest{
						Old:          currentTag,
						New:          potentialTag.Tag,
						Time:         time.Now(),
						PatternType:  patternType,
						PatternValue: patternValue,
						Image:        image,
						File:         file,
					}
					return true, cr
				}
				return false, cr
			}
		}

		return false, cr
	}
	logger.Warnw(fmt.Sprintf("sorry, %s doesn't match", patternType),
		"currentTag", currentTag,
		"patternValue", patternValue,
	)
	return false, cr
}

// matchGlob accepts the input string and then the glob string, returns "if match" bool
func matchGlob(input string, globString string) bool {
	g := glob.MustCompile(globString)
	return g.Match(input)
}

// matchRegex accepts a regex pattern then an input string
func matchRegex(input string, regexPattern string) bool {
	match, err := regexp.MatchString(regexPattern, input)
	if err != nil {
		logger.Fatal("bad regex supplied or something? I have no idea sorry")
	}

	logger.Debugw("comparing regex:",
		"input", input,
		"pattern", regexPattern,
		"result", match,
	)
	return match
}

// matchSemver accepts a semverConstraint then an input string
func matchSemver(input string, semverConstraint string) bool {
	c, err := semver.NewConstraint(semverConstraint)

	if err != nil {
		logger.Warnf("Bad semver constraint '%s' supplied", semverConstraint)
		return false
	}

	v, err := semver.NewVersion(input)

	if err != nil {
		logger.Warnf("Version '%s' is not a valid semver version", input)
		return false
	}

	match := c.Check(v)

	logger.Debugw("comparing semver:",
		"input", input,
		"pattern", semverConstraint,
		"result", match,
	)

	return match
}

// readFile will return the raw []byte content of a file
func readFile(filePath string) ([]byte, string) {
	r, err := os.ReadFile(filePath)
	stringContents := string(r)
	if err != nil {
		logger.Fatal(err)
	}
	return r, stringContents
}

// grepFile returns a slice of hits that match a string inside a file
// The assumption is that this is only used against YAML files
func grepFile(file string, searchString string) (matches []string) {
	pat := []byte(searchString)
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(f)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			// if this matches we know the string is somewhere **within a line of text**
			// we should split that line of text (strings.Fields) and range over those to ensure that we
			// don't count the entire line as the actual hit
			// This should be enough for yaml (although I imagine it would also detect stuff in comments)
			// but it would be madness for a json file for example.
			for _, field := range strings.Fields(scanner.Text()) {
				if bytes.Contains([]byte(field), pat) {
					// val := strings.Fields(scanner.Text())[1]
					matches = append(matches, field)
					logger.Debug(field)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Error(err)
	}
	if len(matches) > 0 {
		logger.Debugw("found some images that may need updating",
			"count", len(matches),
			"file", file,
			"searchString", searchString,
		)
	} else {
		logger.Debugw("grepFile found no matches",
			"searchString", searchString,
			"file", file,
			"matches", matches,
		)
	}
	return matches
}
