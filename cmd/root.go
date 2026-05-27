package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chenhuijun/op-cli/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var client api.APIClient

// SetClient allows tests to inject a mock client.
func SetClient(c api.APIClient) {
	client = c
}

var rootCmd = &cobra.Command{
	Use:   "op",
	Short: "OpenProject CLI for team leads",
	Long:  "A lean CLI to manage sprints, backlogs, and work packages in OpenProject.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip client init for config-only commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		url := viper.GetString("url")
		apiKey := viper.GetString("api_key")
		project := viper.GetString("project")

		if url == "" || apiKey == "" {
			return fmt.Errorf("missing config: set OP_URL and OP_API_KEY in ~/.oprc or environment")
		}

		client = api.NewClient(url, apiKey, project)
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("project", "p", "", "OpenProject project identifier")
	_ = viper.BindPFlag("project", rootCmd.PersistentFlags().Lookup("project"))
}

func initConfig() {
	// Environment variables: OP_URL, OP_API_KEY, OP_PROJECT
	viper.SetEnvPrefix("OP")
	viper.AutomaticEnv()

	// Config file: ~/.oprc (YAML)
	home, err := os.UserHomeDir()
	if err == nil {
		viper.SetConfigName(".oprc")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is OK — env vars might be set
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Warning: error reading config: %s\n", err)
		}
	}

	// Also support ~/.oprc as path directly
	oprcPath := filepath.Join(home, ".oprc")
	if _, err := os.Stat(oprcPath); err == nil {
		viper.SetConfigFile(oprcPath)
		_ = viper.MergeInConfig()
	}
}
