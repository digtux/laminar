package cmd

import (
	"fmt"
	"regexp"
)

func nicerMessage(request ChangeRequest) string {
	f := truncateForwardSlash(request.File)
	img := truncateForwardSlash(request.Image)
	tag := truncateTag(request.New)
	msg := fmt.Sprintf("%s: %s:%s", f, img, tag)
	return msg
}

// truncateForwardSlash will convert strings as such:
// '/some/path/filename.txt' > 'filename'
// '/even/http/index.html'   > 'index'
func truncateForwardSlash(input string) string {
	re := regexp.MustCompile(`^(.*/)?(?:$|(.+?)(?:(\.[^.]*$)|$))`)
	match := re.FindStringSubmatch(input)
	if len(match) > 1 {
		return match[2]
	}
	return input

}

func truncateTag(input string) string {

	// only operate if the input is over 50 chars long
	length := len(input)

	if length > 50 {
		// lets get the left side of the string, first 25 chars
		maxLeft := 35
		maxRight := 8

		leftSide := input[0:maxLeft]
		rightSide := input[length-maxRight : length]

		result := fmt.Sprintf("%s...%s", leftSide, rightSide)
		return result
	}
	return input
}
