package api

import (
	"github.com/urfave/cli"
)

type K3sAgentConfig struct {

	// Server to connect to
	Server string `json:"server,omitempty" yaml:"server,omitempty"`

	// Token to use for authentication
	Token string `json:"token,omitempty" yaml:"token,omitempty"`

	// NodeLabels  Registering and starting kubelet with set of labels
	// +optional
	NodeLabels cli.StringSlice `json:"node-label,omitempty" yaml:"node-label,omitempty"`

	// NodeTaints Registering kubelet with set of taints
	// +optional
	NodeTaints []string `json:"node-taint,omitempty"  yaml:"node-taint,omitempty"`

	// PrivateRegistry  registry configuration file
	// +optional
	PrivateRegistry string `json:"private-registry,omitempty"  yaml:"private-registry,omitempty"`

	// KubeletArgs Customized flag for kubelet process
	// +optional
	KubeletArgs []string `json:"kubelet-arg,omitempty"  yaml:"kubelet-arg,omitempty"`

	// KubeProxyArgs Customized flag for kube-proxy process
	// +optional
	KubeProxyArgs []string `json:"kube-proxy-arg,omitempty"  yaml:"kube-proxy-arg,omitempty"`

	// NodeName Name of the Node
	// +optional
	NodeName string `json:"node-name,omitempty"  yaml:"node-name,omitempty"`

	// NoFlannel to disable flannel
	// +optional
	NoFlannel bool `json:"no-flannel,omitempty"  yaml:"no-flannel,omitempty"`

	// Debug
	// +optional
	Debug bool `json:"debug,omitempty"  yaml:"debug,omitempty"`
}
