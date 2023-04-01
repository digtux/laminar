package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	configFile    string // configuration file that laminar will read
	debug         bool   // True enables verbose and human friendly logging
	configCache   string // buntDB cache.. either "/something.db" or ":memory"
	oneShot       bool   // if laminar should just run once and terminate
	interval      time.Duration
	pauseDuration time.Duration // when laminar is asked to pause, it'll hold off on any operations for this long
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
	rootCmd.PersistentFlags().DurationVar(&interval, "interval", 1*time.Minute, "interval between laminar git polls. EG: 20s, 2m, 5m, 1h20m")
	rootCmd.PersistentFlags().DurationVar(&pauseDuration, "pause", 5*time.Minute, "default pause duration. EG: 20s, 1m, 5m, 1h")
	flagSet := rootCmd.Flags()
	flagSet.BoolVarP(&debug, "debug", "D", false, "enable debug logging")
	flagSet.BoolVarP(&oneShot, "one-shot", "o", false, "only run laminar once (not as a persistent service)")
}
