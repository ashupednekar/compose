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

		chartPath := fmt.Sprintf("/tmp/%s", chart)
		os.RemoveAll(chartPath)
		pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
		pull.Settings = cli.New()
		pull.DestDir = "/tmp"
		pull.Untar = true 
		pull.Version = "*"
		if _, err := pull.Run(fmt.Sprintf("oci://%s/%s", registry, chart)); err != nil{
		 fmt.Printf("error pulling chart: %v\n", err)
		}
		fmt.Printf("chart pulled at: %s\n", chartPath)
		pulledChart, err := loader.Load(chartPath)
		if err != nil{
			fmt.Printf("error loading chart: %v\n", err)
		}
	
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

		fmt.Printf("v: %v\n", values)
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



func unmarshalWithOverride(filename string) (map[string]interface{}, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
    }

    var root yaml.Node
    if err := yaml.Unmarshal(data, &root); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    result := make(map[string]interface{})
    
    // Handle the document structure
    if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
        if err := parseNode(root.Content[0], result); err != nil {
            return nil, err
        }
    }
    
    return result, nil
}

func parseNode(node *yaml.Node, result map[string]interface{}) error {
    switch node.Kind {
    case yaml.MappingNode:
        return parseMappingNode(node, result)
    case yaml.SequenceNode:
        // Handle sequences if needed
        var seq []interface{}
        for _, item := range node.Content {
            var value interface{}
            if err := item.Decode(&value); err != nil {
                return err
            }
            seq = append(seq, value)
        }
        // This case shouldn't occur at root level for typical YAML files
        return fmt.Errorf("unexpected sequence at root level")
    default:
        return fmt.Errorf("unexpected node type: %v", node.Kind)
    }
}

func parseMappingNode(node *yaml.Node, result map[string]interface{}) error {
    if len(node.Content)%2 != 0 {
        return fmt.Errorf("invalid mapping node: odd number of elements")
    }

    for i := 0; i < len(node.Content); i += 2 {
        keyNode := node.Content[i]
        valueNode := node.Content[i+1]
        
        if keyNode.Kind != yaml.ScalarNode {
            return fmt.Errorf("non-scalar key found at line %d", keyNode.Line)
        }
        
        key := keyNode.Value
        
        // Handle different value types
        var value interface{}
        switch valueNode.Kind {
        case yaml.MappingNode:
            // Nested mapping
            nestedMap := make(map[string]interface{})
            if err := parseMappingNode(valueNode, nestedMap); err != nil {
                return err
            }
            value = nestedMap
        case yaml.SequenceNode:
            // Array/slice
            var seq []interface{}
            for _, item := range valueNode.Content {
                var itemValue interface{}
                if err := item.Decode(&itemValue); err != nil {
                    return err
                }
                seq = append(seq, itemValue)
            }
            value = seq
        default:
            // Scalar value (string, number, boolean, etc.)
            if err := valueNode.Decode(&value); err != nil {
                return err
            }
        }
        
        // Override any previous value with the same key
        // This is where duplicate keys are handled - last one wins
        if _, exists := result[key]; exists {
            fmt.Printf("Warning: Overriding duplicate key '%s' (previous definition ignored)\n", key)
        }
        result[key] = value
    }
    
    return nil
}
