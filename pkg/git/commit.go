package git

import (
	"github.com/digtux/laminar/pkg/config"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"go.uber.org/zap"
	"time"
)

func CommitAndPush(registry config.GitRepo, global config.Global, message string, log *zap.SugaredLogger) {
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
	if err != nil {
		log.Error(err, status)
	}

}
