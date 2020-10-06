package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"go.uber.org/zap"
	"os"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
)

func Pull(registry cfg.GitRepo, log *zap.SugaredLogger) {
	path := GetRepoPath(registry)
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Errorw("error opening repo",
			"registry", registry,
			"ERR", err,
		)
	}

	w, err := r.Worktree()
	if err != nil {
		log.Fatal("Couldn't open git in %v [%v]", path, err)
	}

	auth := getAuth(registry.Key, log)
	log.Debugw("pulling",
		"registry", registry.URL,
		"branch", registry.Branch,
	)
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if fmt.Sprintf("%v", err) == "already up-to-date" {
		log.Debugf("pull success, already up-to-date")
		err = nil
	}
	if err != nil {
		log.Fatal("Couldn't pull.. [%v]", err)
	}
	log.Debugf(GetCommitId(path, log))
}

func GetRepoPath(registry cfg.GitRepo) string {
	return string("/tmp/" + registry.URL + "-" + registry.Branch)
}

// All-In-One method that will do a clone and checkout
func InitialGitCloneAndCheckout(registry cfg.GitRepo, log *zap.SugaredLogger) {
	diskPath := GetRepoPath(registry)
	log.Debugw("Doing initialGitClone",
		"url", registry.URL,
		"branch", registry.Branch,
		"key", registry.Key,
	)

	auth := getAuth(registry.Key, log)

	if common.IsDir(diskPath, log) {
		log.Debugw("previous checkout detected.. purging it",
			"path", diskPath,
		)
		err := os.RemoveAll(diskPath)
		if err != nil {
			log.Fatalw("couldn't remove dir",
				"diskPath", diskPath,
				"error", err,
			)
		}
	}

	r, err := git.PlainClone(diskPath, false, &git.CloneOptions{
		URL:      registry.URL,
		Progress: nil,
		Auth:     auth,
	})
	if err != nil {
		log.Fatalw("unable to clone the git repo",
			"gitRepo", registry.URL,
			"error", err,
		)
		defer os.Exit(0)
	}

	opts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	}

	if err := r.Fetch(opts); err != nil {
		log.Fatalw("Error fetching remotes",
			"error", err,
		)
	}

	rev , err := r.ResolveRevision(plumbing.Revision(registry.Branch))
	if err != nil {
		log.Fatalw("Error resolving branch",
			"branch", registry.Branch,
			"error", err,
			)
		return
	}
	log.Infow("calculated git revision",
		"branch", registry.Branch,
		"rev", rev,
	)

	w, err := r.Worktree()
	if err != nil{
		log.Fatalw("unable to get Worktree of the repo",
			"error", err,
		)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash:  *rev,
		Force: true,
	})

	if err != nil{
		log.Fatalw("Error checking out branch",
			"error", err,
		)
	}





	//else {
	//	log.Infof("InitialCheckout to %v success", diskPath)
	//}
}

func GetCommitId(path string, log *zap.SugaredLogger) string {
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Fatal(err)
	}

	ref, err := r.Head()
	if err != nil {
		log.Fatal(err)
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		log.Fatal(err)
	}
	return fmt.Sprint(commit.Hash)
}
