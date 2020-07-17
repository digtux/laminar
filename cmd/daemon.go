package cmd

import (
	"fmt"
	"github.com/digtux/laminar/pkg/cache"
	"github.com/digtux/laminar/pkg/config"
	"github.com/digtux/laminar/pkg/git"
	"github.com/digtux/laminar/pkg/operations"
	"github.com/digtux/laminar/pkg/registry"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"os"
	"time"
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
		// Override the production Config (I personally don't see the point of using stderr
		// https://github.com/uber-go/zap/blob/feeb9a050b31b40eec6f2470e7599eeeadfe5bdd/config.go#L119
		logConfig := zap.NewProductionConfig()
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

var fileList []string

var rootCmd = &cobra.Command{
	Use:   "daemon",
	Short: "launch laminar daemon service/server",
	Long: `This is the laminar server and API

Laminar is a GitOps utility for automating the promotion of docker images in git.

.. turbulence is bad.. use laminar flow`,
	Run: func(cmd *cobra.Command, args []string) {
		DaemonStart()
	},
}

func DaemonStart() {

	log := startLogger(debug)

	rawFile, err := config.LoadFile(configFile, log)
	if err != nil {
		log.Errorw("Error reading config",
			"file", configFile,
			"error", err,
		)
	}

	cfg, err := config.ParseConfig(rawFile, log)
	if err != nil {
		log.Errorw("error parsing config file",
			"file", configFile,
			"error", err,
		)
	}

	db := cache.Open(configCache, log)
	log.Debug("opened db: ", configCache)

	for _, r := range cfg.GitRepos {
		git.InitialGitClone(r, log)
	}

	for {

		//// from the update policies, make a list of ALL file paths which are referenced in our git repo
		for _, gitRepo := range cfg.GitRepos {

			// This sections deals with loading remote config from the git repo
			// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
			repoPath := git.GetRepoPath(gitRepo)
			if gitRepo.RemoteConfig {
				log.Debugw("Remote config True.. will attempt to update config dynamically",
					"repo", gitRepo.Name,
				)

				remoteUpdates, err := config.GetUpdatesFromGit(repoPath, log)
				if err != nil {
					log.Warnw("Laminar was told to look at .laminar.yaml but failed",
						"repo", gitRepo.Name,
						"path", repoPath,
						"error", err,
					)
				}
				for _, update := range remoteUpdates.Updates {
					log.Debugw("adding config from git Repo",
						"update", update,
					)

					gitRepo.Updates = append(gitRepo.Updates, update)

				}

			}
			log.Infow("Updates configured",
				"count", len(gitRepo.Updates),
				"gitRepo", gitRepo.Name,
			)
			fileList := FileFinder(gitRepo, log)

			// we are ready to dispatch this to start searching the contents of these files
			log.Debugw("matched files in git",
				"GitRepo", gitRepo.Name,
				"fileList", fileList,
			)
		}

		//// TODO: docker reg Timeout?
		//// lets gather a full list of docker images we can find matching the configured registries
		for _, dockerReg := range cfg.DockerRegistries {
			foundDockerImages := FindDockerImages(
				fileList,
				fmt.Sprintf(dockerReg.Reg),
				log,
			)
			registry.Exec(db, dockerReg, foundDockerImages, log)
			if len(foundDockerImages) > 0 {
				log.Infow("found images (in git) matching a configured docker registry",
					"regName", dockerReg.Name,
					"reg", dockerReg.Reg,
					"imageCount", len(foundDockerImages),
				)
			} else {
				log.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
					"regName", dockerReg.Name,
					"reg", dockerReg.Reg,
				)
			}
		}

		//// this is a slice of the registry URLs as we expect to see them inside files
		var registryStrings []string
		for _, reg := range cfg.DockerRegistries {
			registryStrings = append(registryStrings, reg.Reg)
		}

		//// now that we can assume we have some tags in cache, we run a
		//// loop over GitRepos
		//
		for _, gitRepo := range cfg.GitRepos {
			triggerCommitAndPush := false
			changeCount := 0
			for _, updatePolicy := range gitRepo.Updates {
				fileList = []string{}
				// assemble a list of target files for this Update
				for _, p := range updatePolicy.Files {
					// get the path of where the git repo is checked out
					relativeGitPath := git.GetRepoPath(gitRepo)
					// combine these
					realPath := fmt.Sprintf("%s/%s", relativeGitPath, p.Path)

					// finally this will return all files found
					for _, x := range operations.FindFiles(realPath, log) {
						fileList = append(fileList, x)
					}
				}

				for _, f := range fileList {
					log.Infow("applying update policy",
						"file", f,
						"pattern", updatePolicy.PatternString,
						"blacklist", updatePolicy.BlackList,
					)
					fileChangesHappened := DoUpdate(f, updatePolicy, registryStrings, db, log)
					if fileChangesHappened > 0 {
						triggerCommitAndPush = true
						changeCount = changeCount + fileChangesHappened
						fileChangesHappened = 0
					}
				}
			}

			if triggerCommitAndPush {
				msg := fmt.Sprintf("%s [%d]", cfg.Global.GitMessage, changeCount)
				log.Warnw("pretending to commit",
					"gitRepo", gitRepo)
				git.CommitAndPush(gitRepo, cfg.Global, msg, log)
			}

			// TODO: use a Tick() instead of this Sleep()
			time.Sleep(60 * time.Second)
		}

	}

}