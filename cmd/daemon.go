package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/digtux/laminar/pkg/web"
	"github.com/pkg/errors"
	"github.com/tidwall/buntdb"

	"github.com/digtux/laminar/pkg/cache"
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/gitoperations"
	"github.com/digtux/laminar/pkg/logger"
	"github.com/digtux/laminar/pkg/operations"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
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
	appConfig, err := loadConfig()
	if err != nil {
		return nil, err
	}
	cacheDB := cache.Open(configCache)
	d = &Daemon{
		cacheDB:          cacheDB,
		dockerRegistries: mapDockerRegistries(appConfig.DockerRegistries),
		gitConfig:        appConfig.Global,
		gitOpsClient:     gitoperations.New(appConfig.Global),
		gitState:         nil,
		opsClient:        operations.New(),
		registryClient:   registry.New(cacheDB),
		webClient:        web.New(appConfig),
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

func loadConfig() (appConfig cfg.Config, err error) {
	var rawFile []byte
	if rawFile, err = cfg.LoadFile(configFile); err == nil {
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
	logger.Debug("opened db: ", configCache)
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
	logger.Infow("laminar paused",
		"pauseDuration", pauseDuration,
	)
	<-time.Tick(pauseDuration)
	logger.Infow("laminar paused expired. continuing")
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
		logger.Warn("--one-shot detected.. laminar is now terminating")
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
			logger.Debugw("found files in git repo", "laminar.filesFound", filesFound)
			fileList = append(fileList, filesFound...)
		}

		for _, filePath := range fileList {
			logger.Debugw("applying update policy",
				"laminar.file", filePath,
				"laminar.pattern", updatePolicy.PatternString,
				"laminar.blacklist", updatePolicy.BlackList,
			)
			newChanges := d.doUpdate(filePath, updatePolicy, registryStrings)
			if len(newChanges) > 0 {
				logger.Infow("updates desired",
					"laminar.file", filePath,
					"laminar.pattern", updatePolicy.PatternString,
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
	logger.Infow("doing commit",
		"laminar.gitRepo", cfgGit.URL,
		"laminar.msg", msg,
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
	logger.Infow("scanning docker registry", "url", dockerReg.Reg)
	foundDockerImages := d.FindDockerImages(
		d.fileList,
		fmt.Sprintf(dockerReg.Reg),
	)
	if len(foundDockerImages) > 0 {
		d.registryClient.Exec(dockerReg, foundDockerImages)
		logger.Infow("found images (in gitoperations) matching a configured docker registry",
			"laminar.regName", dockerReg.Name,
			"laminar.reg", dockerReg.Reg,
			"laminar.imageCount", len(foundDockerImages),
		)
	} else {
		logger.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
			"laminar.regName", dockerReg.Name,
			"laminar.reg", dockerReg.Reg,
		)
	}
}

//goland:noinspection GoMixedReceiverTypes
func (d *Daemon) updateGitRepoState(state GitState) {
	// Clone all repos that haven't been cloned yet
	if state.Repo != nil {
		d.gitOpsClient.Pull(*state.repoCfg)
	} else {
		logger.Warnw("repo has not been initialised",
			"repo.URL", state.repoCfg.URL)
		return
	}

	// This sections deals with loading remote config from the gitoperations repo
	// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
	repoPath := gitoperations.GetRepoPath(*state.repoCfg)
	if state.repoCfg.RemoteConfig {
		logger.Debugw("'remote config' == True.. will attempt to update config dynamically",
			"laminar.repo", state.repoCfg.Name,
		)

		remoteUpdates, err := cfg.GetUpdatesFromGit(repoPath)
		if err != nil {
			logger.Warnw("Laminar was told to look at .laminar.yaml but failed",
				"laminar.repo", state.repoCfg.Name,
				"laminar.path", repoPath,
				"laminar.error", err,
			)
		}

		// clear out the Updates for this repoNum
		state.repoCfg.Updates = make([]cfg.Updates, 0)
		// now assemble that list for this run
		for _, update := range remoteUpdates.Updates {
			logger.Infow("using 'remote config' from gitoperations repo .laminar.yaml",
				"laminar.update", update,
			)
			state.repoCfg.Updates = append(state.repoCfg.Updates, update)
		}
	}
	// equalise the state. damn this needs a nice rewrite sometime
	logger.Infow("configured for",
		"laminar.gitRepo", state.repoCfg.Name,
		"laminar.updateRules", len(state.repoCfg.Updates),
	)
	d.UpdateFileList(*state.repoCfg)

	// we are ready to dispatch this to start searching the contents of these files
	logger.Debugw("matched files in gitoperations",
		"laminar.GitRepo", state.repoCfg.Name,
		"laminar.fileList", d.fileList,
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
