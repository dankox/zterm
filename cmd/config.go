// Copyright (c) 2020 dankox
// This code is licensed under MIT license (see LICENSE for details)

package cmd

import (
	"fmt"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var configCmd = &cobra.Command{
	Use:   "config [host]",
	Short: "add/update configuration file",
	Long:  `Add or update configuration file with provided arguments.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			fmt.Printf("config '%v'", strings.Join(args, ","))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		writeConfig()
		fmt.Println("Config command executed!")
	},
}

func init() {
	configCmd.Flags().String("host", "", "host name of the remote server")
	viper.BindPFlag("server.host", configCmd.Flags().Lookup("host"))
	configCmd.Flags().String("user", "", "user name used to connect to the remote server")
	viper.BindPFlag("server.user", configCmd.Flags().Lookup("user"))
	configCmd.Flags().IntP("refresh-interval", "r", 5, "refresh interval in seconds used to get new data (default: 5s)")
	viper.BindPFlag("server.refresh", configCmd.Flags().Lookup("refresh-interval"))
}

// initConfig reads in config file and ENV variables if set.
func writeConfig() {
	// TODO: not working
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		viper.WriteConfig()
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
		viper.SetConfigName(".zmonitor")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv() // read in environment variables that match
}
