package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile  string // configuration file that laminar will read
	debug       bool   // True enables verbose and human friendly logging
	configCache string // bundDB cache.. either "/something.db" or ":memory"
	//developer  string
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize()

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.yaml", "config file (default is 'config.yaml')")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	//rootCmd.PersistentFlags().StringVar(&developer, "developer", "Unknown Developer!", "Developer name.")
	rootCmd.PersistentFlags().StringVar(&configCache, "cache", "cache.db", "cache location.. also supports ':memory:'")

	flagSet := rootCmd.Flags()
	flagSet.BoolVarP(&debug, "debug", "D", false, "enable debug logging")
}
