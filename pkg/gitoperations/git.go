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
)

var ErrAlreadyUpToDate = "already up-to-date"
var ErrNonFastForwardUpdate = "non-fast-forward update"

func (c *Client) Pull(registry cfg.GitRepo) {
	path := GetRepoPath(registry)
	branchRefName := plumbing.NewBranchReferenceName(registry.Branch)
	r, err := git.PlainOpen(path)
	if err != nil {
		c.logger.Errorw("error opening repo",
			"laminar.registry", registry,
			"laminar.error", err,
		)
	}

	w, err := r.Worktree()
	if err != nil {
		c.logger.Fatalw("Couldn't open git",
			"error", err,
			"path", path,
		)
	}

	// auth := getAuth(registry.Key, log)
	c.logger.Debugw("pulling",
		"registry", registry.URL,
		"branch", registry.Branch,
	)

	pullOpts := &git.PullOptions{
		ReferenceName: branchRefName,
		// Auth:          auth,
		Force:      true,
		RemoteName: "origin",
		Depth:      1,
	}
	err = w.Pull(pullOpts)

	errMsg := fmt.Sprintf("%v", err)

	// downgrade "OK" errors to warnings
	if err != nil {
		switch {
		// case errMsg == ErrNonFastForwardUpdate:
		// 	c.logger.Warnw("pull warning",
		// 		"error", err,
		// 	)
		// 	err = nil
		case errMsg == ErrAlreadyUpToDate:
			c.logger.Warnw("pull warning",
				"error", err,
			)
			err = nil
		default:
			c.logger.Fatalw("Couldn't pull from git",
				"error", err,
				"errMsg", errMsg,
				"registry", registry,
			)
		}
	}
	c.logger.Infow("Git Pull OK",
		"resgisty", registry,
		"commitID", c.GetCommitID(path),
	)
}

func GetRepoPath(registry cfg.GitRepo) string {
	replacedSlash := strings.ReplaceAll(registry.Branch, "/", "-")
	replacedColon := strings.ReplaceAll(replacedSlash, ":", "-")
	return "/tmp/" + registry.URL + "-" + replacedColon
}

// InitialGitCloneAndCheckout All-In-One method that will do a clone and checkout
func (c *Client) InitialGitCloneAndCheckout(registry cfg.GitRepo) *git.Repository {
	diskPath := GetRepoPath(registry)
	c.logger.Debugw("Doing initialGitClone",
		"url", registry.URL,
		"branch", registry.Branch,
		"key", registry.Key,
	)

	authMethod := c.getAuth(registry.Key)

	if common.IsDir(diskPath, c.logger) {
		c.logger.Debugw("previous checkout detected.. purging it",
			"path", diskPath,
		)
		err := os.RemoveAll(diskPath)
		if err != nil {
			c.logger.Fatalw("couldn't remove dir",
				"diskPath", diskPath,
				"error", err,
			)
		}
	}
	var mergeRef = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", registry.Branch))

	r, err := git.PlainClone(diskPath, false, &git.CloneOptions{
		URL:           registry.URL,
		Progress:      nil,
		Auth:          authMethod,
		SingleBranch:  true,
		NoCheckout:    false,
		ReferenceName: mergeRef,
	})
	if err != nil {
		c.logger.Fatalw("unable to clone the git repo",
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
			c.logger.Fatalw("Error fetching remotes",
				"error", err,
			)
		}
	}

	w, err := r.Worktree()
	if err != nil {
		c.logger.Fatalw("unable to get Worktree of the repo",
			"error", err,
		)
	}

	err = w.Checkout(&git.CheckoutOptions{
		// Hash:  *rev,
		// Force:  true,
		Branch: mergeRef,
	})

	if err != nil {
		c.logger.Fatalw("Error checking out branch",
			"error", err,
		)
	}

	c.logger.Infow("successful checkout",
		"url", registry.URL,
		"branch", registry.Branch,
		"key", registry.Key,
	)

	return r
}

func (c *Client) GetCommitID(path string) string {
	r, err := git.PlainOpen(path)
	if err != nil {
		c.logger.Fatal(err)
	}

	ref, err := r.Head()
	if err != nil {
		c.logger.Fatal(err)
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		c.logger.Fatal(err)
	}
	return fmt.Sprint(commit.Hash)
}
