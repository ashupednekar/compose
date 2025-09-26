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
			fmt.Printf("error getting chart flag: %s", err)
		}
		valuesPath, err := cmd.Flags().GetString("values")
		if err != nil{
			fmt.Printf("error getting chart flag: %s", err)
		}
		cUtils, err := charts.NewChartUtils()
		if err != nil{
			fmt.Printf("error initializing chart utils")
		}
		apps, err := cUtils.Parse(chart, valuesPath)
		if err != nil{
			fmt.Printf("error parsing manifest")
		}
		for _, app := range apps{
			fmt.Printf("Name: %v\n", app.Name)
			fmt.Printf("Image: %v\n", app.Image)
			fmt.Printf("Command: %v\n", app.Command)
			fmt.Printf("Envs: %v\n", app.Configs)
			fmt.Printf("PostStart: %v\n===", app.PostStart)	
		}
		if err := charts.WriteCompose(apps, charts.ExtractName(chart)); err != nil {
				fmt.Printf("error writing docker compose: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringP("chart", "c", "chart", "chart repository")
	syncCmd.Flags().StringP("values", "f", "values", "values path")
}
