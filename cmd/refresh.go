/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// refreshCmd represents the refresh command
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Fetch the latest artifacts and show differences",
	Long: `
The refresh command pulls the latest Helm-based artifacts from the configured OCI repository.
It does not modify the local working copy but shows a diff of containers, configs, ingress, and volumes against the currently stored version.
This lets you preview changes before applying them with sync.

Usage:
compose refresh identity 6.2.4

module – Name of the module (e.g., identity, payments).
version – Specific version to fetch. Defaults to $MODULE_VERSION if not provided.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("refresh called")
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// refreshCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// refreshCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
