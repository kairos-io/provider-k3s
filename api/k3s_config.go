package api

type K3sServerConfig struct {
	ClusterInit       bool     `yaml:"cluster-init,omitempty"`
	DatastoreEndpoint string   `yaml:"datastore-endpoint,omitempty"`
	DatastoreCaFile   string   `yaml:"datastore-cafile,omitempty"`
	DatastoreCertFile string   `yaml:"datastore-certfile,omitempty"`
	DatastoreKeyFile  string   `yaml:"datastore-keyfile,omitempty"`
	Token             string   `yaml:"token,omitempty"`
	Server            string   `yaml:"server,omitempty"`
	TLSSan            []string `yaml:"tls-san,omitempty"`
}
