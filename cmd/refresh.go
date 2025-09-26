/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/ashupednekar/compose/pkg/charts"
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
compose refresh 

	`,
	Run: func(cmd *cobra.Command, args []string) {
		registry, err := cmd.Flags().GetString("registry")
		if err != nil{
			fmt.Printf("error getting registry flag: %s", err)
		}
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
		rel, err := cUtils.Template(registry, chart, valuesPath)
		if err != nil{
			fmt.Printf("error templating chart: %v\n", err)
		}
		apps, err := cUtils.Parse(rel.Manifest)
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
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
  refreshCmd.Flags().StringP("registry", "r", "registry url", "registry url")
	refreshCmd.Flags().StringP("chart", "c", "chart", "chart repository")
	refreshCmd.Flags().StringP("values", "f", "values", "values path")

}

