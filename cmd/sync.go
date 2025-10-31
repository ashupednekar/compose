/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/ashupednekar/compose/pkg/charts"
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
		chart, err := cmd.Flags().GetString("chart")
		if err != nil{
			fmt.Printf("error getting chart flag: %s\n", err)
			return
		}
		setValues, err := cmd.Flags().GetStringSlice("set")
		if err != nil {
			fmt.Printf("error getting set flags: %s\n", err)
			return
		}
		valuesPath, err := cmd.Flags().GetString("values")
		if err != nil{
			fmt.Printf("error getting chart flag: %s\n", err)
			return
		}
		useHostNetwork, err := cmd.Flags().GetBool("useHostNetwork")
		if err != nil{
			fmt.Printf("error getting useHostNetwork flag: %s\n", err)
			return
		}
		insecureSkipTLSVerify, err := cmd.Flags().GetBool("insecure-skip-tls-verify")
		if err != nil {
			fmt.Printf("error getting insecure-skip-tls-verify flag: %s\n", err)
			return
		}
		cUtils, err := charts.NewChartUtils(insecureSkipTLSVerify)
		if err != nil{
			fmt.Printf("error initializing chart utils: %s\n", err)
		}
		apps, err := cUtils.Parse(chart, valuesPath, setValues, useHostNetwork)
		if err != nil{
			fmt.Printf("error parsing manifest: %v\n", err)
		}
		if err := charts.WriteCompose(apps, charts.ExtractName(chart)); err != nil {
				fmt.Printf("error writing docker compose: %s\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringP("chart", "c", "chart", "chart repository")
	syncCmd.Flags().StringP("values", "f", "values", "values path")
	syncCmd.Flags().Bool("useHostNetwork", false, "whether to use host network or not")
	syncCmd.Flags().Bool("insecure-skip-tls-verify", false, "skip tls verification for chart pulling")
	syncCmd.Flags().StringSliceP("set", "s", []string{}, "Set values on the command line (can specify multiple)")
}
