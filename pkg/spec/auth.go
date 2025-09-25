package spec

type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

// DockerAuth represents authentication info for a registry
type DockerAuth struct {
	Auth     string `json:"auth,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type AuthInfo struct {
	Username string
	Password string
	Registry string
}
