package api

import "time"

var StringListKeys = []string{
	"tls-san",
	"cluster-cidr",
	"service-cidr",
	"cluster-dns",
	"node-label",
	"node-taint",
	"kube-apiserver-arg",
	"etcd-arg",
	"kube-controller-manager-arg",
	"kube-scheduler-arg",
	"kube-cloud-controller-manager-arg",
	"disable",
	"airgap-extra-registry",
	"node-ip",
	"node-external-ip",
	"node-internal-dns",
	"node-external-dns",
	"kubelet-arg",
	"kube-proxy-arg",
	"kube-controller-arg",
	"kube-cloud-controller-arg",
}

type K3sServerConfig struct {
	ConfigFile                     string        `json:"config,omitempty" yaml:"config,omitempty"`
	Debug                          bool          `yaml:"debug,omitempty" json:"debug,omitempty"`
	V                              int           `yaml:"v,omitempty" json:"v,omitempty"`
	VModule                        string        `yaml:"vmodule,omitempty" json:"vmodule,omitempty"`
	Log                            string        `yaml:"log,omitempty" json:"log,omitempty"`
	AlsoLogToStderr                bool          `yaml:"alsologtostderr,omitempty" json:"alsologtostderr,omitempty"`
	BindAddress                    string        `yaml:"bind-address,omitempty" json:"bind-address,omitempty"`
	HTTPSListenPort                int           `yaml:"https-listen-port,omitempty" json:"https-listen-port,omitempty"`
	SupervisorPort                 int           `yaml:"supervisor-port,omitempty" json:"supervisor-port,omitempty"`
	ApiServerPort                  int           `yaml:"apiserver-port,omitempty" json:"apiserver-port,omitempty"`
	ApiServerBindAddress           string        `yaml:"apiserver-bind-address,omitempty" json:"apiserver-bind-address,omitempty"`
	AdvertiseAddress               string        `yaml:"advertise-address,omitempty" json:"advertise-address,omitempty"`
	AdvertisePort                  int           `yaml:"advertise-port,omitempty" json:"advertise-port,omitempty"`
	TLSSan                         []string      `yaml:"tls-san,omitempty" json:"tls-san,omitempty"`
	TLSSanSecurity                 bool          `yaml:"tls-san-security,omitempty" json:"tls-san-security,omitempty"`
	DataDir                        string        `yaml:"data-dir,omitempty" json:"data-dir,omitempty"`
	ClusterCIDR                    []string      `yaml:"cluster-cidr,omitempty" json:"cluster-cidr,omitempty"`
	ServiceCIDR                    []string      `yaml:"service-cidr,omitempty" json:"service-cidr,omitempty"`
	ServiceNodePortRange           string        `yaml:"service-node-port-range,omitempty" json:"service-node-port-range,omitempty"`
	ClusterDNS                     []string      `yaml:"cluster-dns,omitempty" json:"cluster-dns,omitempty"`
	ClusterDomain                  string        `yaml:"cluster-domain,omitempty" json:"cluster-domain,omitempty"`
	FlannelBackend                 string        `yaml:"flannel-backend,omitempty" json:"flannel-backend,omitempty"`
	FlannelIPv6Masq                bool          `yaml:"flannel-ipv6-masq,omitempty" json:"flannel-ipv6-masq,omitempty"`
	FlannelExternalIP              bool          `yaml:"flannel-external-ip,omitempty" json:"flannel-external-ip,omitempty"`
	EgressSelectorMode             string        `json:"egress-selector-mode,omitempty" yaml:"egress-selector-mode,omitempty"`
	ServiceLBNamespace             string        `yaml:"servicelb-namespace,omitempty" json:"servicelb-namespace,omitempty"`
	WriteKubeconfig                string        `yaml:"write-kubeconfig,omitempty" json:"write-kubeconfig,omitempty"`
	WriteKubeconfigMode            string        `yaml:"write-kubeconfig-mode,omitempty" json:"write-kubeconfig-mode,omitempty"`
	WriteKubeconfigGroup           string        `yaml:"write-kubeconfig-group,omitempty" json:"write-kubeconfig-group,omitempty"`
	HelmJobImage                   string        `yaml:"helm-job-image,omitempty" json:"helm-job-image,omitempty"`
	Token                          string        `yaml:"token,omitempty" json:"token,omitempty"`
	TokenFile                      string        `yaml:"token-file,omitempty" json:"token-file,omitempty"`
	AgentToken                     string        `yaml:"agent-token,omitempty" json:"agent-token,omitempty"`
	AgentTokenFile                 string        `yaml:"agent-token-file,omitempty" json:"agent-token-file,omitempty"`
	Server                         string        `yaml:"server,omitempty" json:"server,omitempty"`
	ClusterInit                    bool          `yaml:"cluster-init,omitempty" json:"cluster-init,omitempty"`
	ClusterReset                   bool          `yaml:"cluster-reset,omitempty" json:"cluster-reset,omitempty"`
	ClusterResetRestorePath        string        `yaml:"cluster-reset-restore-path,omitempty" json:"cluster-reset-restore-path,omitempty"`
	KubeApiServerArg               []string      `yaml:"kube-apiserver-arg,omitempty" json:"kube-apiserver-arg,omitempty"`
	EtcdArg                        []string      `yaml:"etcd-arg,omitempty" json:"etcd-arg,omitempty"`
	KubeControllerManagerArg       []string      `yaml:"kube-controller-manager-arg,omitempty" json:"kube-controller-manager-arg,omitempty"`
	KubeSchedulerArg               []string      `yaml:"kube-scheduler-arg,omitempty" json:"kube-scheduler-arg,omitempty"`
	KubeCloudControllerManagerArg  []string      `yaml:"kube-cloud-controller-manager-arg,omitempty" json:"kube-cloud-controller-manager-arg,omitempty"`
	KineTLS                        bool          `yaml:"kine-tls,omitempty" json:"kine-tls,omitempty"`
	DatastoreEndpoint              string        `yaml:"datastore-endpoint,omitempty" json:"datastore-endpoint,omitempty"`
	DatastoreCaFile                string        `yaml:"datastore-cafile,omitempty" json:"datastore-cafile,omitempty"`
	DatastoreCertFile              string        `yaml:"datastore-certfile,omitempty" json:"datastore-certfile,omitempty"`
	DatastoreKeyFile               string        `yaml:"datastore-keyfile,omitempty" json:"datastore-keyfile,omitempty"`
	EtcdExposeMetrics              bool          `yaml:"etcd-expose-metrics,omitempty" json:"etcd-expose-metrics,omitempty"`
	EtcdDisableSnapshots           bool          `yaml:"etcd-disable-snapshots,omitempty" json:"etcd-disable-snapshots,omitempty"`
	EtcdSnapshotName               string        `yaml:"etcd-snapshot-name,omitempty" json:"etcd-snapshot-name,omitempty"`
	EtcdSnapshotScheduleCron       string        `yaml:"etcd-snapshot-schedule-cron,omitempty" json:"etcd-snapshot-schedule-cron,omitempty"`
	EtcdSnapshotRetention          int           `yaml:"etcd-snapshot-retention,omitempty" json:"etcd-snapshot-retention,omitempty"`
	EtcdSnapshotDir                string        `yaml:"etcd-snapshot-dir,omitempty" json:"etcd-snapshot-dir,omitempty"`
	EtcdSnapshotCompress           bool          `yaml:"etcd-snapshot-compress,omitempty" json:"etcd-snapshot-compress,omitempty"`
	EtcdS3                         bool          `json:"etcd-s3,omitempty" yaml:"etcd-s3,omitempty"`
	EtcdS3Endpoint                 string        `json:"etcd-s3-endpoint,omitempty" yaml:"etcd-s3-endpoint,omitempty"`
	EtcdS3EndpointCA               string        `json:"etcd-s3-endpoint-ca,omitempty" yaml:"etcd-s3-endpoint-ca,omitempty"`
	EtcdS3SkipSSLVerify            bool          `json:"etcd-s3-skip-ssl-verify,omitempty" yaml:"etcd-s3-skip-ssl-verify,omitempty"`
	EtcdS3AccessKey                string        `json:"etcd-s3-access-key,omitempty" yaml:"etcd-s3-access-key,omitempty"`
	EtcdS3SecretKey                string        `json:"etcd-s3-secret-key,omitempty" yaml:"etcd-s3-secret-key,omitempty"`
	EtcdS3Bucket                   string        `json:"etcd-s3-bucket,omitempty" yaml:"etcd-s3-bucket,omitempty"`
	EtcdS3Region                   string        `json:"etcd-s3-region,omitempty" yaml:"etcd-s3-region,omitempty"`
	EtcdS3Folder                   string        `json:"etcd-s3-folder,omitempty" yaml:"etcd-s3-folder,omitempty"`
	EtcdS3Proxy                    string        `json:"etcd-s3-proxy,omitempty" yaml:"etcd-s3-proxy,omitempty"`
	EtcdS3ConfigSecret             string        `json:"etcd-s3-config-secret,omitempty" yaml:"etcd-s3-config-secret,omitempty"`
	EtcdS3Insecure                 bool          `json:"etcd-s3-insecure,omitempty" yaml:"etcd-s3-insecure,omitempty"`
	EtcdS3Timeout                  time.Duration `json:"etcd-s3-timeout,omitempty" yaml:"etcd-s3-timeout,omitempty"`
	DefaultLocalStoragePath        string        `yaml:"default-local-storage-path,omitempty" json:"default-local-storage-path,omitempty"`
	Disable                        []string      `json:"disable,omitempty" yaml:"disable,omitempty"`
	DisableScheduler               bool          `json:"disable-scheduler,omitempty" yaml:"disable-scheduler,omitempty"`
	DisableCloudController         bool          `json:"disable-cloud-controller,omitempty" yaml:"disable-cloud-controller,omitempty"`
	DisableKubeProxy               bool          `json:"disable-kube-proxy,omitempty" yaml:"disable-kube-proxy,omitempty"`
	DisableNetworkPolicy           bool          `json:"disable-network-policy,omitempty" yaml:"disable-network-policy,omitempty"`
	DisableHelmController          bool          `json:"disable-helm-controller,omitempty" yaml:"disable-helm-controller,omitempty"`
	DisableApiServer               bool          `json:"disable-apiserver,omitempty" yaml:"disable-apiserver,omitempty"`
	DisableControllerManager       bool          `json:"disable-controller-manager,omitempty" yaml:"disable-controller-manager,omitempty"`
	DisableEtcd                    bool          `json:"disable-etcd,omitempty" yaml:"disable-etcd,omitempty"`
	EmbeddedRegistry               bool          `json:"embedded-registry,omitempty" yaml:"embedded-registry,omitempty"`
	SupervisorMetrics              bool          `json:"supervisor-metrics,omitempty" yaml:"supervisor-metrics,omitempty"`
	NodeName                       string        `yaml:"node-name,omitempty" json:"node-name,omitempty"`
	WithNodeID                     bool          `json:"with-node-id,omitempty" yaml:"with-node-id,omitempty"`
	NodeLabel                      []string      `yaml:"node-label,omitempty" json:"node-label,omitempty"`
	NodeTaint                      []string      `yaml:"node-taint,omitempty" json:"node-taint,omitempty"`
	ImageCredentialProviderBinDir  string        `json:"image-credential-provider-bin-dir,omitempty" yaml:"image-credential-provider-bin-dir,omitempty"`
	ImageCredentialProviderConfig  string        `json:"image-credential-provider-config,omitempty" yaml:"image-credential-provider-config,omitempty"`
	Docker                         bool          `json:"docker,omitempty" yaml:"docker,omitempty"`
	ContainerRuntimeEndpoint       string        `json:"container-runtime-endpoint,omitempty" yaml:"container-runtime-endpoint,omitempty"`
	DefaultRuntime                 string        `json:"default-runtime,omitempty" yaml:"default-runtime,omitempty"`
	ImageServiceEndpoint           string        `json:"image-service-endpoint,omitempty" yaml:"image-service-endpoint,omitempty"`
	DisableDefaultRegistryEndpoint bool          `json:"disable-default-registry-endpoint,omitempty" yaml:"disable-default-registry-endpoint,omitempty"`
	NonrootDevices                 bool          `json:"nonroot-devices,omitempty" yaml:"nonroot-devices,omitempty"`
	PauseImage                     string        `json:"pause-image,omitempty" yaml:"pause-image,omitempty"`
	Snapshotter                    string        `json:"snapshotter,omitempty" yaml:"snapshotter,omitempty"`
	PrivateRegistry                string        `json:"private-registry,omitempty"  yaml:"private-registry,omitempty"`
	SystemDefaultRegistry          string        `yaml:"system-default-registry,omitempty" json:"system-default-registry,omitempty"`
	AirgapExtraRegistry            []string      `json:"airgap-extra-registry,omitempty" yaml:"airgap-extra-registry,omitempty"`
	NodeIP                         []string      `yaml:"node-ip,omitempty" json:"node-ip,omitempty"`
	NodeExternalIP                 []string      `yaml:"node-external-ip,omitempty" json:"node-external-ip,omitempty"`
	NodeInternalDNS                []string      `json:"node-internal-dns,omitempty" yaml:"node-internal-dns,omitempty"`
	NodeExternalDNS                []string      `json:"node-external-dns,omitempty" yaml:"node-external-dns,omitempty"`
	ResolvConf                     string        `yaml:"resolv-conf,omitempty" json:"resolv-conf,omitempty"`
	FlannelIface                   string        `json:"flannel-iface,omitempty" yaml:"flannel-iface,omitempty"`
	FlannelConf                    string        `json:"flannel-conf,omitempty" yaml:"flannel-conf,omitempty"`
	FlannelCniConfFile             string        `json:"flannel-cni-conf,omitempty" yaml:"flannel-cni-conf,omitempty"`
	VPNAuth                        string        `json:"vpn-auth,omitempty" yaml:"vpn-auth,omitempty"`
	VPNAuthFile                    string        `json:"vpn-auth-file,omitempty" yaml:"vpn-auth-file,omitempty"`
	KubeletArg                     []string      `json:"kubelet-arg,omitempty"  yaml:"kubelet-arg,omitempty"`
	KubeProxyArg                   []string      `json:"kube-proxy-arg,omitempty"  yaml:"kube-proxy-arg,omitempty"`
	ProtectKernelDefaults          bool          `json:"protect-kernel-defaults,omitempty" yaml:"protect-kernel-defaults,omitempty"`
	SecretsEncryption              bool          `yaml:"secrets-encryption,omitempty" json:"secrets-encryption,omitempty"`
	EnablePProf                    bool          `yaml:"enable-pprof,omitempty" json:"enable-pprof,omitempty"`
	Rootless                       bool          `yaml:"rootless,omitempty" json:"rootless,omitempty"`
	PreferBundledBin               bool          `json:"prefer-bundled-bin,omitempty" yaml:"prefer-bundled-bin,omitempty"`
	EnableSELinux                  bool          `json:"selinux,omitempty" yaml:"selinux,omitempty"`
	LBServerPort                   int           `json:"lb-server-port,omitempty" yaml:"lb-server-port,omitempty"`
	DisableAgent                   bool          `json:"disable-agent,omitempty" yaml:"disable-agent,omitempty"`
	KubeControllerArg              []string      `json:"kube-controller-arg,omitempty" yaml:"kube-controller-arg,omitempty"`
	KubeCloudControllerArg         []string      `json:"kube-cloud-controller-arg,omitempty" yaml:"kube-cloud-controller-arg,omitempty"`
}
