package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage Eino Notebook configuration. Use subcommands to view, init, or validate config.`,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  `Display the current configuration values from .env file and environment variables.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Current Configuration:")
		fmt.Println("=====================")
		fmt.Printf("Env File: %s\n", viper.ConfigFileUsed())
		fmt.Printf("Server Host: %s\n", viper.GetString("SERVER_HOST"))
		fmt.Printf("Server Port: %d\n", viper.GetInt("SERVER_PORT"))
		fmt.Printf("Log Level: %s\n", viper.GetString("LOG_LEVEL"))
		fmt.Printf("Verbose: %v\n", viper.GetBool("verbose"))
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .env file",
	Long:  `Create a new .env file in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := ".env"

		// Check if file already exists
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("Env file '%s' already exists. Overwrite? (y/N): ", filename)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Create default .env file
		defaultEnv := `# Eino Notebook Environment Configuration

# Server Configuration
SERVER_HOST=localhost
SERVER_PORT=8080

# Logging
LOG_LEVEL=info

# Environment variables can be overridden by:
# 1. Command line flags (e.g., --port 9090)
# 2. System environment variables with EINO_ prefix (e.g., EINO_SERVER_PORT=9090)
`

		err := os.WriteFile(filename, []byte(defaultEnv), 0600)
		if err != nil {
			return fmt.Errorf("failed to create .env file: %w", err)
		}

		absPath, _ := filepath.Abs(filename)
		fmt.Printf("Environment file created at: %s\n", absPath)
		fmt.Println("You can edit this file or use environment variables to override settings.")
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  `Validate the current configuration and report any issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Validating configuration...")

		issues := 0

		// Check server port
		port := viper.GetInt("SERVER_PORT")
		if port < 1 || port > 65535 {
			fmt.Printf("❌ Invalid server port: %d (must be 1-65535)\n", port)
			issues++
		} else {
			fmt.Printf("✓ Server port: %d\n", port)
		}

		// Check log level
		validLogLevels := map[string]bool{
			"debug": true, "info": true, "warn": true, "error": true,
		}
		logLevel := viper.GetString("LOG_LEVEL")
		if !validLogLevels[logLevel] {
			fmt.Printf("❌ Invalid log level: %s\n", logLevel)
			issues++
		} else {
			fmt.Printf("✓ Log level: %s\n", logLevel)
		}

		// Check server host
		host := viper.GetString("SERVER_HOST")
		if host == "" {
			fmt.Println("❌ Server host is empty")
			issues++
		} else {
			fmt.Printf("✓ Server host: %s\n", host)
		}

		if issues > 0 {
			fmt.Printf("\n❌ Found %d issue(s)\n", issues)
			os.Exit(1)
		} else {
			fmt.Println("\n✓ Configuration is valid!")
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}
