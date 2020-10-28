// Copyright (c) 2020 dankox
// This code is licensed under MIT license (see LICENSE for details)

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dankox/zmonitor-go/monitor"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "zmonitor-go",
	Short: "Monitoring tool for z/OS systems",
	Long: `zMonitor is a TUI monitoring tool for z/OS systems.

Displays system logs, jobs, activity and user defined commands in semi-online
mode, which refreshes data obtained from the remote system in intervals.
This way user has live representation of the activity on z/OS system.
(Access to the systems is done thru ssh protocol.)`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			viper.Set("server.host", strings.Join(args, ","))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		monitor.Main()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.zmonitor or ~/.zmonitor)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().String("host", "", "host name of the remote server")
	// viper.BindPFlag("server.host", rootCmd.Flags().Lookup("host"))
	viper.SetDefault("server.host", "")

	rootCmd.Flags().String("user", "", "user name used to connect to the remote server")
	viper.BindPFlag("server.user", rootCmd.Flags().Lookup("user"))

	rootCmd.Flags().IntP("refresh-interval", "r", 5, "refresh interval in seconds used to get new data (default: 5s)")
	viper.BindPFlag("server.refresh", rootCmd.Flags().Lookup("refresh-interval"))

	rootCmd.AddCommand(configCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in current directory and if not found in home directory
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigName(".zmonitor") // should look for different extensions
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// TODO: add log?
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		// no config file? don't care
	}
}
