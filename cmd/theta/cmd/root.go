package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgPath string
var snapshotPath string
var chainImportDirPath string
var chainCorrectionPath string

var nodePassword string

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
	RootCmd.PersistentFlags().StringVar(&snapshotPath, "snapshot", "", "snapshot path")
	RootCmd.PersistentFlags().StringVar(&chainImportDirPath, "chain_import", "", "chain import path")
	RootCmd.PersistentFlags().StringVar(&chainCorrectionPath, "chain_correction", "", "chain correction path")
	//RootCmd.PersistentFlags().StringVar(&snapshotPath, "snapshot", getDefaultSnapshotPath(), fmt.Sprintf("snapshot path (default is %s)", getDefaultSnapshotPath()))
	RootCmd.PersistentFlags().StringVar(&nodePassword, "password", "", "password for the node")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(cfgPath)

	// Search config (without extension).
	viper.SetConfigName("config")

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
