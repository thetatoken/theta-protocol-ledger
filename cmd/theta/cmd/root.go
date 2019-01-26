package cmd

import (
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgPath string
var snapshotPath string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "theta",
	Short: "Theta",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgPath, "config", getDefaultConfigPath(), fmt.Sprintf("config path (default is %s)", getDefaultConfigPath()))
	RootCmd.PersistentFlags().StringVar(&snapshotPath, "snapshot", getDefaultSnapshotPath(), fmt.Sprintf("snapshot path (default is %s)", getDefaultSnapshotPath()))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(cfgPath)

	// Search config (without extension).
	viper.SetConfigName("config")

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// getDefaultConfigPath returns the default config path.
func getDefaultConfigPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return path.Join(home, ".theta")
}

func getDefaultSnapshotPath() string {
	return path.Join(getDefaultConfigPath(), "genesis")
}
