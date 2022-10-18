package api

type K3sConfig struct {
	ClusterInit bool     `yaml:"cluster-init,omitempty"`
	Token       string   `yaml:"token,omitempty"`
	Server      string   `yaml:"server,omitempty"`
	TLSSan      []string `yaml:"tls-san,omitempty"`
}
