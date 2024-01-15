package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"

	"github.com/kairos-io/provider-k3s/api"
	"github.com/kairos-io/provider-k3s/pkg/constants"
)

const (
	configurationPath       = "/etc/rancher/k3s/config.d"
	containerdEnvConfigPath = "/etc/default"
	localImagesPath         = "/opt/content/images"

	serverSystemName = "k3s"
	agentSystemName  = "k3s-agent"

	k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
)

func clusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	k3sConfig := &api.K3sServerConfig{
		Token: cluster.ClusterToken,
	}
	userOptionConfig := cluster.Options

	logrus.Infof("current node role %s", cluster.Role)
	logrus.Infof("received cluster options %s", cluster.Options)

	// Handle optional node role override
	if cluster.ProviderOptions != nil {
		if v, ok := cluster.ProviderOptions["node-role"]; ok {
			cluster.Role = clusterplugin.Role(v)
		}
	}

	switch cluster.Role {
	case clusterplugin.RoleInit:
		k3sConfig.ClusterInit = true
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
	case clusterplugin.RoleControlPlane:
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
		k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
	case clusterplugin.RoleWorker:
		userOptionConfig = ""
		k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
		// Data received from upstream contains config for both control plane and worker. Thus, for worker,
		// config is being filtered via unmarshal into agent config.
		var agentCfg api.K3sAgentConfig
		if err := yaml.Unmarshal([]byte(cluster.Options), &agentCfg); err == nil {
			out, _ := yaml.Marshal(agentCfg)
			userOptionConfig = string(out)
		} else {
			logrus.Fatalf("failed to un-marshal cluster options in k3s agent config %s", err)
		}
	}

	// if provided, parse additional K3s server options (which may override the above settings)
	if cluster.ProviderOptions != nil {
		providerOpts, err := yaml.Marshal(cluster.ProviderOptions)
		if err != nil {
			logrus.Fatalf("failed to marshal cluster.ProviderOptions: %v", err)
		}
		if err := yaml.Unmarshal(providerOpts, k3sConfig); err != nil {
			logrus.Fatalf("failed to unmarshal cluster.ProviderOptions: %v", err)
		}
		logrus.Infof("applied cluster provider options: %+v", cluster.ProviderOptions)
	}

	systemName := serverSystemName
	if cluster.Role == clusterplugin.RoleWorker {
		systemName = agentSystemName
	}

	userOptions, _ := kyaml.YAMLToJSON([]byte(userOptionConfig))
	proxyOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
	options, _ := json.Marshal(k3sConfig)

	logrus.Infof("received cluster env %+v", cluster.Env)

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

	proxyValues := proxyEnv(proxyOptions, cluster.Env)

	if len(proxyValues) > 0 {
		logrus.Infof("setting proxy values %s", proxyValues)
		files = append(files, yip.File{
			Path:        filepath.Join(containerdEnvConfigPath, systemName),
			Permissions: 0400,
			Content:     proxyValues,
		})
	}

	var stages []yip.Stage

	stages = append(stages, yip.Stage{
		Name:  constants.InstallK3sConfigFiles,
		Files: files,
		Commands: []string{
			fmt.Sprintf("jq -s 'def flatten: reduce .[] as $i([]; if $i | type == \"array\" then . + ($i | flatten) else . + [$i] end); [.[] | to_entries] | flatten | reduce .[] as $dot ({}; .[$dot.key] += $dot.value)' %s/*.yaml > /etc/rancher/k3s/config.yaml", configurationPath),
		},
	})

	var importStage yip.Stage
	if cluster.ImportLocalImages {
		if cluster.LocalImagesPath == "" {
			cluster.LocalImagesPath = localImagesPath
		}

		importStage = yip.Stage{
			Name: constants.ImportK3sImages,
			Commands: []string{
				"chmod +x /opt/k3s/scripts/import.sh",
				fmt.Sprintf("/bin/sh /opt/k3s/scripts/import.sh %s > /var/log/k3s-import-images.log", cluster.LocalImagesPath),
			},
		}
		stages = append(stages, importStage)
	}

	stages = append(stages,
		yip.Stage{
			Name: constants.EnableOpenRCServices,
			If:   "[ -x /sbin/openrc-run ]",
			Commands: []string{
				fmt.Sprintf("rc-update add %s default >/dev/null", systemName),
				fmt.Sprintf("service %s start", systemName),
			},
		},
		yip.Stage{
			Name: constants.EnableSystemdServices,
			If:   "[ -x /bin/systemctl ]",
			Commands: []string{
				fmt.Sprintf("systemctl enable %s", systemName),
				fmt.Sprintf("systemctl restart %s", systemName),
			},
		},
	)

	cfg := yip.YipConfig{
		Name: "K3s Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			"boot.before": stages,
		},
	}

	return cfg
}

func proxyEnv(proxyOptions []byte, proxyMap map[string]string) string {
	var proxy []string
	var noProxy string
	var isProxyConfigured bool

	httpProxy := proxyMap["HTTP_PROXY"]
	httpsProxy := proxyMap["HTTPS_PROXY"]
	userNoProxy := proxyMap["NO_PROXY"]

	defaultNoProxy := getDefaultNoProxy(proxyOptions)
	logrus.Infof("setting default no proxy to %s", defaultNoProxy)

	if len(httpProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTP_PROXY=%s", httpProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_HTTP_PROXY=%s", httpProxy))
		isProxyConfigured = true
	}

	if len(httpsProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("HTTPS_PROXY=%s", httpsProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_HTTPS_PROXY=%s", httpsProxy))
		isProxyConfigured = true
	}

	if isProxyConfigured {
		noProxy = defaultNoProxy
	}

	if len(userNoProxy) > 0 {
		noProxy = noProxy + "," + userNoProxy
	}

	if len(noProxy) > 0 {
		proxy = append(proxy, fmt.Sprintf("NO_PROXY=%s", noProxy))
		proxy = append(proxy, fmt.Sprintf("CONTAINERD_NO_PROXY=%s", noProxy))
	}

	return strings.Join(proxy, "\n")
}

func getDefaultNoProxy(proxyOptions []byte) string {
	var noProxy string

	data := make(map[string]interface{})
	err := json.Unmarshal(proxyOptions, &data)
	if err != nil {
		logrus.Fatalf("error while unmarshalling user options %s", err)
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
	noProxy = noProxy + "," + getNodeCIDR() + "," + k8sNoProxy

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
	f, err := os.OpenFile("/var/log/provider-k3s.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	logrus.SetOutput(f)

	plugin := clusterplugin.ClusterPlugin{
		Provider: clusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}
