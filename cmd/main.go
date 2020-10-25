package cmd

import (
	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/git"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
	"net/http"
	"strings"
)

var fileList []string

var rootCmd = &cobra.Command{
	Use:   "run",
	Short: "launchs the laminar service",
	Long: `This is main laminar service and API

Laminar is a GitOps utility for automating the promotion of docker images in git.

.. turbulence is bad.. use laminar flow`,
	Run: func(cmd *cobra.Command, args []string) {
		runLaminar()
	},
}

// decide if we want the webserver to log or not here
func skipPath(c echo.Context) bool {
	ignored := []string{
		"/metrics",
		"/healthz",
	}

	for _, x := range ignored {
		if strings.HasPrefix(c.Path(), x) {
			return true
		}
	}
	return false
}

// DaemonStart is the main entrypoint for laminar
func runLaminar() {

	// start a logger.
	log, _ := startLogger(debug)

	// infer config
	appConfig := cfg.LoadConfig(configFile, log)

	//db := cache.Open(configCache, log)
	//log.Debug("opened db: ", configCache)

	for _, r := range appConfig.GitRepos {
		go git.InitialGitCloneAndCheckout(r, log)
	}

	// start a new echo instance
	e := echo.New()

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)

	// hide banner and port messages at launch (we'll use a the regular logger to ensure consistency)
	e.HideBanner = true
	e.HidePort = true

	// give the logger a "skipper" method (lets us decide if we want to ignore logging for certain contexts)
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		// https://echo.labstack.com/middleware/logger
		Skipper: skipPath,
	}))

	e.GET("/hello", routeHello)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// start the webserver (blocks here)
	log.Info("starting webserver")
	e.Logger.Fatal(e.Start(":1313"))

}

func routeHello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
