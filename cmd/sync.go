/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Update artifacts for a module.",
	Long: `
The sync command applies the refreshed artifacts at the module level.
It updates containers, configs, ingress, and volumes for the specified module, ensuring the compose artifacts are aligned with the upstream Helm release.

Usage:
compose sync <module> [version]

    module – Name of the module.
    version – Target version to sync. Defaults to $MODULE_VERSION.

Alto triggers activities like image pulls in the background... 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sync called")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
