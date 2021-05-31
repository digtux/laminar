package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"time"
)

var (
	configFile  string // configuration file that laminar will read
	debug       bool   // True enables verbose and human friendly logging
	configCache string // buntDB cache.. either "/something.db" or ":memory"
	oneShot     bool   // if laminar should just run once and terminate
	interval    time.Duration
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.yaml", "yaml config file")
	rootCmd.PersistentFlags().StringVar(&configCache, "cache", ":memory:", "cache location.. EG  'cache.db'")
	rootCmd.PersistentFlags().DurationVar(&interval, "interval", 1*time.Minute, "interval between laminar to updating git EG: 20s, 2m, 5m, 1h20m")

	flagSet := rootCmd.Flags()
	flagSet.BoolVarP(&debug, "debug", "D", false, "enable debug logging")
	flagSet.BoolVarP(&oneShot, "one-shot", "o", false, "only run laminar once (not as a persistent service)")

}
