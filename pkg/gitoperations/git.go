package gitoperations

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"

	//"github.com/go-git/go-git/v5/plumbing"
	"go.uber.org/zap"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
)

func Pull(registry cfg.GitRepo, log *zap.SugaredLogger) {
	path := GetRepoPath(registry)
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Errorw("error opening repo",
			"laminar.registry", registry,
			"laminar.error", err,
		)
	}

	w, err := r.Worktree()
	//w, err := stuff.Worktree()
	if err != nil {
		log.Fatal("Couldn't open git in %v [%v]", path, err)
	}

	//auth := getAuth(registry.Key, log)
	log.Debugw("pulling",
		"registry", registry.URL,
		"branch", registry.Branch,
	)
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Depth:      1,
	})
	// TODO: replace with err.Error() and check if functions the same
	if fmt.Sprintf("%v", err) == "already up-to-date" {
		log.Debugf("pull success, already up-to-date")
		err = nil
	}
	if err != nil {
		log.Errorf("Couldn't pull.. [%v]", err)
		os.Exit(1)
	}
	log.Debugf(GetCommitId(path, log))
}

func GetRepoPath(registry cfg.GitRepo) string {
	replacedSlash := strings.Replace(registry.Branch, "/", "-", -1)
	replacedColon := strings.Replace(replacedSlash, ":", "-", -1)
	return string("/tmp/" + registry.URL + "-" + replacedColon)
}

// All-In-One method that will do a clone and checkout
func InitialGitCloneAndCheckout(registry cfg.GitRepo, log *zap.SugaredLogger) *git.Repository {
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
	var mergeRef = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", registry.Branch))

	r, err := git.PlainClone(diskPath, false, &git.CloneOptions{
		URL:           registry.URL,
		Progress:      nil,
		Auth:          auth,
		SingleBranch:  true,
		NoCheckout:    false,
		ReferenceName: mergeRef,
	})
	if err != nil {
		log.Fatalw("unable to clone the git repo",
			"gitRepo", registry.URL,
			"error", err,
		)
		defer os.Exit(0)
	}

	opts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
	}

	if err := r.Fetch(opts); err != nil {
		log.Fatalw("Error fetching remotes",
			"error", err,
		)
	}

	w, err := r.Worktree()
	if err != nil {
		log.Fatalw("unable to get Worktree of the repo",
			"error", err,
		)
	}

	err = w.Checkout(&git.CheckoutOptions{
		//Hash:  *rev,
		// Force:  true,
		Branch: mergeRef,
	})

	if err != nil {
		log.Fatalw("Error checking out branch",
			"error", err,
		)
	}

	return r
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
