package main

import (
	"os"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/mudler/go-pluggable"
	"github.com/sirupsen/logrus"

	"github.com/kairos-io/provider-k3s/pkg/provider"
)

func main() {
	f, err := os.OpenFile("/var/log/provider-k3s.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	logrus.SetOutput(f)

	plugin := clusterplugin.ClusterPlugin{
		Provider: provider.ClusterProvider,
	}

	if err := plugin.Run(pluggable.FactoryPlugin{
		EventType:     clusterplugin.EventClusterReset,
		PluginHandler: provider.HandleClusterReset,
	}); err != nil {
		logrus.Fatal(err)
	}
}
