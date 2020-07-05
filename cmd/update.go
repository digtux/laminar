package cmd

//
//// FileFinder returns a list of files found in the git repo
//func UpdateFinder(cfg config.Config, gitRepo config.GitRepo) []string {
//
//	// empty the list of files found
//	fileList = nil
//
//	var thisReposPaths []string
//
//	// now loop though the UpdatePolicies and gather their files[].path values
//	// only IF the gitRepo specifies the name of our gitRepo
//	for _, tempUpdatePolicy := range cfg.UpdatePolicy{
//
//		for _, configuredRepo := range tempUpdatePolicy.GitRepos{
//
//			if configuredRepo == gitRepo.Name {
//
//				log.Debugw("Good: update policy is pointing to a configured gitRepo",
//					"updatePolicy", tempUpdatePolicy,
//					"gitRepo", gitRepo,bit branch
//				)
//				for _, p := range tempUpdatePolicy.Files{
//					thisReposPaths = append(thisReposPaths, p.Path)
//				}
//			}
//		}
//	}
//	thisReposPaths = common.UniqueStrings(thisReposPaths)
//
//	// get ready to add the discovered files to the slice
//	for _, p := range thisReposPaths{
//		// get the path of where the git repo is checked out
//		relativeGitPath := git.GetRepoPath(gitRepo)
//		// combine these
//		realPath := fmt.Sprintf("%s/%s", relativeGitPath, p)
//
//		// finally this will return all files found
//		for _, x := range operations.FindFiles(realPath) {
//			fileList = append(fileList, x)
//		}
//
//	}
//	return fileList
//}
