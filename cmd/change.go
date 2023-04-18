package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/digtux/laminar/pkg/logger"
	"github.com/jinzhu/copier"
)

func DoChange(change ChangeRequest) (result bool) {
	r, stringContents := readFile(change.File)

	var originalContents string

	err := copier.Copy(&originalContents, &stringContents) // makes a copy of the original string contents
	if err != nil {
		fmt.Println(err)
	}

	// assemble the full strings of the images
	oldString := fmt.Sprintf("%s:%s", change.Image, change.Old)
	newString := fmt.Sprintf("%s:%s", change.Image, change.New)

	logger.Debugw("Doing change",
		"old", oldString,
		"new", newString,
	)

	// apply changes to stringContents
	stringContents = strings.ReplaceAll(string(r), oldString, newString)

	// see if it changed
	if originalContents != stringContents {
		// write bytes to disk
		err = os.WriteFile(change.File, []byte(stringContents), 0)
		if err != nil {
			fmt.Println(err)
		}

		return true
	}

	logger.Infow("no changes detected")
	return false
}
