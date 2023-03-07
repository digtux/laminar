package cmd

import (
	"fmt"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/gitoperations"
)

// UpdateFileList returns a list of files found in the gitoperations repo
// NOTE: updatePolicies with references to the specific gitRepo are required
// NOTE2: if it looks like a folder all files found underneath will be included
// TODO: how does this play with .gitoperations files? we probably want to filter+exclude them here
// TODO: might want to exclude tarballs and other kind of things also I guess..
func (d *Daemon) UpdateFileList(gitRepo cfg.GitRepo) {
	d.logger.Info("updating Daemon's file list")

	// empty the list of files found
	d.fileList = make([]string, 0)

	var thisReposPaths []string

	// now loop though the UpdatePolicies and gather their files[].path values
	// only IF the gitRepo specifies the name of our gitRepo
	for _, update := range gitRepo.Updates {
		for _, p := range update.Files {
			d.logger.Debugw("FileFinder searching for files",
				"path", p.Path,
				"gitRepo", gitRepo.URL,
				"branch", gitRepo.Branch,
			)

			thisReposPaths = append(thisReposPaths, p.Path)
		}
	}
	thisReposPaths = common.UniqueStrings(thisReposPaths)

	// get ready to add the discovered files to the slice
	for _, path := range thisReposPaths {
		// get the path of where the gitoperations repo is checked out
		relativeGitPath := gitoperations.GetRepoPath(gitRepo)
		// combine these
		realPath := fmt.Sprintf("%s/%s", relativeGitPath, path)

		// finally this will return all files found
		for _, fileList := range d.opsClient.FindFiles(realPath) {
			d.fileList = append(d.fileList, fileList)
		}
	}
	d.logger.Debugw("successfully found files in gitoperations",
		"count", len(d.fileList),
	)
}
