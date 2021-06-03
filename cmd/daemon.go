package cmd

import (
	"fmt"
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

var fileList []string

var rootCmd = &cobra.Command{
	Use:   "daemon",
	Short: "launch laminar daemon service/server",
	Long: `This is the laminar server and API

Laminar is a GitOps utility for automating the promotion of docker images in gitoperations.

.. turbulence is bad.. use laminar flow`,
	Run: func(cmd *cobra.Command, args []string) {
		DaemonStart()
	},
}

type GitState struct {
	Repo   *git.Repository
	Cloned bool
}

func DaemonStart() {

	log := startLogger(debug)

	rawFile, err := cfg.LoadFile(configFile, log)
	if err != nil {
		log.Errorw("Error reading config",
			"laminar.file", configFile,
			"laminar.error", err,
		)
	}

	appConfig, err := cfg.ParseConfig(rawFile, log)
	if err != nil {
		log.Errorw("error parsing config file",
			"laminar.file", configFile,
			"laminar.error", err,
		)
	}

	db := cache.Open(configCache, log)
	log.Debug("opened db: ", configCache)

	var GitStateList []GitState

	for _, r := range appConfig.GitRepos {
		log.Info(appConfig)
		repoObj := gitoperations.InitialGitCloneAndCheckout(r, log)
		newObj := GitState{
			Repo:   repoObj,
			Cloned: true,
		}
		GitStateList = append(GitStateList, newObj)
	}

	for {

		//// from the update policies, make a list of ALL file paths which are referenced in our gitoperations repo
		for repoNum, gitRepo := range appConfig.GitRepos {

			if GitStateList[repoNum].Cloned {
				w := GitStateList[repoNum].Repo
				gitoperations.Pull(w, gitRepo, log)
			}

			// This sections deals with loading remote config from the gitoperations repo
			// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
			repoPath := gitoperations.GetRepoPath(gitRepo)
			if gitRepo.RemoteConfig {
				log.Debugw("'remote config' == True.. will attempt to update config dynamically",
					"laminar.repo", gitRepo.Name,
				)

				remoteUpdates, err := cfg.GetUpdatesFromGit(repoPath, log)
				if err != nil {
					log.Warnw("Laminar was told to look at .laminar.yaml but failed",
						"laminar.repo", gitRepo.Name,
						"laminar.path", repoPath,
						"laminar.error", err,
					)
				}

				// clear out the Updates for this repoNum
				appConfig.GitRepos[repoNum].Updates = []cfg.Updates{}
				// now assemble that list for this run
				for _, update := range remoteUpdates.Updates {
					log.Infow("using 'remote config' from gitoperations repo .laminar.yaml",
						"laminar.update", update,
					)
					appConfig.GitRepos[repoNum].Updates = append(appConfig.GitRepos[repoNum].Updates, update)
				}

			}
			// equalise the state.. damn this needs a nice rewrite sometime
			gitRepo = appConfig.GitRepos[repoNum]
			log.Infow("configured for",
				"laminar.gitRepo", gitRepo.Name,
				"laminar.updateRules", len(gitRepo.Updates),
			)
			fileList := FileFinder(gitRepo, log)

			// we are ready to dispatch this to start searching the contents of these files
			log.Debugw("matched files in gitoperations",
				"laminar.GitRepo", gitRepo.Name,
				"laminar.fileList", fileList,
			)
		}

		//// TODO: docker reg Timeout?
		//// lets gather a full list of docker images we can find matching the configured registries
		for _, dockerReg := range appConfig.DockerRegistries {
			foundDockerImages := FindDockerImages(
				fileList,
				fmt.Sprintf(dockerReg.Reg),
				log,
			)
			registry.Exec(db, dockerReg, foundDockerImages, log)
			if len(foundDockerImages) > 0 {
				log.Infow("found images (in gitoperations) matching a configured docker registry",
					"laminar.regName", dockerReg.Name,
					"laminar.reg", dockerReg.Reg,
					"laminar.imageCount", len(foundDockerImages),
				)
			} else {
				log.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
					"laminar.regName", dockerReg.Name,
					"laminar.reg", dockerReg.Reg,
				)
			}
		}

		//// this is a slice of the registry URLs as we expect to see them inside files
		var registryStrings []string
		for _, reg := range appConfig.DockerRegistries {
			registryStrings = append(registryStrings, reg.Reg)
		}

		//// now that we can assume we have some tags in cache, we run a
		//// loop over GitRepos
		//
		for _, gitRepo := range appConfig.GitRepos {
			triggerCommitAndPush := false
			var changes []ChangeRequest
			for _, updatePolicy := range gitRepo.Updates {
				fileList = []string{}
				// assemble a list of target files for this Update
				for _, p := range updatePolicy.Files {
					// get the path of where the gitoperations repo is checked out
					relativeGitPath := gitoperations.GetRepoPath(gitRepo)
					// combine these
					realPath := fmt.Sprintf("%s/%s", relativeGitPath, p.Path)

					// finally this will return all files found
					for _, x := range operations.FindFiles(realPath, log) {
						fileList = append(fileList, x)
					}
				}

				for _, f := range fileList {
					log.Debugw("applying update policy",
						"laminar.file", f,
						"laminar.pattern", updatePolicy.PatternString,
						"laminar.blacklist", updatePolicy.BlackList,
					)
					newChanges := DoUpdate(f, updatePolicy, registryStrings, db, log)
					if len(newChanges) > 0 {
						log.Infow("updates desired",
							"laminar.file", f,
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
				msg := ""
				if len(changes) > 1 {
					msg = fmt.Sprintf("%s [%d]", appConfig.Global.GitMessage, len(changes))
				} else {
					prettyMessage := nicerMessage(changes[0])
					//fmt.Println(changes)
					msg = fmt.Sprintf("%s", prettyMessage)
				}
				log.Infow("doing commit",
					"laminar.gitRepo", gitRepo.URL,
					"laminar.msg", msg,
				)
				gitoperations.CommitAndPush(gitRepo, appConfig.Global, msg, log)
			}

		}
		if oneShot {
			log.Warn("--one-shot detected.. laminar is now terminating")
			os.Exit(0)
		}
		// TODO: use a Tick() instead of this Sleep()
		//time.Sleep(10 * time.Second)
		time.Sleep(interval)

	}

}
