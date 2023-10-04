package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/kairos-io/kairos/provider-k3s/api"

	yip "github.com/mudler/yip/pkg/schema"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	kyaml "sigs.k8s.io/yaml"
)

const (
	configurationPath       = "/etc/rancher/k3s/config.d"
	containerdEnvConfigPath = "/etc/default"
	systemdConfigPath       = "/etc/systemd/system"

	serverSystemName = "k3s"
	agentSystemName  = "k3s-agent"
	K8S_NO_PROXY     = ".svc,.svc.cluster,.svc.cluster.local"
	BootBefore       = "boot.before"
	LocalImagesPath  = "/opt/content/images"

	twoNodeConfigPath = "/etc/two-node"
	marmot            = "marmot"
	marmotUnit        = `
[Unit]
Description=Marmot synchronizes the k8s state in SQLite between nodes in a two node topology

[Service]
TimeoutStartSec=0
Restart=always

ExecStart=/usr/local/bin/marmot -config /etc/two-node/marmot.toml

[Install]
WantedBy=multi-user.target
`
	marmotLeader = `
# Path to target SQLite database
seq_map_path="/etc/kubernetes/marmot-sm.cbor"
db_path="/etc/kubernetes/state.sqlite3"
node_id=1

[nats]
server_config="/etc/two-node/nats.config"

# Console STDOUT configurations
[logging]
# Configure console logging
verbose=true
# "console" | "json"
format="console"
`
	marmotFollowerTmpl = `
# Path to target SQLite database
seq_map_path="/etc/kubernetes/marmot-sm.cbor"
db_path="/etc/kubernetes/state.sqlite3"

[nats]
# address of the nats leader
urls=[
  "{{ .NatsLeaderUri }}"
]

# Console STDOUT configurations
[logging]
# Configure console logging
verbose=true
# "console" | "json"
format="console"
`
	natsConfig = "listen: 0.0.0.0:4222"
)

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

	var providerConfig bytes.Buffer
	_ = yaml.NewEncoder(&providerConfig).Encode(&k3sConfig)

	userOptions, _ := kyaml.YAMLToJSON([]byte(userOptionConfig))
	proxyOptions, _ := kyaml.YAMLToJSON([]byte(cluster.Options))
	options, _ := kyaml.YAMLToJSON(providerConfig.Bytes())

	logrus.Infof("cluster.Env : %+v", cluster.Env)
	proxyValues := proxyEnv(proxyOptions, cluster.Env)
	logrus.Infof("proxyValues : %s", proxyValues)

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

	stages := []yip.Stage{}

	_, twoNode := cluster.Env["two-node"]

	if twoNode {
		marmotConfig := marmotLeader

		if cluster.Role == clusterplugin.RoleWorker {
			var buf bytes.Buffer
			marmotArgs := map[string]interface{}{
				"NatsLeaderUri": cluster.ControlPlaneHost,
			}
			tmpl, _ := template.New(marmot).Parse(marmotFollowerTmpl)
			_ = tmpl.Execute(&buf, &marmotArgs)
			marmotConfig = buf.String()
		}

		stages = append(stages, []yip.Stage{
			{
				Name: "Prepare two-node sqlite db",
				Commands: []string{
					"chmod +x /opt/k3s/scripts/prepare_sqlite.sh",
					"/bin/sh /opt/k3s/scripts/prepare_sqlite.sh > /var/log/prepare_sqlite.log",
				},
			},
			{
				Name: "Prepare marmot config",
				Files: []yip.File{
					{
						Path:        filepath.Join(twoNodeConfigPath, "marmot.toml"),
						Permissions: 0644,
						Content:     marmotConfig,
					},
				},
			},
			{
				Name: "Enable marmot systemd service",
				If:   "[ -x /bin/systemctl ]",
				Files: []yip.File{
					{
						// add nats config irrespective of node role
						Path:        filepath.Join(twoNodeConfigPath, "nats.config"),
						Permissions: 0644,
						Content:     natsConfig,
					},
					{
						Path:        filepath.Join(systemdConfigPath, "marmot.service"),
						Permissions: 0644,
						Content:     marmotUnit,
					},
				},
				Systemctl: yip.Systemctl{
					Enable: []string{marmot},
					Start:  []string{marmot},
				},
			},
		}...)
	}

	stages = append(stages, yip.Stage{
		Name:  "Install K3s Configuration Files",
		Files: files,
		Commands: []string{
			fmt.Sprintf("jq -s 'def flatten: reduce .[] as $i([]; if $i | type == \"array\" then . + ($i | flatten) else . + [$i] end); [.[] | to_entries] | flatten | reduce .[] as $dot ({}; .[$dot.key] += $dot.value)' %s/*.yaml > /etc/rancher/k3s/config.yaml", configurationPath),
		},
	})

	var importStage yip.Stage
	if cluster.ImportLocalImages {
		if cluster.LocalImagesPath == "" {
			cluster.LocalImagesPath = LocalImagesPath
		}

		importStage = yip.Stage{
			Commands: []string{
				"chmod +x /opt/k3s/scripts/import.sh",
				fmt.Sprintf("/bin/sh /opt/k3s/scripts/import.sh %s > /var/log/import.log", cluster.LocalImagesPath),
			},
		}
		stages = append(stages, importStage)
	}

	stages = append(stages,
		yip.Stage{
			Name: "Enable OpenRC Services",
			If:   "[ -x /sbin/openrc-run ]",
			Commands: []string{
				fmt.Sprintf("rc-update add %s default >/dev/null", systemName),
				fmt.Sprintf("service %s start", systemName),
			},
		},
		yip.Stage{
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
		})

	cfg := yip.YipConfig{
		Name: "K3s Kairos Cluster Provider",
		Stages: map[string][]yip.Stage{
			BootBefore: stages,
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
