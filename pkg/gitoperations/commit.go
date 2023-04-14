package gitoperations

import (
	"bytes"
	"os"
	"os/exec"
	"time"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/logger"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Client struct {
	config cfg.Global
}

func New(config cfg.Global) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) executeCmd(command string, path string) {
	var stdout bytes.Buffer

	logger.Infow("executeCmd",
		"laminar.command", command,
	)

	args := []string{"-c"}

	args = append(args, command)

	cmd := exec.Command("sh", args...)
	// , args...)
	// set the location for the command
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	// run it
	err := cmd.Run()
	if err != nil {
		logger.Error(err)
	}
	logger.Infow("exec",
		"laminar.command", "sh -c "+command,
		"laminar.output", stdout.String(),
	)
}

func (c *Client) CommitAndPush(registry cfg.GitRepo, message string) {
	path := GetRepoPath(registry)
	r, err := git.PlainOpen(path)
	if err != nil {
		logger.Errorw("error opening repo",
			"laminar.registry", registry,
			"laminar.error", err,
		)
	}

	w, err := r.Worktree()
	if err != nil {
		logger.Fatalw("CommitAndPush: Couldn't open git",
			"path", path,
			"err", err)
	}

	if len(registry.PreCommitCommands) > 0 {
		for _, cmd := range registry.PreCommitCommands {
			c.executeCmd(cmd, path)
		}
	}

	// auth := getAuth(registry.Key)
	logger.Infow("time to commit git",
		"laminar.registry", registry.URL,
		"laminar.branch", registry.Branch,
	)
	// _, err = w.Add("./")
	// if err != nil {
	// 	log.Error(err, "add")
	// }
	// log.Debug("did a git add")
	// status, err := w.Status()
	// if err != nil {
	// 	log.Error(err, status)
	// }
	commit, err := w.Commit(message, &git.CommitOptions{
		All: true,
		Author: &object.Signature{
			Name:  c.config.GitUser,
			Email: c.config.GitEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		logger.Errorw("Error doing git commit",
			"laminar.error", err,
		)
	}
	obj, err := r.CommitObject(commit)
	if err != nil {
		logger.Error(err)
	}

	// push using default options
	logger.Infow("doing git push",
		"laminar.commit", commit,
		"laminar.obj", obj,
	)
	err = r.Push(&git.PushOptions{})
	if err != nil {
		// TODO: handle this error by re-cloning the repo or similar
		// TODO: don't handle this until there are prometheus metrics to alert us of issues
		logger.Fatalw("Something terrible happened!!!!",
			"error", err,
		)
		os.Exit(1)
	}
}
