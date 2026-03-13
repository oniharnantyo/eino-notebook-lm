package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "eino-notebook",
	Short: "Eino Notebook CLI - Manage your notebook operations",
	Long: `Eino Notebook is a production-ready CLI application for managing
notebook operations with support for multiple commands like serve, config,
and more. Built with Cobra and Viper for powerful CLI capabilities.`,
	Example: `  eino-notebook serve
  eino-notebook serve --port 8080
  eino-notebook config init
  eino-notebook version`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags are available to all subcommands
	rootCmd.PersistentFlags().StringVar(&cfgFile, "env", "", "env file (default is ./.env or ./configs/.env)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	// Bind flags to viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Local flags only available to root command
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in .env file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use env file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search .env in multiple locations (in order of priority)
		viper.AddConfigPath(".")           // Current directory
		viper.AddConfigPath("./configs")   // ./configs directory
		viper.AddConfigPath(home)          // Home directory
		viper.SetConfigName(".env")
		viper.SetConfigType("env")
	}

	// Set defaults
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("SERVER_PORT", 8080)
	viper.SetDefault("SERVER_HOST", "localhost")

	// Enable environment variable reading with prefix
	viper.SetEnvPrefix("EINO")
	viper.AutomaticEnv()

	// Read .env file if found (silent fail if not found)
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using env file:", viper.ConfigFileUsed())
	}
}
