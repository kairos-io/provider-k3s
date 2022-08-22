package main

import (
	"bytes"
	"fmt"
	"github.com/c3os-io/c3os/sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"path/filepath"
	kyaml "sigs.k8s.io/yaml"
)

const (
	configurationPath = "/etc/rancher/k3s/config.d"

	serverSystemName = "k3s"
	agentSystemName  = "k3s-agent"
)

type K3sConfig struct {
	ClusterInit bool     `yaml:"cluster-init,omitempty"`
	Token       string   `yaml:"token,omitempty"`
	Server      string   `yaml:"server,omitempty"`
	TLSSan      []string `yaml:"tls-san,omitempty"`
}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	k3sConfig := K3sConfig{
		Token: cluster.ClusterToken,
	}

	switch cluster.Role {
	case clusterplugin.RoleInit:
		k3sConfig.ClusterInit = true
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
	case clusterplugin.RoleControlPlane:
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
	case clusterplugin.RoleWorker:
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
	}

	systemName := serverSystemName
	if cluster.Role == clusterplugin.RoleWorker {
		systemName = agentSystemName
	}

	// ensure we always have  a valid user config
	if cluster.Options == "" {
		cluster.Options = "{}"
	}

	var providerConfig bytes.Buffer
	_ = yaml.NewEncoder(&providerConfig).Encode(&k3sConfig)

	userOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
	options, _ := kyaml.YAMLToJSON(providerConfig.Bytes())

	cfg := yip.YipConfig{
		Name: "K3s C3OS Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": {
				{
					Name: "Install K3s Configuration Files",
					Files: []yip.File{
						{
							Path:        filepath.Join(configurationPath, "90_userdata.yaml"),
							Permissions: 0400,
							Content:     string(userOptions),
						},
						{
							Path:        filepath.Join(configurationPath, "99_userdata.yaml"),
							Permissions: 0400,
							Content:     string(options),
						},
					},
					Commands: []string{
						fmt.Sprintf("jq -rs 'reduce .[] as $item ({}; . * $item)' %s/*.yaml > /etc/rancher/k3s/config.yaml", configurationPath),
					},
				},
				{
					Name: "Enable OpenRC Services",
					If:   "[ -x /sbin/openrc-run ]",
					Commands: []string{
						fmt.Sprintf("rc-update add %s default >/dev/null", systemName),
						fmt.Sprintf("service %s start", systemName),
					},
				},
				{
					Name: "Enable Systemd Services",
					If:   "[ -x /bin/systemctl ]",
					Systemctl: yip.Systemctl{
						Enable: []string{
							systemName,
						},
						Start: []string{
							systemName,
						},
					},
				},
			},
		},
	}

	return cfg
}

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}
