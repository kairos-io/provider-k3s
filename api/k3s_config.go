package api

type K3sServerConfig struct {
	// ClusterInit must not have an omitempty tag, otherwise it is elided by JSON/YAML encoders when false.
	ClusterInit       bool     `yaml:"cluster-init" json:"cluster-init"`
	Token             string   `yaml:"token,omitempty" json:"token,omitempty"`
	Server            string   `yaml:"server,omitempty" json:"server,omitempty"`
	TLSSan            []string `yaml:"tls-san,omitempty" json:"tls-san,omitempty"`
	DatastoreEndpoint string   `yaml:"datastore-endpoint,omitempty" json:"datastore-endpoint,omitempty"`
	DatastoreCaFile   string   `yaml:"datastore-cafile,omitempty" json:"datastore-cafile,omitempty"`
	DatastoreCertFile string   `yaml:"datastore-certfile,omitempty" json:"datastore-certfile,omitempty"`
	DatastoreKeyFile  string   `yaml:"datastore-keyfile,omitempty" json:"datastore-keyfile,omitempty"`
}
