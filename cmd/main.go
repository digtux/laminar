package cmd

import (
	"net/http"
	"strings"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/digtux/laminar/pkg/git"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
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

	f := echo.New()
	// f.HideBanner = true
	// f.HidePort = true
	f.GET("/hello", routeHello)

	p := prometheus.NewPrometheus("laminar", nil)
	p.Use(f)

	// give the logger a "skipper" method
	// this lets us decide to ignore logging certain endpoints
	f.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: skipPath,
	}))

	// start the webserver (blocks here)
	log.Infow("starting webserver",
		"port", appConfig.Global.Listener,
	)

	f.Logger.Fatal(f.Start(appConfig.Global.Listener))

}

func routeHello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
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
