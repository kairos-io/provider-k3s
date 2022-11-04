package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"

	"os"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos/pkg/config"
	"github.com/kairos-io/kairos/provider-k3s/api"
	"github.com/kairos-io/kairos/sdk/clusterplugin"

	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	kyaml "sigs.k8s.io/yaml"
)

const (
	configurationPath       = "/etc/rancher/k3s/config.d"
	containerdEnvConfigPath = "/etc/default"

	serverSystemName = "k3s"
	agentSystemName  = "k3s-agent"
	K8S_NO_PROXY     = ".svc,.svc.cluster,.svc.cluster.local"
)

var configScanDir = []string{"/oem", "/usr/local/cloud-config", "/run/initramfs/live"}

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	k3sConfig := api.K3sServerConfig{
		Token: cluster.ClusterToken,
	}

	var userOptionConfig string
	switch cluster.Role {
	case clusterplugin.RoleInit:
		k3sConfig.ClusterInit = true
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
		userOptionConfig = cluster.Options
	case clusterplugin.RoleControlPlane:
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
		userOptionConfig = cluster.Options
	case clusterplugin.RoleWorker:
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
		//Data received from upstream contains config for both control plane and worker. Thus, for worker, config is being filtered
		//via unmarshal into agent config.
		var agentCfg api.K3sAgentConfig
		if err := yaml.Unmarshal([]byte(cluster.Options), &agentCfg); err == nil {
			out, _ := yaml.Marshal(agentCfg)
			userOptionConfig = string(out)
		}
	}

	systemName := serverSystemName
	if cluster.Role == clusterplugin.RoleWorker {
		systemName = agentSystemName
	}

	_config, _ := config.Scan(config.Directories(configScanDir...))

	if _config != nil {
		for _, e := range _config.Env {
			pair := strings.SplitN(e, "=", 2)
			if len(pair) >= 2 {
				os.Setenv(pair[0], pair[1])
			}
		}
	}

	var providerConfig bytes.Buffer
	_ = yaml.NewEncoder(&providerConfig).Encode(&k3sConfig)

	userOptions, _ := kyaml.YAMLToJSON([]byte(userOptionConfig))
	options, _ := kyaml.YAMLToJSON(providerConfig.Bytes())

	proxyValues := proxyEnv(userOptions)

	files := []yip.File{
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
	}

	if len(proxyValues) > 0 {
		files = append(files, yip.File{
			Path:        filepath.Join(containerdEnvConfigPath, systemName),
			Permissions: 0400,
			Content:     proxyValues,
		})
	}

	cfg := yip.YipConfig{
		Name: "K3s Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": {
				{
					Name:  "Install K3s Configuration Files",
					Files: files,
					Commands: []string{
						fmt.Sprintf("jq -s 'def flatten: reduce .[] as $i([]; if $i | type == \"array\" then . + ($i | flatten) else . + [$i] end); [.[] | to_entries] | flatten | reduce .[] as $dot ({}; .[$dot.key] += $dot.value)' %s/*.yaml > /etc/rancher/k3s/config.yaml", configurationPath),
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

func proxyEnv(userOptions []byte) string {
	var proxy []string

	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	noProxy := getNoProxy(userOptions)

	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_HTTP_PROXY=%s", httpProxy))
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_HTTPS_PROXY=%s", httpsProxy))
	}

	if len(noProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_NO_PROXY=%s", noProxy))
	}

	return strings.Join(proxy, "\n")
}

func getNoProxy(userOptions []byte) string {

	noProxy := os.Getenv("NO_PROXY")

	if len(noProxy) > 0 {
		var data map[string]interface{}
		err := json.Unmarshal(userOptions, &data)
		if err != nil {
			fmt.Println("error while unmarshalling user options", err)
		}

		if data != nil {
			clusterCIDR := data["cluster-cidr"].(string)
			serviceCIDR := data["service-cidr"].(string)

			if len(clusterCIDR) > 0 {
				noProxy = noProxy + "," + clusterCIDR
			}
			if len(serviceCIDR) > 0 {
				noProxy = noProxy + "," + serviceCIDR
			}
		}
		noProxy = noProxy + "," + getNodeCIDR() + "," + K8S_NO_PROXY
	}
	return noProxy
}

func getNodeCIDR() string {
	addrs, _ := net.InterfaceAddrs()
	var result string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				result = addr.String()
				break
			}
		}
	}
	return result
}

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}
