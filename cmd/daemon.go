package cmd

import (
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/git"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"net/http"
)

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

// decide if we want the webserver to log or not here
func skipperLogs(ctx echo.Context) bool {
	if ctx.Path() == "/healthz" {
		return true
	}
	return false
}

// DaemonStart is the main entrypoint for laminar
func DaemonStart() {

	// start a logger.
	log, _ := startLogger(debug)

	// infer config
	appConfig := cfg.LoadConfig(configFile, log)

	// start a new echo instance
	e := echo.New()

	// hide banner and port messages at launch (we'll use a the regular logger to ensure consistency)
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		// https://echo.labstack.com/middleware/logger
		Skipper: skipperLogs,
	}))

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// start the webserver (blocks here)
	log.Info("starting webserver")
	e.Logger.Fatal(e.Start(":1313"))

	//db := cache.Open(configCache, log)
	//log.Debug("opened db: ", configCache)

	for _, r := range appConfig.GitRepos {
		git.InitialGitCloneAndCheckout(r, log)
	}

	//for {
	//
	//	//// from the update policies, make a list of ALL file paths which are referenced in our git repo
	//	for _, gitRepo := range appConfig.GitRepos {
	//
	//		// This sections deals with loading remote config from the git repo
	//		// if RemoteConfig is set we want to attempt to read '.laminar.yaml' from the remote repo
	//		repoPath := git.GetRepoPath(gitRepo)
	//		if gitRepo.RemoteConfig {
	//			log.Debugw("Remote config True.. will attempt to update config dynamically",
	//				"repo", gitRepo.Name,
	//			)
	//
	//			remoteUpdates, err := cfg.GetUpdatesFromGit(repoPath, log)
	//			if err != nil {
	//				log.Warnw("Laminar was told to look at .laminar.yaml but failed",
	//					"repo", gitRepo.Name,
	//					"path", repoPath,
	//					"error", err,
	//				)
	//			}
	//			for _, update := range remoteUpdates.Updates {
	//				log.Debugw("adding config from git Repo",
	//					"update", update,
	//				)
	//
	//				gitRepo.Updates = append(gitRepo.Updates, update)
	//
	//			}
	//
	//		}
	//		log.Infow("Updates configured",
	//			"count", len(gitRepo.Updates),
	//			"gitRepo", gitRepo.Name,
	//		)
	//		fileList := FileFinder(gitRepo, log)
	//
	//		// we are ready to dispatch this to start searching the contents of these files
	//		log.Debugw("matched files in git",
	//			"GitRepo", gitRepo.Name,
	//			"fileList", fileList,
	//		)
	//	}
	//
	//	//// TODO: docker reg Timeout?
	//	//// lets gather a full list of docker images we can find matching the configured registries
	//	for _, dockerReg := range appConfig.DockerRegistries {
	//		foundDockerImages := FindDockerImages(
	//			fileList,
	//			fmt.Sprintf(dockerReg.Reg),
	//			log,
	//		)
	//		registry.Exec(db, dockerReg, foundDockerImages, log)
	//		if len(foundDockerImages) > 0 {
	//			log.Infow("found images (in git) matching a configured docker registry",
	//				"regName", dockerReg.Name,
	//				"reg", dockerReg.Reg,
	//				"imageCount", len(foundDockerImages),
	//			)
	//		} else {
	//			log.Infow("no images tags found.. ensure the full <image>:<tag> strings present",
	//				"regName", dockerReg.Name,
	//				"reg", dockerReg.Reg,
	//			)
	//		}
	//	}
	//
	//	//// this is a slice of the registry URLs as we expect to see them inside files
	//	var registryStrings []string
	//	for _, reg := range appConfig.DockerRegistries {
	//		registryStrings = append(registryStrings, reg.Reg)
	//	}
	//
	//	//// now that we can assume we have some tags in cache, we run a
	//	//// loop over GitRepos
	//	//
	//	for _, gitRepo := range appConfig.GitRepos {
	//		triggerCommitAndPush := false
	//		changeCount := 0
	//		for _, updatePolicy := range gitRepo.Updates {
	//			fileList = []string{}
	//			// assemble a list of target files for this Update
	//			for _, p := range updatePolicy.Files {
	//				// get the path of where the git repo is checked out
	//				relativeGitPath := git.GetRepoPath(gitRepo)
	//				// combine these
	//				realPath := fmt.Sprintf("%s/%s", relativeGitPath, p.Path)
	//
	//				// finally this will return all files found
	//				for _, x := range operations.FindFiles(realPath, log) {
	//					fileList = append(fileList, x)
	//				}
	//			}
	//
	//			for _, f := range fileList {
	//				log.Infow("applying update policy",
	//					"file", f,
	//					"pattern", updatePolicy.PatternString,
	//					"blacklist", updatePolicy.BlackList,
	//				)
	//				fileChangesHappened := DoUpdate(f, updatePolicy, registryStrings, db, log)
	//				if fileChangesHappened > 0 {
	//					triggerCommitAndPush = true
	//					changeCount = changeCount + fileChangesHappened
	//					fileChangesHappened = 0
	//				}
	//			}
	//		}
	//
	//		if triggerCommitAndPush {
	//			msg := fmt.Sprintf("%s [%d]", appConfig.Global.GitMessage, changeCount)
	//			log.Infow("doing commit",
	//				"gitRepo", gitRepo.URL,
	//				"msg", msg,
	//			)
	//			git.CommitAndPush(gitRepo, appConfig.Global, msg, log)
	//		}
	//
	//		// TODO: use a Tick() instead of this Sleep()
	//		time.Sleep(10 * time.Second)
	//	}
	//
	//}

}
