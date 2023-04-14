package cmd

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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
		return d.caseGlob(filePath, potentialUpdatesAll, patternValue)
	case "regex":
		return d.caseRegex(filePath, potentialUpdatesAll, patternValue)
	default:
		logger.Fatalf("Support for this pattern type (%s) does not exist yet (sorry)", patternType)
		var changeList []ChangeRequest
		return changeList
	}
}

func (d *Daemon) caseRegex(filePath string, potentialUpdatesAll []string, patternValue string) []ChangeRequest {
	var changeList []ChangeRequest
	for _, candidateString := range potentialUpdatesAll {
		// TODO brute force splitting by ":", this will be a problem with registries with additional :123 ports
		// EG.. then the split(":") + len() egg.. if the url is localhost:1234/image:tag
		candidateStringSplit := strings.Split(candidateString, ":")
		if len(candidateStringSplit) < 2 {
			logger.Warnw("Refusing to update image",
				"laminar.image", candidateString,
				"laminar.file", filePath,
				"laminar.info", "expected the format: '<registry>:<tag>",
				"laminar.len", len(strings.Split(candidateString, ":")),
			)
		}
		candidateImage := candidateStringSplit[0]

		// this trick will grab the last slice
		candidateTag := candidateStringSplit[len(candidateStringSplit)-1]
		if MatchRegex(candidateTag, patternValue) {
			index := "created"
			tagListFromDB := d.registryClient.CachedImagesToTagInfoListSpecificImage(
				candidateImage,
				index,
			)

			// shouldChange is a bool to assist with logic later
			// changeRequest will go into a []changeList, so we can record it to db one day
			shouldChange, changeRequest := EvaluateIfImageShouldChangeRegex(
				candidateTag,
				tagListFromDB,
				patternValue,
				candidateImage,
				filePath,
			)
			if shouldChange {
				logger.Infow("newer tag detected",
					"laminar.image", changeRequest.File,
					"laminar.old", changeRequest.Old,
					"laminar.new", changeRequest.New,
				)

				changeHappened := DoChange(changeRequest)
				if changeHappened {
					logger.Debugw("changeList updated with changeRequest",
						"laminar.changeRequest", changeRequest)
					changeList = append(changeList, changeRequest)
				}
			}
		} else {
			logger.Debugw("Failed to Match Regex",
				"laminar.candidateImage", candidateImage,
				"laminar.candidateTag", candidateTag,
				"laminar.pattern", patternValue,
			)
		}
	}

	// All changes that occurred will be in this slice
	// TODO later: record to DB/cache
	if len(changeList) == 0 {
		logger.Debugw("no changes done",
			"laminar.changeList", changeList,
			"laminar.filePath", filePath,
		)
	} else {
		logger.Infow("changes to git have happened",
			"laminar.changeList", changeList,
			"laminar.filePath", filePath,
		)
	}
	return changeList
}

func (d *Daemon) caseGlob(filePath string, potentialUpdatesAll []string, patternValue string) []ChangeRequest {
	var changeList []ChangeRequest
	// TODO.. whole glob case section to its own function/package
	for _, candidateString := range potentialUpdatesAll {
		// TODO brute force splitting by ":", this will be a problem with registries with additional :123 ports
		// EG.. then the split(":") + len() egg.. if the url is localhost:1234/image:tag
		candidateStringSplit := strings.Split(candidateString, ":")
		if len(candidateStringSplit) < 2 {
			logger.Warnw("Refusing to update image",
				"laminar.image", candidateString,
				"laminar.file", filePath,
				"laminar.info", "expected the format: '<registry>:<tag>",
				"laminar.len", len(strings.Split(candidateString, ":")),
			)
		}
		candidateImage := candidateStringSplit[0]

		// this trick will grab the last slice
		candidateTag := candidateStringSplit[len(candidateStringSplit)-1]
		if MatchGlob(candidateTag, patternValue) {
			// get a full list of tags for the image from our cache
			index := "created"
			tagListFromDB := d.registryClient.CachedImagesToTagInfoListSpecificImage(
				candidateImage,
				index,
			)

			// shouldChange is a bool to assist with logic later
			// changeRequest will go into a []changeList, so we can record it to db one day
			shouldChange, changeRequest := EvaluateIfImageShouldChangeGlob(
				candidateTag,
				tagListFromDB,
				patternValue,
				candidateImage,
				filePath,
			)
			if shouldChange {
				logger.Infow("newer tag detected",
					"laminar.image", changeRequest.File,
					"laminar.old", changeRequest.Old,
					"laminar.new", changeRequest.New,
				)

				changeHappened := DoChange(changeRequest)
				if changeHappened {
					logger.Debugw("changeList updated with changeRequest",
						"laminar.changeRequest", changeRequest)
					changeList = append(changeList, changeRequest)
				}
			}
		} else {
			logger.Debugw("Failed to Match Globs",
				"laminar.candidateImage", candidateImage,
				"laminar.candidateTag", candidateTag,
				"laminar.pattern", patternValue,
			)
		}
	}

	// All changes that occurred will be in this slice
	// TODO later: record to DB/cache
	if len(changeList) == 0 {
		logger.Debugw("no changes done",
			"laminar.changeList", changeList,
			"laminar.filePath", filePath,
		)
	} else {
		logger.Infow("changes to git have happened",
			"laminar.changeList", changeList,
			"laminar.filePath", filePath,
		)
	}
	return changeList
}

// EvaluateIfImageShouldChangeGlob checks if a currentTag should be updated
// required:
// - currentTag (string)
// - []TagInfo list of tags from cache (candidates to be promoted)
// - glob-string (all tags must match)
// returns (intent bool, struct ChangeRecord{})
// ChangeRecord? we can record the ChangeRecord in the DB for potential "undo" button (One day)
// this function will evaluate "created" timestamps to ensure it has the latest match glob tag
// NOTE: if the currentTag is not indexed in the cache (eg the tag disappeared from docker registry)
//
//	it will assume that **every tag is more recent than the current one**
func EvaluateIfImageShouldChangeGlob(
	currentTag string,
	cachedTagList []registry.TagInfo,
	patternValue string,
	image string,
	file string,
) (
	intent bool,
	cr ChangeRequest,
) {
	// first lets just be 100% that the currentTag matches the glob
	if MatchGlob(currentTag, patternValue) {
		// This list assumes descending by time, so the first glob match (if any) will be correct
		for _, potentialTag := range cachedTagList {
			// check tag matches. also "latest" is forbidden
			if MatchGlob(potentialTag.Tag, patternValue) && potentialTag.Tag != "latest" {
				// exclude identical tags from git+registry
				if potentialTag.Tag != currentTag {
					cr = ChangeRequest{
						Old:          currentTag,
						New:          potentialTag.Tag,
						Time:         time.Now(),
						PatternType:  "glob",
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
	logger.Warnw("sorry, glob doesn't match",
		"laminar.currentTag", currentTag,
		"laminar.patternValue", patternValue,
	)
	return false, cr
}

// MatchGlob accepts the input string and then the glob string, returns "if match" bool
func MatchGlob(input string, globString string) bool {
	g := glob.MustCompile(globString)
	return g.Match(input)
}

// MatchRegex accepts a regex pattern then an input string
func MatchRegex(input string, regexPattern string) bool {
	match, err := regexp.MatchString(regexPattern, input)
	if err != nil {
		logger.Fatal("bad regex supplied or something? I have no idea sorry")
	}

	logger.Debugw("comparing regex:",
		"laminar.input", input,
		"laminar.pattern", regexPattern,
		"laminar.result", match,
	)
	return match
}

// EvaluateIfImageShouldChangeRegex checks if a currentTag should be updated
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
func EvaluateIfImageShouldChangeRegex(
	currentTag string,
	cachedTagList []registry.TagInfo,
	patternValue string,
	image string,
	file string,
) (
	intent bool,
	cr ChangeRequest,
) {
	// first lets just be 100% that the currentTag matches the glob
	if MatchRegex(
		currentTag,
		patternValue) {
		// This list assumes descending by time, so the first glob match (if any) will be correct
		for _, potentialTag := range cachedTagList {
			// check tag matches. also "latest" is forbidden
			if MatchRegex(
				potentialTag.Tag,
				patternValue) && potentialTag.Tag != "latest" { // exclude identical tags from git+registry
				if potentialTag.Tag != currentTag {
					cr = ChangeRequest{
						Old:          currentTag,
						New:          potentialTag.Tag,
						Time:         time.Now(),
						PatternType:  "regex",
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
	logger.Warnw("sorry, regex doesn't match",
		"laminar.currentTag", currentTag,
		"laminar.patternValue", patternValue,
	)
	return false, cr
}

// ReadFile will return the raw []byte content of a file
func ReadFile(filePath string) ([]byte, string) {
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
			"laminar.count", len(matches),
			"laminar.file", file,
			"laminar.searchString", searchString,
		)
	} else {
		logger.Debugw("grepFile found no matches",
			"laminar.searchString", searchString,
			"laminar.file", file,
			"laminar.matches", matches,
		)
	}
	return matches
}
