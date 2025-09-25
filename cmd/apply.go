/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply updated artifacts and restart services",
	Long: `
	The apply command takes the currently synced artifacts and applies them to the on-prem environment.
It ensures that the containers are restarted with the updated configurations, providing a CD-like workflow without manual edits.

Usage:
compose apply <module> [version]

    module – Name of the module.
    version – Version of artifacts to apply. Defaults to $MODULE_VERSION.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("apply called")
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// applyCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// applyCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
