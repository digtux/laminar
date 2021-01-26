package cmd

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/gobwas/glob"
	"github.com/tidwall/buntdb"
	"go.uber.org/zap"
)

// ChangeRequests is an object recording what changed and when
type ChangeRequest struct {
	Old          string    `json:"old"`
	New          string    `json:"new"`
	Time         time.Time `json:"time"`
	PatternValue string    `json:"patternValue"`
	PatternType  string    `json:"patternType"`
	Image        string    `json:"image"`
	File         string    `json:"file"`
}

func DoUpdate(
	filePath string,
	updates cfg.Updates,
	registryStrings []string,
	db *buntdb.DB,
	log *zap.SugaredLogger,
) (changesDone int) {

	// split the "PatternString" (eg:  `glob:develop-*`) and determine the style
	// TODO: do this check when loading config file
	if len(strings.Split(updates.PatternString, ":")) != 2 {
		log.Fatalw("pattern string misconfigured.. ",
			"got", updates.PatternString,
			"expected", "'glob:foo-*'   or  'semver:~1.1'   (EG)",
		)
	}

	// slice of potential image strings to operate on
	potentialUpdatesAll := []string{}
	for _, regString := range registryStrings {
		potentialUpdatesAll = grepFile(filePath, regString, log)
	}

	// run unique on that string, we'll replace all occurences
	// TODO: this may not be desirable, check with realworld results
	potentialUpdatesAll = common.UniqueStrings(potentialUpdatesAll)

	patternType := strings.Split(updates.PatternString, ":")[0]
	patternValue := strings.Split(updates.PatternString, ":")[1]

	switch patternType {
	case "glob":

		var changeList = []ChangeRequest{}
		// TODO.. whole glob case section to its own function/package
		//log.Infof("searching file %s as type glob", filePath)
		for _, candidateString := range potentialUpdatesAll {

			// TODO brute force splitting by ":", this will be a problem with registries with additional :123 ports
			// EG.. then the split(":") + len() egg.. if the url is localhost:1234/image:tag
			candidateStringSplit := strings.Split(candidateString, ":")
			if len(candidateStringSplit) < 2 {
				log.Warnw("Refusing to update image",
					"image", candidateString,
					"file", filePath,
					"info", "expected the format: '<registry>:<tag>",
					"len", len(strings.Split(candidateString, ":")),
				)
			}
			candidateImage := candidateStringSplit[0]

			// this trick will grab the last slice
			candidateTag := candidateStringSplit[len(candidateStringSplit)-1]

			if MatchGlob(candidateTag, patternValue) {

				// log.Debugw("Matched Globs",
				// 	"candidateImage", candidateImage,
				// 	"candidateTag", candidateTag,
				// 	"pattern", patternValue,
				// 	"type", patternType,
				// )

				// get a full list of tags for the image from our cache
				index := "created"
				tagListFromDb := registry.CachedImagesToTagInfoListSpecificImage(
					db,
					candidateImage,
					index,
					log,
				)

				// shouldChange is a bool to assist with logic later
				// changeRequest will go into a []changeList so we can record it to db one day
				shouldChange, changeRequest := EvaluateIfImageShouldChangeGlob(
					candidateTag,
					tagListFromDb,
					patternValue,
					candidateImage,
					filePath,
					log,
				)
				if shouldChange {
					log.Infow("newer tag detected",
						"image", changeRequest.File,
						"old", changeRequest.Old,
						"new", changeRequest.New,
					)

					changeHappened := DoChange(changeRequest, log)
					if changeHappened {
						log.Debugw("changeList updated with changeRequest",
							"changeRequest", changeRequest)
						changeList = append(changeList, changeRequest)
					}
				}

			} else {
				log.Debugw("Failed to Match Globs",
					"candidateImage", candidateImage,
					"candidateTag", candidateTag,
					"pattern", patternValue,
					"type", patternType,
				)
			}
		}

		// All changes that occured will be in this slice
		// TODO later: record to DB/cache
		if len(changeList) == 0 {
			log.Infow("no changes done",
				"changeList", changeList,
				"filePath", filePath,
			)
		} else {
			log.Infow("changes to git have happened",
				"changeList", changeList,
				"filePath", filePath,
			)
		}
		return len(changeList)

	case "regex":
		var changeList = []ChangeRequest{}
		//log.Infof("searching file %s as type glob", filePath)
		for _, candidateString := range potentialUpdatesAll {

			// TODO brute force splitting by ":", this will be a problem with registries with additional :123 ports
			// EG.. then the split(":") + len() egg.. if the url is localhost:1234/image:tag
			candidateStringSplit := strings.Split(candidateString, ":")
			if len(candidateStringSplit) < 2 {
				log.Warnw("Refusing to update image",
					"image", candidateString,
					"file", filePath,
					"info", "expected the format: '<registry>:<tag>",
					"len", len(strings.Split(candidateString, ":")),
				)
			}
			candidateImage := candidateStringSplit[0]

			// this trick will grab the last slice
			candidateTag := candidateStringSplit[len(candidateStringSplit)-1]

			if MatchRegex(
				candidateTag,
				patternValue,
				log) {

				index := "created"
				tagListFromDb := registry.CachedImagesToTagInfoListSpecificImage(
					db,
					candidateImage,
					index,
					log,
				)

				// shouldChange is a bool to assist with logic later
				// changeRequest will go into a []changeList so we can record it to db one day
				shouldChange, changeRequest := EvaluateIfImageShouldChangeRegex(
					candidateTag,
					tagListFromDb,
					patternValue,
					candidateImage,
					filePath,
					log,
				)
				if shouldChange {
					log.Infow("newer tag detected",
						"image", changeRequest.File,
						"old", changeRequest.Old,
						"new", changeRequest.New,
					)

					changeHappened := DoChange(changeRequest, log)
					if changeHappened {
						log.Debugw("changeList updated with changeRequest",
							"changeRequest", changeRequest)
						changeList = append(changeList, changeRequest)
					}
				}

			} else {
				log.Debugw("Failed to Match Regex",
					"candidateImage", candidateImage,
					"candidateTag", candidateTag,
					"pattern", patternValue,
					"type", patternType,
				)
			}
		}

		// All changes that occured will be in this slice
		// TODO later: record to DB/cache
		if len(changeList) == 0 {
			log.Infow("no changes done",
				"changeList", changeList,
				"filePath", filePath,
			)
		} else {
			log.Infow("changes to git have happened",
				"changeList", changeList,
				"filePath", filePath,
			)
		}
		return len(changeList)

	default:
		log.Fatalf("Support for this pattern type (%s) does not exist yet (sorry)", patternType)
		return 0

	}
}

// EvaluateIfImageShouldChangeGlob checks if a currentTag should be updated
// required:
// - currentTag (string)
// - []TagInfo list of tags from cache (candidates to be promoted)
// - glob-string (all tags must match)
// returns (dochange bool, struct ChangeRecord{})
// ChangeRecord? we can record the ChangeRecord in the DB for potential "undo" button oneday
// this function will evaulate "created" timestamps to ensure it has the latest match glob tag
// NOTE: if the currentTag is not indexed in the cache (eg the tag disappeared from docker registry)
//       it will assume that **every tag is more recent than the current one**
func EvaluateIfImageShouldChangeGlob(
	currentTag string,
	cachedTagList []registry.TagInfo,
	patternValue string,
	image string,
	file string,
	log *zap.SugaredLogger,
) (
	intent bool,
	cr ChangeRequest,
) {
	// first lets just be 100% that the currentTag matches the glob
	if MatchGlob(currentTag, patternValue) {
		// This list assumes descending by time, so the first glob match (if any) will be correct
		for _, potentialTag := range cachedTagList {
			// check tag matches.. also "latest" is fobidden
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
	log.Warnw("sorry, glob doesn't match",
		"currentTag", currentTag,
		"patternValue", patternValue,
	)
	return false, cr

}

// MatchGlob accepts the input string and then the glob string, returns "if match" bool
func MatchGlob(input string, globString string) bool {
	var g glob.Glob
	g = glob.MustCompile(globString)
	return g.Match(input)
}

// MatchRegex accepts a regex pattern then an input string
func MatchRegex(input string, regexPattern string, log *zap.SugaredLogger) bool {
	match, err := regexp.MatchString(regexPattern, input)
	if err != nil {
		log.Fatal("bad regex supplied or something? I have no idea sorry")
	}

	log.Debugw("comparing regex:",
		"input", input,
		"pattern", regexPattern,
		"result", match,
	)
	return match
}

// EvaluateIfImageShouldChangeGlob checks if a currentTag should be updated
// required:
// - currentTag (string)
// - []TagInfo list of tags from cache (candidates to be promoted)
// - glob-string (all tags must match)
// returns (dochange bool, struct ChangeRecord{})
// ChangeRecord? we can record the ChangeRecord in the DB for potential "undo" button oneday
// this function will evaulate "created" timestamps to ensure it has the latest match glob tag
// NOTE: if the currentTag is not indexed in the cache (eg the tag disappeared from docker registry)
//       it will assume that **every tag is more recent than the current one**
func EvaluateIfImageShouldChangeRegex(
	currentTag string,
	cachedTagList []registry.TagInfo,
	patternValue string,
	image string,
	file string,
	log *zap.SugaredLogger,
) (
	intent bool,
	cr ChangeRequest,
) {
	// first lets just be 100% that the currentTag matches the glob
	if MatchRegex(
		currentTag,
		patternValue,
		log) {
		// This list assumes descending by time, so the first glob match (if any) will be correct
		for _, potentialTag := range cachedTagList {
			// check tag matches.. also "latest" is fobidden
			if MatchRegex(
				potentialTag.Tag,
				patternValue,
				log) && potentialTag.Tag != "latest" {

				// exclude identical tags from git+registry
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
	log.Warnw("sorry, regex doesn't match",
		"currentTag", currentTag,
		"patternValue", patternValue,
	)
	return false, cr

}

func ReadFile(filePath string, log *zap.SugaredLogger) ([]byte, string) {
	r, err := ioutil.ReadFile(filePath)
	stringContents := string(r)
	if err != nil {
		log.Fatal(err)
	}
	return r, stringContents
}

// grepFile returns a slice of hits that match a string inside a file
// The assumption is that this is only used against YAML files
func grepFile(file string, searchString string, log *zap.SugaredLogger) (matches []string) {
	pat := []byte(searchString)
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), pat) {
			// if this matches we know the string is somewhere **within a line of text**
			// we should split that line of text (strings.Fields) and range over those to ensure that we
			// don't count the entire line as the actual hit
			// This should be enough for yaml (althoug I imagine it would also detect stuff in comments)
			// but it would be madness for a json file for example..
			for _, field := range strings.Fields(scanner.Text()) {
				if bytes.Contains([]byte(field), pat) {
					// val := strings.Fields(scanner.Text())[1]
					matches = append(matches, field)
					log.Debug(field)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}
	if len(matches) > 0 {
		log.Debugw("found some images that may need updating",
			"count", len(matches),
			"file", file,
			"searchString", searchString,
		)
	} else {
		log.Debugw("grepFile found no matches",
			"searchString", searchString,
			"file", file,
			"matches", matches,
		)
	}
	return matches
}
