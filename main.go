package main

import (
	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/mudler/go-pluggable"
	"github.com/sirupsen/logrus"

	"github.com/kairos-io/provider-k3s/pkg/log"
	"github.com/kairos-io/provider-k3s/pkg/provider"
)

func main() {
	log.InitLogger("/var/log/provider-k3s.log")

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
