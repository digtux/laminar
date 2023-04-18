package gitoperations

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/logger"
)

func (c *Client) Pull(registry cfg.GitRepo) {
	path := GetRepoPath(registry)
	r, err := git.PlainOpen(path)
	if err != nil {
		logger.Errorw("error opening repo",
			"registry", registry,
			"error", err,
		)
	}

	w, err := r.Worktree()
	// w, err := stuff.Worktree()
	if err != nil {
		logger.Fatal("Couldn't open git in %v [%v]", path, err)
	}

	// auth := getAuth(registry.Key, log)
	logger.Debugw("pulling",
		"registry", registry.URL,
		"branch", registry.Branch,
	)
	err = w.Pull(&git.PullOptions{
		RemoteName:    "origin",
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName(registry.Branch),
	})
	// TODO: replace with err.Error() and check if functions the same
	if fmt.Sprintf("%v", err) == "already up-to-date" {
		logger.Debugf("pull success, already up-to-date")
		err = nil
	}
	if err != nil {
		logger.Errorf("Couldn't pull.. [%v]", err)
		os.Exit(1)
	}
	logger.Debugf(c.GetCommitID(path))
}

func GetRepoPath(registry cfg.GitRepo) string {
	replacedSlash := strings.ReplaceAll(registry.Branch, "/", "-")
	replacedColon := strings.ReplaceAll(replacedSlash, ":", "-")
	return "/tmp/" + registry.URL + "-" + replacedColon
}

// InitialGitCloneAndCheckout All-In-One method that will do a clone and checkout
func (c *Client) InitialGitCloneAndCheckout(registry cfg.GitRepo) *git.Repository {
	diskPath := GetRepoPath(registry)
	logger.Debugw("Doing initialGitClone",
		"url", registry.URL,
		"branch", registry.Branch,
		"key", registry.Key,
	)

	authMethod := c.getAuth(registry.Key)

	if common.IsDir(diskPath) {
		logger.Debugw("previous checkout detected.. purging it",
			"path", diskPath,
		)
		err := os.RemoveAll(diskPath)
		if err != nil {
			logger.Fatalw("couldn't remove dir",
				"diskPath", diskPath,
				"error", err,
			)
		}
	}
	var branchRef = plumbing.NewBranchReferenceName(registry.Branch)

	r, err := git.PlainClone(diskPath, false, &git.CloneOptions{
		URL:           registry.URL,
		Progress:      nil,
		Auth:          authMethod,
		SingleBranch:  true,
		NoCheckout:    false,
		ReferenceName: branchRef,
	})
	if err != nil {
		logger.Fatalw("unable to clone the git repo",
			"gitRepo", registry.URL,
			"error", err,
		)
		defer os.Exit(0)
	}

	opts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*"},
	}

	if err := r.Fetch(opts); err != nil {
		acceptableError := errors.New("already up-to-date")
		if err.Error() != acceptableError.Error() {
			logger.Fatalw("Error fetching remotes",
				"error", err,
			)
		}
	}

	w, err := r.Worktree()
	if err != nil {
		logger.Fatalw("unable to get Worktree of the repo",
			"error", err,
		)
	}

	err = w.Checkout(&git.CheckoutOptions{
		// Hash:  *rev,
		// Force:  true,
		Branch: branchRef,
	})

	if err != nil {
		logger.Fatalw("Error checking out branch",
			"error", err,
		)
	}

	return r
}

func (c *Client) GetCommitID(path string) string {
	r, err := git.PlainOpen(path)
	if err != nil {
		logger.Fatal(err)
	}

	ref, err := r.Head()
	if err != nil {
		logger.Fatal(err)
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		logger.Fatal(err)
	}
	return fmt.Sprint(commit.Hash)
}
