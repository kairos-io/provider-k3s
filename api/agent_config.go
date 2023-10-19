package api

type K3sAgentConfig struct {
	Server                   string   `json:"server,omitempty" yaml:"server,omitempty"`
	Token                    string   `json:"token,omitempty" yaml:"token,omitempty"`
	NodeLabels               []string `json:"node-label,omitempty" yaml:"node-label,omitempty"`
	NodeTaints               []string `json:"node-taint,omitempty"  yaml:"node-taint,omitempty"`
	PrivateRegistry          string   `json:"private-registry,omitempty"  yaml:"private-registry,omitempty"`
	KubeletArgs              []string `json:"kubelet-arg,omitempty"  yaml:"kubelet-arg,omitempty"`
	KubeProxyArgs            []string `json:"kube-proxy-arg,omitempty"  yaml:"kube-proxy-arg,omitempty"`
	NodeName                 string   `json:"node-name,omitempty"  yaml:"node-name,omitempty"`
	Debug                    bool     `json:"debug,omitempty"  yaml:"debug,omitempty"`
	WithNodeId               string   `json:"with-node-id,omitempty" yaml:"with-node-id,omitempty"`
	NodeIP                   string   `json:"node-ip,omitempty" yaml:"node-ip,omitempty"`
	NodeExternalIP           string   `json:"node-external-ip,omitempty" yaml:"node-external-ip,omitempty"`
	FlannelIface             string   `json:"flannel-iface,omitempty" yaml:"flannel-iface,omitempty"`
	FlannelConf              string   `json:"flannel-conf,omitempty" yaml:"flannel-conf,omitempty"`
	FlannelCniConfFile       string   `json:"flannel-cni-conf-file,omitempty" yaml:"flannel-cni-conf-file,omitempty"`
	EnableSELinux            bool     `json:"selinux,omitempty" yaml:"selinux,omitempty"`
	PauseImage               string   `json:"pause-image,omitempty" yaml:"pause-image,omitempty"`
	Snapshotter              string   `json:"snapshotter,omitempty" yaml:"snapshotter,omitempty"`
	Docker                   bool     `json:"docker,omitempty" yaml:"docker,omitempty"`
	ContainerRuntimeEndpoint string   `json:"container-runtime-endpoint,omitempty" yaml:"container-runtime-endpoint,omitempty"`
	ProtectKernelDefaults    bool     `json:"protect-kernel-defaults,omitempty" yaml:"protect-kernel-defaults,omitempty"`
	DataDir                  string   `json:"data-dir,omitempty" yaml:"data-dir,omitempty"`
	TokenFile                string   `json:"token-file,omitempty" yaml:"token-file,omitempty"`
	LBServerPort             int      `json:"lb-server-port,omitempty" yaml:"lb-server-port,omitempty"`
	ResolvConf               string   `json:"resolv-conf,omitempty" yaml:"resolv-conf,omitempty"`
}
