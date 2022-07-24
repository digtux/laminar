package cmd

import (
	"fmt"
	"github.com/digtux/laminar/pkg/web"
	"github.com/pkg/errors"
	"github.com/tidwall/buntdb"
	"os"
	"time"

	"go.uber.org/zap/zapcore"

	"github.com/digtux/laminar/pkg/cache"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/gitoperations"
	"github.com/digtux/laminar/pkg/operations"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// switch between a vanilla Development or Production logging format (--debug)
// The only change from vanilla zap is the ProductionConfig outputs to stdout instead of stderr
func startLogger(debug bool) (zapLog *zap.SugaredLogger) {
	// https://blog.sandipb.net/2018/05/02/using-zap-simple-use-cases/
	if debug {
		zapLogger, err := zap.NewDevelopment()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		sugar := zapLogger.Sugar()
		sugar.Debug("debug enabled")
		return sugar
	} else {
		// Override the default zap production Config a little
		// NewProductionConfig is json

		logConfig := zap.NewProductionConfig()
		// customise the "time" field to be ISO8601
		logConfig.EncoderConfig.TimeKey = "time"
		logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		// main message data into the key "msg"
		logConfig.EncoderConfig.MessageKey = "msg"

		// stdout+sterr into stdout
		logConfig.OutputPaths = []string{"stdout"}
		logConfig.ErrorOutputPaths = []string{"stdout"}
		zapLogger, err := logConfig.Build()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return zapLogger.Sugar()
	}
}

var rootCmd = &cobra.Command{
	Use:   "daemon",
	Short: "launch laminar daemon service/server",
	Long: `This is the laminar server and API

Laminar is a GitOps utility for automating the promotion of docker images in gitoperations.

.. turbulence is bad.. use laminar flow`,
	Run: func(cmd *cobra.Command, args []string) {
		d, err := New()
		if err != nil {
			panic(err)
		}
		d.Start()
	},
}

type GitState struct {
	Repo    *git.Repository
	repoCfg *cfg.GitRepo
}

type Daemon struct {
	logger           *zap.SugaredLogger
	registryClient   *registry.Client
	webClient        *web.Client
	cacheDb          *buntdb.DB
	fileList         []string // list of files containing docker images urls
	gitState         []GitState
	dockerRegistries []cfg.DockerRegistry
	gitConfig        cfg.Global
	gitOpsClient     *gitoperations.Client
	opsClient        *operations.Client
}

func New() (d *Daemon, err error) {
	logger := startLogger(debug)
	appConfig, err := loadConfig(logger)
	if err != nil {
		return nil, err
	}
	cacheDb := cache.Open(configCache, logger)
	d = &Daemon{
		logger:           logger,
		registryClient:   registry.New(logger, cacheDb),
		gitState:         nil,
		webClient:        web.New(logger, appConfig.Global.GitHubToken),
		cacheDb:          cacheDb,
		dockerRegistries: appConfig.DockerRegistries,
		gitConfig:        appConfig.Global,
		gitOpsClient:     gitoperations.New(logger, appConfig.Global),
		opsClient:        operations.New(logger),
	}
	d.initialiseGitState(appConfig.GitRepos)
	return
}

func loadConfig(logger *zap.SugaredLogger) (appConfig cfg.Config, err error) {
	var rawFile []byte
	if rawFile, err = cfg.LoadFile(configFile, logger); err == nil {
		if appConfig, err = cfg.ParseConfig(rawFile); err != nil {
			err = errors.Wrap(err, "error parsing config file")
		}
	} else {
		err = errors.Wrap(err, "error reading config")
	}
	if err != nil {
		logger.Errorw("error loading config",
			"laminar.file", configFile,
			"laminar.error", err,
		)
	}
	return
}

func (d *Daemon) initialiseGitState(repos []cfg.GitRepo) {
	d.gitState = make([]GitState, len(repos))
	for i, repoCfg := range repos {
		d.gitState[i] = GitState{
			Repo:    d.gitOpsClient.InitialGitCloneAndCheckout(repoCfg),
			repoCfg: &repoCfg,
		}
	}
}

func (d *Daemon) Start() {
	go d.webClient.StartWeb()
	d.logger.Debug("opened db: ", configCache)
	d.enterControlLoop()
}

func (d *Daemon) enterControlLoop() {
	ticker := time.Tick(interval)
	for {
		select {
		case repo := <-d.webClient.BuildChan:
			d.scanNow(repo)
		case <-d.webClient.PauseChan:
			d.pause()
		case <-ticker:
			d.masterTask()
		}
	}
}

func (d *Daemon) pause() {
	d.logger.Infow("laminar paused",
		"pauseDuration", pauseDuration,
	)
	<-time.Tick(pauseDuration)
	d.logger.Infow("laminar paused expired. continuing")
}

func (d *Daemon) masterTask() {
	// from the update policies, make a list of ALL file paths which are referenced in our gitoperations repo
	for _, state := range d.gitState {
		d.updateRepoStates(state)
	}

	// TODO: docker reg Timeout?
	// lets gather a full list of docker images we can find matching the configured registries
	d.scanDockerRegistries()

	// now that we can assume we have some tags in cache, we run a
	// loop over GitRepos
	for _, state := range d.gitState {
		d.updateFiles(*state.repoCfg)
	}
	if oneShot {
		d.logger.Warn("--one-shot detected.. laminar is now terminating")
		os.Exit(0)
	}
}

func (d *Daemon) updateFiles(gitRepo cfg.GitRepo) {
	registryStrings := getRegistryStrings(d.dockerRegistries)
	triggerCommitAndPush := false
	var changes []ChangeRequest
	for _, updatePolicy := range gitRepo.Updates {
		var fileList []string
		// assemble a list of target files for this Update
		for _, p := range updatePolicy.Files {
			// get the path of where the gitoperations repo is checked out
			relativeGitPath := gitoperations.GetRepoPath(gitRepo)
			// combine these
			realPath := fmt.Sprintf("%s/%s", relativeGitPath, p.Path)

			// finally this will return all files found
			for _, paths := range d.opsClient.FindFiles(realPath) {
				fileList = append(fileList, paths)
			}
		}

		for _, filePath := range fileList {
			d.logger.Debugw("applying update policy",
				"laminar.file", filePath,
				"laminar.pattern", updatePolicy.PatternString,
				"laminar.blacklist", updatePolicy.BlackList,
			)
			newChanges := d.doUpdate(filePath, updatePolicy, registryStrings)
			if len(newChanges) > 0 {
				d.logger.Infow("updates desired",
					"laminar.file", filePath,
					"laminar.pattern", updatePolicy.PatternString,
				)
				triggerCommitAndPush = true
				for _, stuffDone := range newChanges {
					changes = append(changes, stuffDone)
				}
			}
		}
	}

	if triggerCommitAndPush {
		d.commitAndPush(changes, gitRepo)
	}
}

func (d *Daemon) commitAndPush(changes []ChangeRequest, repo cfg.GitRepo) {
	msg := ""
	if len(changes) > 1 {
		msg = fmt.Sprintf("%s [%d]", d.gitConfig, len(changes))
	} else {
		prettyMessage := nicerMessage(changes[0])
		//fmt.Println(changes)
		msg = fmt.Sprintf("%s", prettyMessage)
	}
	d.logger.Infow("doing commit",
		"laminar.gitRepo", repo.URL,
		"laminar.msg", msg,
	)
	d.gitOpsClient.CommitAndPush(repo, msg)
}

func (d *Daemon) scanDockerRegistries() {
	for _, dockerReg := range d.dockerRegistries {
		foundDockerImages := d.FindDockerImages(
			d.fileList,
			fmt.Sprintf(dockerReg.Reg),
		)
		d.registryClient.Exec(dockerReg, foundDockerImages)
		if len(foundDockerImages) > 0 {
			d.logger.Infow("found images (in gitoperations) matching a configured docker registry",
				"laminar.regName", dockerReg.Name,
				"laminar.reg", dockerReg.Reg,
				"laminar.imageCount", len(foundDockerImages),
			)
		} else {
			d.logger.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
				"laminar.regName", dockerReg.Name,
				"laminar.reg", dockerReg.Reg,
			)
		}
	}
}

func (d *Daemon) updateRepoStates(state GitState) {
	// Clone all repos that haven't been cloned yet
	if state.Repo != nil {
		d.gitOpsClient.Pull(*state.repoCfg)
	} else {
		d.logger.Warnw("repo has not been initialised",
			"repo.URL", state.repoCfg.URL)
		//TODO: implement initialisation of uninitialised repos (probs issue for dynamic configs)
		return
	}

	// This sections deals with loading remote config from the gitoperations repo
	// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
	repoPath := gitoperations.GetRepoPath(*state.repoCfg)
	if state.repoCfg.RemoteConfig {
		d.logger.Debugw("'remote config' == True.. will attempt to update config dynamically",
			"laminar.repo", state.repoCfg.Name,
		)

		remoteUpdates, err := cfg.GetUpdatesFromGit(repoPath, d.logger)
		if err != nil {
			d.logger.Warnw("Laminar was told to look at .laminar.yaml but failed",
				"laminar.repo", state.repoCfg.Name,
				"laminar.path", repoPath,
				"laminar.error", err,
			)
		}

		// clear out the Updates for this repoNum
		state.repoCfg.Updates = make([]cfg.Updates, 0)
		// now assemble that list for this run
		for _, update := range remoteUpdates.Updates {
			d.logger.Infow("using 'remote config' from gitoperations repo .laminar.yaml",
				"laminar.update", update,
			)
			state.repoCfg.Updates = append(state.repoCfg.Updates, update)
		}
	}
	// equalise the state.. damn this needs a nice rewrite sometime
	d.logger.Infow("configured for",
		"laminar.gitRepo", state.repoCfg.Name,
		"laminar.updateRules", len(state.repoCfg.Updates),
	)
	d.UpdateFileList(*state.repoCfg)

	// we are ready to dispatch this to start searching the contents of these files
	d.logger.Debugw("matched files in gitoperations",
		"laminar.GitRepo", state.repoCfg.Name,
		"laminar.fileList", d.fileList,
	)
}

func (d *Daemon) scanNow(repo string) {
	//todo: implement
}

func getRegistryStrings(registries []cfg.DockerRegistry) []string {
	// this is a slice of the registry URLs as we expect to see them inside files
	var registryStrings []string
	for _, reg := range registries {
		registryStrings = append(registryStrings, reg.Reg)
	}
	return registryStrings
}
