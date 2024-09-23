package provider

import (
	"bytes"
	"encoding/json"
	"fmt"

	"net"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	kyaml "sigs.k8s.io/yaml"

	_ "embed"
	"text/template"

	"github.com/kairos-io/provider-k3s/api"
	"github.com/kairos-io/provider-k3s/pkg/constants"
	"github.com/kairos-io/provider-k3s/pkg/types"
)

//go:embed mount.tmpl
var mountTemplate string

const (
	configurationPath       = "/etc/rancher/k3s/config.d"
	containerdEnvConfigPath = "/etc/default"
	localImagesPath         = "/opt/content/images"

	serverSystemName = "k3s"
	agentSystemName  = "k3s-agent"

	bootBefore = "boot.before"
	k8sNoProxy = ".svc,.svc.cluster,.svc.cluster.local"
)

func ClusterProvider(cluster clusterplugin.Cluster) yip.YipConfig {
	logrus.Infof("current node role %s", cluster.Role)
	logrus.Infof("received cluster env %+v", cluster.Env)
	logrus.Infof("received cluster options %s", cluster.Options)

	systemName := serverSystemName
	if cluster.Role == clusterplugin.RoleWorker {
		systemName = agentSystemName
	}

	cfg := yip.YipConfig{
		Name: "K3s Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			bootBefore: parseStages(cluster, parseFiles(cluster, systemName), systemName),
		},
	}

	return cfg
}

func parseOptions(cluster clusterplugin.Cluster) ([]byte, []byte, []byte) {
	k3sConfig := &api.K3sServerConfig{
		Token: cluster.ClusterToken,
	}
	userOptionConfig := cluster.Options

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

	userOptions, _ := kyaml.YAMLToJSON([]byte(userOptionConfig))
	proxyOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
	options, _ := json.Marshal(k3sConfig)

	// if provided, parse additional K3s server options (which may override the above settings)
	if len(cluster.ProviderOptions) > 0 {
		logrus.Infof("applying cluster provider options: %+v", cluster.ProviderOptions)

		providerOpts, err := yaml.Marshal(cluster.ProviderOptions)
		if err != nil {
			logrus.Fatalf("failed to marshal cluster.ProviderOptions: %v", err)
		}
		if err := yaml.Unmarshal(providerOpts, k3sConfig); err != nil {
			logrus.Fatalf("failed to unmarshal cluster.ProviderOptions: %v", err)
		}
		options, _ = json.Marshal(k3sConfig)

		if v, ok := cluster.ProviderOptions[constants.ClusterInit]; ok && v == "no" {
			// Manually set cluster-init to false, as it's dropped by the above marshal.
			// We want to omit this field in all other scenarios, hence this special, ugly case.
			override := []byte(`{"cluster-init":false,`)
			options = append(override, options[1:]...)
		}
	}

	return options, proxyOptions, userOptions
}

func parseFiles(cluster clusterplugin.Cluster, systemName string) []yip.File {
	options, proxyOptions, userOptions := parseOptions(cluster)

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

	return files
}

func rootPathMountStage(rootPath string) yip.Stage {
	mps := []types.MountPoint{
		{
			Name:   "etc-rancher",
			Source: filepath.Join(rootPath, "etc/rancher"),
			Target: "/etc/rancher",
		},
		{
			Name:   "var-lib-rancher",
			Source: filepath.Join(rootPath, "var/lib/rancher"),
			Target: "/var/lib/rancher",
		},
	}

	stage := yip.Stage{
		Name: "Mount K3s data, conf directories",
	}
	for _, mp := range mps {
		stage.Files = append(stage.Files, yip.File{
			Path:        filepath.Join(constants.RunSystemdSystemDir, fmt.Sprintf("%s.mount", mp.Name)),
			Permissions: 0644,
			Content:     parseMountUnitFile(mp),
		})

		stage.Commands = append(stage.Commands,
			fmt.Sprintf("mkdir -p %s", mp.Source),
			fmt.Sprintf("mkdir -p %s", mp.Target),
			fmt.Sprintf("systemctl enable --now %s.mount", mp.Name),
		)
	}

	return stage
}

func parseMountUnitFile(mp types.MountPoint) string {
	mount, _ := template.New("mount").Parse(mountTemplate)
	var buf bytes.Buffer
	_ = mount.Execute(&buf, mp)
	return buf.String()
}

func parseStages(cluster clusterplugin.Cluster, files []yip.File, systemName string) []yip.Stage {
	var stages []yip.Stage
	clusterRootPath := getClusterRootPath(cluster)

	if len(clusterRootPath) > 0 && clusterRootPath != "/" {
		stages = append(stages, rootPathMountStage(clusterRootPath))
	}

	stages = append(stages, yip.Stage{
		Name:  constants.InstallK3sConfigFiles,
		Files: files,
		Commands: []string{
			fmt.Sprintf("jq -s 'def flatten: reduce .[] as $i([]; if $i | type == \"array\" then . + ($i | flatten) else . + [$i] end); [.[] | to_entries] | flatten | reduce .[] as $dot ({}; .[$dot.key] += $dot.value)' %s/*.yaml > /etc/rancher/k3s/config.yaml", configurationPath),
		},
	})

	if cluster.ImportLocalImages {
		if cluster.LocalImagesPath == "" {
			cluster.LocalImagesPath = localImagesPath
		}
		importStage := yip.Stage{
			Name: constants.ImportK3sImages,
			Commands: []string{
				fmt.Sprintf("chmod +x %s/opt/k3s/scripts/import.sh", clusterRootPath),
				fmt.Sprintf("/bin/sh %s/opt/k3s/scripts/import.sh %s > /var/log/k3s-import-images.log", clusterRootPath, filepath.Join(clusterRootPath, cluster.LocalImagesPath)),
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

	return stages
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

func getClusterRootPath(cluster clusterplugin.Cluster) string {
	return cluster.ProviderOptions[constants.ClusterRootPath]
}
