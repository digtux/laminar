package cmd

import (
	"fmt"
	"github.com/jinzhu/copier"
	"go.uber.org/zap"
	"io/ioutil"
	"strings"
)

func DoChange(change ChangeRequest, log *zap.SugaredLogger) (result bool) {

	//log.Infow("calling DoChange()",
	//	"change", change,
	//)

	r, stringContents := ReadFile(change.File, log)

	var originalContents string

	err := copier.Copy(&originalContents, &stringContents) // makes a copy of the original string contents
	if err != nil {
		fmt.Println(err)
	}

	// apply changes to stringContents
	stringContents = strings.Replace(string(r), change.Old, change.New, -1)

	// see if it changed
	if originalContents != stringContents {

		// write bytes to disk
		err = ioutil.WriteFile(change.File, []byte(stringContents), 0)
		if err != nil {
			fmt.Println(err)
		}

		if err != nil {
			fmt.Println(err)
		}

		//log.Debugw("DoChange wrote a change to disk",
		//	"change", change,
		//)

		return true
	}

	fmt.Println("no changes detected")
	return false
}
