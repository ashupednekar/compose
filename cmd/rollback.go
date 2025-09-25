/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// rollbackCmd represents the rollback command
var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Revert to a previous artifact version",
	Long: `
The rollback command reverts a module’s artifacts to a previously synced version.
Versions are tracked either in a local Git repository or in a lightweight SQLite index.
This ensures safe recovery from failed updates.

Usage:
compose rollback <module> [version]

    module – Name of the module.
    version – Target version to roll back to. Defaults to $MODULE_VERSION.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("rollback called")
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rollbackCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rollbackCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
