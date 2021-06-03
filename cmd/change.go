package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"
)

func DoChange(change ChangeRequest, log *zap.SugaredLogger) (result bool) {

	r, stringContents := ReadFile(change.File, log)

	var originalContents string

	err := copier.Copy(&originalContents, &stringContents) // makes a copy of the original string contents
	if err != nil {
		fmt.Println(err)
	}

	// assemble the full strings of the images
	oldString := fmt.Sprintf("%s:%s", change.Image, change.Old)
	newString := fmt.Sprintf("%s:%s", change.Image, change.New)

	log.Debugw("Doing change",
		"old", oldString,
		"new", newString,
	)

	// apply changes to stringContents
	stringContents = strings.Replace(string(r), oldString, newString, -1)

	// see if it changed
	if originalContents != stringContents {

		// write bytes to disk
		err = ioutil.WriteFile(change.File, []byte(stringContents), 0)
		if err != nil {
			fmt.Println(err)
		}

		return true
	}

	fmt.Println("no changes detected")
	return false
}
