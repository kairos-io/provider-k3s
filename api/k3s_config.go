package api

type K3sServerConfig struct {
	ClusterInit       bool     `yaml:"cluster-init,omitempty" json:"cluster-init,omitempty"`
	Token             string   `yaml:"token,omitempty" json:"token,omitempty"`
	Server            string   `yaml:"server,omitempty" json:"server,omitempty"`
	TLSSan            []string `yaml:"tls-san,omitempty" json:"tls-san,omitempty"`
	DatastoreEndpoint string   `yaml:"datastore-endpoint,omitempty" json:"datastore-endpoint,omitempty"`
	DatastoreCaFile   string   `yaml:"datastore-cafile,omitempty" json:"datastore-cafile,omitempty"`
	DatastoreCertFile string   `yaml:"datastore-certfile,omitempty" json:"datastore-certfile,omitempty"`
	DatastoreKeyFile  string   `yaml:"datastore-keyfile,omitempty" json:"datastore-keyfile,omitempty"`
	BindAddress       string   `yaml:"bind-address,omitempty" json:"bind-address,omitempty"`
}
