package spec

//--kubernetes respources--

type App struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Image     string            `json:"image"`
	Command   []string          `json:"command,omitempty"`
	PostStart *PostStartHook    `json:"postStart,omitempty"`
	Configs   map[string]string `json:"configs"`
	Mounts    map[string]string `json:"mounts"` 
}

type PostStartHook struct {
	Type    string   `json:"type"`
	Command []string `json:"command,omitempty"`
	HTTPGet string   `json:"httpGet,omitempty"`
}
type Resource struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   map[string]interface{} `yaml:"metadata"`
	Spec       map[string]interface{} `yaml:"spec,omitempty"`
	Data       map[string]interface{} `yaml:"data,omitempty"`
}

type EnvFrom struct {
	ConfigMapRef *ConfigMapRef `yaml:"configMapRef,omitempty"`
}

type ConfigMapRef struct {
	Name string `yaml:"name"`
}

type EnvVar struct {
	Name      string         `yaml:"name"`
	Value     string         `yaml:"value,omitempty"`
	ValueFrom *EnvVarSource  `yaml:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	ConfigMapKeyRef *ConfigMapKeyRef `yaml:"configMapKeyRef,omitempty"`
}

type ConfigMapKeyRef struct {
	Name string `yaml:"name"`
	Key  string `yaml:"key"`
}

type VolumeMount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
}

type Volume struct {
	Name      string                 `yaml:"name"`
	ConfigMap *VolumeConfigMapSource `yaml:"configMap,omitempty"`
}

type VolumeConfigMapSource struct {
	Name string `yaml:"name"`
}

type Lifecycle struct {
	PostStart *LifecycleHandler `yaml:"postStart,omitempty"`
}

type LifecycleHandler struct {
	Exec    *ExecAction    `yaml:"exec,omitempty"`
	HTTPGet *HTTPGetAction `yaml:"httpGet,omitempty"`
}

type ExecAction struct {
	Command []string `yaml:"command"`
}

type HTTPGetAction struct {
	Path string `yaml:"path"`
	Port string `yaml:"port"`
}

//--docker-compose respources--

type DockerComposeService struct{
	Image string `yaml:"image"`
  Command     []string          `yaml:"command,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	NetworkMode string `yaml:"network_mode,omitempty"`
}

type DockerCompose struct{
	Services map[string]DockerComposeService `yaml:"services"`
	Networks map[string]interface{} `yaml:"networks"`
}

