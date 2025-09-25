/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/ashupednekar/compose/pkg/charts"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
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
		// values, err := cmd.Flags().GetString("values")
		// if err != nil{
		// 	fmt.Printf("error getting chart flag: %s", err)
		// }
		c, err := charts.NewChartUtils()
		if err != nil{
			fmt.Printf("error initializing chart utils")
		}
		actionConfig := new(action.Configuration)
		if err := actionConfig.Init(nil, "", "secret", logDebug); err != nil{
			 fmt.Printf("error initiating action config")
		}
		actionConfig.RegistryClient = c.Client
		pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
		pull.Settings = cli.New()
		pull.DestDir = "."
	  if _, err := pull.Run(fmt.Sprintf("oci://%s/%s", registry, chart)); err != nil{
		  fmt.Printf("error pulling chart: %v", err)
	  }
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)

	refreshCmd.Flags().StringP("chart", "c", "chart", "chart repository")
  refreshCmd.Flags().StringP("registry", "r", "registry url", "registry url")

}

func logDebug(format string, v ...interface{}){
	fmt.Printf(format, v...)
}
