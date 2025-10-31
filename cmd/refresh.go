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
		chart, err := cmd.Flags().GetString("chart")
		if err != nil{
			fmt.Printf("error getting chart flag: %s", err)
		}
		valuesPath, err := cmd.Flags().GetString("values")
		if err != nil{
			fmt.Printf("error getting chart flag: %s", err)
		}
		setValues, err := cmd.Flags().GetStringSlice("set")
		if err != nil {
			fmt.Printf("error getting set flags: %s\n", err)
			return
		}
		insecureSkipTLSVerify, err := cmd.Flags().GetBool("insecure-skip-tls-verify")
		if err != nil {
			fmt.Printf("error getting insecure-skip-tls-verify flag: %s\n", err)
			return
		}
		cUtils, err := charts.NewChartUtils(insecureSkipTLSVerify)
		if err != nil{
			fmt.Printf("error initializing chart utils")
		}
		rel, err := cUtils.Template(chart, valuesPath, setValues)
		if err != nil{
			fmt.Printf("error templating chart: %v\n", err)
		}
		fmt.Printf("%s\n", rel.Manifest)
		
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().StringP("chart", "c", "chart", "chart repository")
	refreshCmd.Flags().StringP("values", "f", "values", "values path")
	refreshCmd.Flags().Bool("insecure-skip-tls-verify", false, "skip tls verification for chart pulling")
	refreshCmd.Flags().StringSliceP("set", "s", []string{}, "Set values on the command line (can specify multiple)")
}

