package cmd

import (
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/call"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/key"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/query"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/tx"
)

var cfgPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "banjo",
	Short: "Theta wallet",
	Long:  `Theta wallet.`,
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

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", getDefaultConfigPath(), fmt.Sprintf("config path (default is %s)", getDefaultConfigPath()))

	rootCmd.AddCommand(key.KeyCmd)
	rootCmd.AddCommand(tx.TxCmd)
	rootCmd.AddCommand(query.QueryCmd)
	rootCmd.AddCommand(call.CallCmd)
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

func getDefaultConfigPath() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return path.Join(home, ".banjo")
}
