package charts

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
)

func (utils *ChartUtils) Template(chart string, valuesPath string, setValues []string) (*release.Release, error){
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(cli.New().RESTClientGetter(), "", "secret", logDebug); err != nil{
		 fmt.Printf("error initiating action config")
	}
	actionConfig.RegistryClient = utils.Client
	name := ExtractName(chart)
	os.RemoveAll(name)
	chartPath := fmt.Sprintf("/tmp/%s", name)
	pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
	pull.Settings = cli.New()
	pull.DestDir = "/tmp"
	pull.Untar = true 
	pull.Version = "*"
	if _, err := pull.Run(chart); err != nil{
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
	renderer.ReleaseName = ExtractName(chart)
	renderer.Namespace = "default"
	renderer.DisableHooks = true

	values, err := unmarshalWithOverride(valuesPath)
	//TODO: use setValues to add/override stuff

	if err != nil {
      return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
  }

	rel, err := renderer.Run(pulledChart, values)
	if err != nil{
		return nil, fmt.Errorf("error templating chart: %v\n", err)
	}
	return rel, nil
}


func logDebug(format string, v ...interface{}){
	fmt.Printf(format, v...)
}


func unmarshalWithOverride(filename string) (map[string]interface{}, error) {
		_, err := os.Stat(filename)
		if err != nil && os.IsNotExist(err){
			return make(map[string]interface{}), nil
		}

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
            nestedMap := make(map[string]interface{})
            if err := parseMappingNode(valueNode, nestedMap); err != nil {
                return err
            }
            value = nestedMap
        case yaml.SequenceNode:
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
            if err := valueNode.Decode(&value); err != nil {
                return err
            }
        }
        
        if _, exists := result[key]; exists {
            //fmt.Printf("Warning: Overriding duplicate key '%s' (previous definition ignored)\n", key)
        }
        result[key] = value
    }
    
    return nil
}

func ExtractName(ref string) string {
	u, err := url.Parse(ref)
	var path string
	if err == nil && u.Scheme != "" && u.Path != "" {
		// e.g. oci://host/repo/chart:tag
		path = u.Path
	} else {
		// fallback: treat the whole string as a path
		path = ref
	}
	// Trim leading slashes and split into segments
	path = strings.TrimLeft(path, "/")
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return ""
	}
	last := segments[len(segments)-1]
	// Remove any :tag or @digest suffix
	if i := strings.IndexAny(last, ":@"); i >= 0 {
		last = last[:i]
	}
	return last
}

