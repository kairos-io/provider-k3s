package provider

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/kairos-io/kairos-sdk/bus"
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/mudler/go-pluggable"
	"gopkg.in/yaml.v3"
)

func HandleClusterReset(event *pluggable.Event) pluggable.EventResponse {
	var payload bus.EventPayload
	var config clusterplugin.Config
	var response pluggable.EventResponse

	// parse the boot payload
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		response.Error = fmt.Sprintf("failed to parse boot event: %s", err.Error())
		return response
	}

	// parse config from boot payload
	if err := yaml.Unmarshal([]byte(payload.Config), &config); err != nil {
		response.Error = fmt.Sprintf("failed to parse config from boot event: %s", err.Error())
		return response
	}

	if config.Cluster == nil {
		return response
	}

	clusterRootPath := getClusterRootPath(*config.Cluster)

	var uninstallScript string
	if config.Cluster.Role == clusterplugin.RoleWorker {
		uninstallScript = "k3s-agent-uninstall.sh"
	} else {
		uninstallScript = "k3s-uninstall.sh"
	}

	cmd := exec.Command("/bin/sh", "-c", filepath.Join(clusterRootPath, "/opt/k3s/scripts", uninstallScript))
	output, err := cmd.CombinedOutput()
	if err != nil {
		response.Error = fmt.Sprintf("failed to reset cluster: %s", string(output))
	}

	return response
}
