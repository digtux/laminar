package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"go.uber.org/zap"
)

func executeCmd(command string, path string, log *zap.SugaredLogger) {

	var stdout bytes.Buffer

	log.Infow("executeCmd",
		"command", command,
	)

	args := []string{"-c"}

	args = append(args, command)

	cmd := exec.Command("sh", args...)
	//, args...)
	// set the location for the command
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	// run it
	err := cmd.Run()
	if err != nil {
		log.Error(err)
	}
	fmt.Println(stdout.String())
	// TODO: improve logging
}

func CommitAndPush(registry cfg.GitRepo, global cfg.Global, message string, log *zap.SugaredLogger) {
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

	if len(registry.PreCommitCommands) > 0 {
		for _, cmd := range registry.PreCommitCommands {
			executeCmd(cmd, path, log)
		}
	}

	// auth := getAuth(registry.Key)
	log.Debugw("time to commit git",
		"registry", registry.URL,
		"branch", registry.Branch,
	)
	_, err = w.Add("./")
	if err != nil {
		log.Error(err, "add")
	}
	log.Debug("did a git add")
	status, err := w.Status()
	if err != nil {
		log.Error(err, status)
	}
	commit, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  global.GitUser,
			Email: global.GitEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Error(err, status)
	}
	log.Debug("git show -s")
	obj, err := r.CommitObject(commit)
	if err != nil {
		log.Error(err, status)
	}

	log.Debug(obj)

	// push using default options
	log.Info("git push")
	err = r.Push(&git.PushOptions{})
	// TODO: handle this error by re-cloning the repo or similar
	// TODO: don't handle this untill there are prometheus metrics to alert us of issues
	if err != nil {
		log.Fatalw("Something terrible happened!!!!",
			"error", err,
			"status", status,
		)
	}

}
