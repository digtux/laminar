package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/digtux/laminar/pkg/common"
	"github.com/digtux/laminar/pkg/web"
	"github.com/pkg/errors"
	"github.com/tidwall/buntdb"

	"github.com/digtux/laminar/pkg/cache"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/gitoperations"
	"github.com/digtux/laminar/pkg/operations"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "run",
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
	cacheDB          *buntdb.DB
	fileList         []string // list of files containing docker images urls
	gitState         []GitState
	dockerRegistries map[string]cfg.DockerRegistry
	gitConfig        cfg.Global
	gitOpsClient     *gitoperations.Client
	opsClient        *operations.Client
}

func New() (d *Daemon, err error) {
	logger := common.GetLogger(debug)
	appConfig, err := loadConfig(logger)
	if err != nil {
		return nil, err
	}
	cacheDB := cache.Open(configCache, logger)
	d = &Daemon{
		cacheDB:          cacheDB,
		dockerRegistries: mapDockerRegistries(appConfig.DockerRegistries),
		gitConfig:        appConfig.Global,
		gitOpsClient:     gitoperations.New(logger, appConfig.Global),
		gitState:         nil,
		logger:           logger,
		opsClient:        operations.New(logger),
		registryClient:   registry.New(logger, cacheDB),
		webClient:        web.New(logger, appConfig),
	}
	d.initialiseGitState(appConfig.GitRepos)
	return
}

func mapDockerRegistries(registries []cfg.DockerRegistry) map[string]cfg.DockerRegistry {
	result := map[string]cfg.DockerRegistry{}
	for _, reg := range registries {
		result[reg.Reg] = reg
	}
	return result
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
			"file", configFile,
			"error", err,
		)
	}
	return
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) initialiseGitState(repos []cfg.GitRepo) {
	d.gitState = make([]GitState, len(repos))
	for i, repoCfg := range repos {
		d.gitState[i] = GitState{
			Repo:    d.gitOpsClient.InitialGitCloneAndCheckout(repoCfg),
			repoCfg: &repoCfg,
		}
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) Start() {
	d.logger.Debug("opened db: ", configCache)
	if oneShot {
		d.masterTask()
		os.Exit(0)
	}
	go d.webClient.StartWeb()
	d.enterControlLoop()
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) enterControlLoop() {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	doWork := func() {
		d.masterTask()
	}
	// when we enter the control loop, no point waiting. just go for it
	doWork()
	// now await http or tick events
	for {
		select {
		case repo := <-d.webClient.BuildChan:
			d.singleRepoTask(repo)
		case <-d.webClient.PauseChan:
			d.pause()
		case <-ticker.C:
			doWork()
		}
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) pause() {
	d.logger.Infow("laminar paused",
		"pauseDuration", pauseDuration,
	)
	<-time.Tick(pauseDuration)
	d.logger.Infow("laminar paused expired. continuing")
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) masterTask() {
	// from the update policies, make a list of ALL file paths which are referenced in our gitoperations repo
	for _, state := range d.gitState {
		d.updateGitRepoState(state)
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

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) singleRepoTask(r web.DockerBuildJSON) {
	if reg, ok := d.dockerRegistries[r.DockerRegistryURL]; ok {
		for _, state := range d.gitState {
			d.updateGitRepoState(state)
		}

		d.scanDockerRegistry(reg)

		for _, state := range d.gitState {
			d.updateFiles(*state.repoCfg)
		}
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) updateFiles(gitRepo cfg.GitRepo) {
	registryStrings := d.getRegistryStrings()
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
			filesFound := d.opsClient.FindFiles(realPath)
			d.logger.Debugw("found files in git repo", "filesFound", filesFound)
			fileList = append(fileList, filesFound...)
		}

		for _, filePath := range fileList {
			d.logger.Debugw("applying update policy",
				"file", filePath,
				"pattern", updatePolicy.PatternString,
				"blacklist", updatePolicy.BlackList,
			)
			newChanges := d.doUpdate(filePath, updatePolicy, registryStrings)
			if len(newChanges) > 0 {
				d.logger.Infow("updates desired",
					"file", filePath,
					"pattern", updatePolicy.PatternString,
				)
				triggerCommitAndPush = true
				changes = append(changes, newChanges...)
			}
		}
	}

	if triggerCommitAndPush {
		d.commitAndPush(changes, gitRepo)
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) commitAndPush(changes []ChangeRequest, cfgGit cfg.GitRepo) {
	msg := ""
	if len(changes) > 1 {
		msg = fmt.Sprintf("%s [%d]", cfgGit.Name, len(changes))
	} else {
		msg = nicerMessage(changes[0])
	}
	d.logger.Infow("doing commit",
		"gitRepo", cfgGit.URL,
		"msg", msg,
	)
	d.gitOpsClient.CommitAndPush(cfgGit, msg)
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) scanDockerRegistries() {
	for _, dockerReg := range d.dockerRegistries {
		d.scanDockerRegistry(dockerReg)
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) scanDockerRegistry(dockerReg cfg.DockerRegistry) {
	d.logger.Infow("scanning docker registry", "url", dockerReg.Reg)
	foundDockerImages := d.FindDockerImages(
		d.fileList,
		fmt.Sprintf(dockerReg.Reg),
	)
	if len(foundDockerImages) > 0 {
		d.registryClient.Exec(dockerReg, foundDockerImages)
		d.logger.Infow("found images (in gitoperations) matching a configured docker registry",
			"regName", dockerReg.Name,
			"reg", dockerReg.Reg,
			"imageCount", len(foundDockerImages),
		)
	} else {
		d.logger.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
			"regName", dockerReg.Name,
			"reg", dockerReg.Reg,
		)
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) updateGitRepoState(state GitState) {
	// Clone all repos that haven't been cloned yet
	if state.Repo != nil {
		d.gitOpsClient.Pull(*state.repoCfg)
	} else {
		d.logger.Warnw("repo has not been initialised",
			"repo.URL", state.repoCfg.URL)
		return
	}

	// This sections deals with loading remote config from the gitoperations repo
	// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
	repoPath := gitoperations.GetRepoPath(*state.repoCfg)
	if state.repoCfg.RemoteConfig {
		d.logger.Debugw("'remote config' == True.. will attempt to update config dynamically",
			"repo", state.repoCfg.Name,
		)

		remoteUpdates, err := cfg.GetUpdatesFromGit(repoPath, d.logger)
		if err != nil {
			d.logger.Warnw("Laminar was told to look at .laminar.yaml but failed",
				"repo", state.repoCfg.Name,
				"path", repoPath,
				"error", err,
			)
		}

		// clear out the Updates for this repoNum
		state.repoCfg.Updates = make([]cfg.Updates, 0)
		// now assemble that list for this run
		for _, update := range remoteUpdates.Updates {
			d.logger.Infow("using 'remote config' from gitoperations repo .laminar.yaml",
				"update", update,
			)
			state.repoCfg.Updates = append(state.repoCfg.Updates, update)
		}
	}
	// equalise the state. damn this needs a nice rewrite sometime
	d.logger.Infow("configured for",
		"gitRepo", state.repoCfg.Name,
		"updateRules", len(state.repoCfg.Updates),
	)
	d.UpdateFileList(*state.repoCfg)

	// we are ready to dispatch this to start searching the contents of these files
	d.logger.Debugw("matched files in gitoperations",
		"GitRepo", state.repoCfg.Name,
		"fileListCount", len(d.fileList),
	)
}

//goland:noinspection GoMixedReceiverTypes
func (d Daemon) getRegistryStrings() []string {
	// this is a slice of the registry URLs as we expect to see them inside files
	var registryStrings []string
	for _, reg := range d.dockerRegistries {
		registryStrings = append(registryStrings, reg.Reg)
	}
	return registryStrings
}
