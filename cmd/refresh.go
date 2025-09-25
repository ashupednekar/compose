/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/ashupednekar/compose/pkg/charts"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
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
		valuesPath, err := cmd.Flags().GetString("values")
		if err != nil{
			fmt.Printf("error getting chart flag: %s", err)
		}
		cUtils, err := charts.NewChartUtils()
		if err != nil{
			fmt.Printf("error initializing chart utils")
		}
		actionConfig := new(action.Configuration)
		if err := actionConfig.Init(nil, "", "secret", logDebug); err != nil{
			 fmt.Printf("error initiating action config")
		}
		actionConfig.RegistryClient = cUtils.Client


		pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
		pull.Settings = cli.New()
		pull.DestDir = "/tmp"
		pull.Untar = false
		pull.Version = "*"
		chartPath, err := pull.Run(fmt.Sprintf("oci://%s/%s", registry, chart))
		if err != nil{
		 fmt.Printf("error pulling chart: %v\n", err)
		}
		pulledChart, err := loader.Load(chartPath)
	
		renderer := action.NewInstall(&action.Configuration{})
		renderer.ClientOnly = true
		renderer.DryRun = true
		renderer.ReleaseName = chart
		renderer.Namespace = "default"
		renderer.DisableHooks = true

		values, err := unmarshalWithOverride(valuesPath)
		if err != nil {
        fmt.Printf("failed to unmarshal YAML: %v", err)
    }
		
		rel, err := renderer.Run(pulledChart, values)
		if err != nil{
			fmt.Printf("error templating chart: %v\n", err)
		}

		fmt.Printf("%v", rel.Manifest)

	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)

  refreshCmd.Flags().StringP("registry", "r", "registry url", "registry url")
	refreshCmd.Flags().StringP("chart", "c", "chart", "chart repository")
	refreshCmd.Flags().StringP("values", "f", "values", "values path")

}

func logDebug(format string, v ...interface{}){
	fmt.Printf(format, v...)
}

// mergeMap merges key-value pairs from src into dst, overriding duplicates
func mergeMap(dst, src map[string]interface{}) {
    for k, v := range src {
        // if both are maps, merge recursively
        if vMap, ok := v.(map[string]interface{}); ok {
            if dstMap, ok := dst[k].(map[string]interface{}); ok {
                mergeMap(dstMap, vMap)
                continue
            }
        }
        // otherwise override
        dst[k] = v
    }
}

// unmarshalWithOverride parses YAML and allows duplicates by overriding
func unmarshalWithOverride(filename string) (map[string]interface{}, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    // Parse into yaml.Node first
    var root yaml.Node
    if err := yaml.Unmarshal(data, &root); err != nil {
        return nil, err
    }

    result := make(map[string]interface{})

    // Walk through the document manually
    if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
        if root.Content[0].Kind == yaml.MappingNode {
            for i := 0; i < len(root.Content[0].Content); i += 2 {
                key := root.Content[0].Content[i].Value
                var value interface{}
                if err := root.Content[0].Content[i+1].Decode(&value); err != nil {
                    return nil, err
                }
                // override duplicates
                result[key] = value
            }
        }
    }

    return result, nil
}

