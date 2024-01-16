package constants

const (
	EnableOpenRCServices  = "Enable OpenRC Services"
	EnableSystemdServices = "Enable Systemd Services"
	InstallK3sConfigFiles = "Install K3s Configuration Files"
	ImportK3sImages       = "Import K3s Images"
)

// ProviderOptions are keys provider-k3s supports if present in Cluster.ProviderOptions from the Kairos SDK.
type ProviderOption string

const (
	// If value == 'yes', provider-k3s will use etcd for its datastore.
	// If value == 'no' and DatastoreEndpoint is not defined, the sqlite datastore will be used.
	// If value == 'no' and DatastoreEndpoint is defined, a custom datastore will be used.
	ClusterInit ProviderOption = "cluster-init"

	// A PostgreSQL, MySQL, NATS, or etcd connection string. Used to describe the connection to the datastore.
	DatastoreEndpoint ProviderOption = "datastore-endpoint"

	// TLS Certificate Authority (CA) file used to help secure communication with the datastore.
	DatastoreCaFile ProviderOption = "datastore-cafile"

	// TLS certificate file used for client certificate based authentication to your datastore.
	DatastoreCertFile ProviderOption = "datastore-certfile"

	// TLS key file used for client certificate based authentication to your datastore.
	DatastoreKeyFile ProviderOption = "datastore-keyfile"
)
