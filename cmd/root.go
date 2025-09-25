/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)



// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "compose",
	Short: "Manage containers and artifacts in Docker or Podman environments",
	Long: `
Converts Helm chart artifacts (Deployments, ConfigMaps, StatefulSets, Ingress, etc.) into docker-compose YAML files and supporting configuration files.
It extracts containers, configs, ingress rules, volumes, and any environment interpolations from the Helm templates.
By default, it does not modify live deployments; it only generates artifacts locally for inspection, versioning, or later application using sync or apply.

This command is useful for clients who cannot run Helm/K8s directly and require a CD-like experience using docker-compose or podman-compose, while still leveraging the same upstream Helm artifacts.
	`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.compose.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


