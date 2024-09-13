package provider

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	yip "github.com/mudler/yip/pkg/schema"
	"github.com/onsi/gomega"
)

func Test_parseOptions(t *testing.T) {
	tests := []struct {
		name                 string
		cluster              clusterplugin.Cluster
		expectedOptions      []byte
		expectedProxyOptions []byte
		expectedUserOptions  []byte
	}{
		{
			name:                 "Empty input",
			cluster:              clusterplugin.Cluster{},
			expectedOptions:      []byte(`{}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`null`),
		},
		{
			name: "Init: Standard",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "init",
			},
			expectedOptions:      []byte(`{"cluster-init":true,"token":"token","tls-san":["localhost"]}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`null`),
		},
		{
			name: "Init: 2-Node",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "init",
				ProviderOptions: map[string]string{
					"cluster-init":       "no",
					"datastore-endpoint": "localhost:2379",
				},
			},
			expectedOptions:      []byte(`{"cluster-init":false,"token":"token","tls-san":["localhost"],"datastore-endpoint":"localhost:2379"}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`null`),
		},
		{
			name: "Control Plane",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "controlplane",
			},
			expectedOptions:      []byte(`{"token":"token","server":"https://localhost:6443","tls-san":["localhost"]}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`null`),
		},
		{
			name: "Worker",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "worker",
			},
			expectedOptions:      []byte(`{"token":"token","server":"https://localhost:6443"}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`{}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, proxyOptions, userOptions := parseOptions(tt.cluster)
			if !bytes.Equal(options, tt.expectedOptions) {
				t.Errorf("parseOptions() options = %v, want %v", string(options), string(tt.expectedOptions))
			}
			if !bytes.Equal(proxyOptions, tt.expectedProxyOptions) {
				t.Errorf("parseOptions() proxyOptions = %v, want %v", string(proxyOptions), string(tt.expectedProxyOptions))
			}
			if !bytes.Equal(userOptions, tt.expectedUserOptions) {
				t.Errorf("parseOptions() userOptions = %v, want %v", string(userOptions), string(tt.expectedUserOptions))
			}
		})
	}
}

func Test_rootPathMountStage(t *testing.T) {
	gomega.RegisterTestingT(t)

	type args struct {
		rootPath string
	}
	tests := []struct {
		name string
		args args
		want yip.Stage
	}{
		{
			name: "custom root path",
			args: args{
				rootPath: "/writable",
			},
			want: yip.Stage{
				Name: "Mount K3s data, conf directories",
				Commands: []string{
					"mkdir -p /writable/etc/rancher",
					"mkdir -p /etc/rancher",
					"systemctl enable --now etc-rancher.mount",
					"mkdir -p /writable/var/lib/rancher",
					"mkdir -p /var/lib/rancher",
					"systemctl enable --now var-lib-rancher.mount",
				},
				Files: []yip.File{
					{
						Path:        "/run/systemd/system/etc-rancher.mount",
						Permissions: 0644,
						Content:     "[Unit]\nDescription=etc-rancher mount unit\nBefore=local-fs.target k3s.service k3s-agent.service\n\n[Mount]\nWhat=/writable/etc/rancher\nWhere=/etc/rancher\nType=none\nOptions=bind\n\n[Install]\nWantedBy=local-fs.target\n",
					},
					{
						Path:        "/run/systemd/system/var-lib-rancher.mount",
						Permissions: 0644,
						Content:     "[Unit]\nDescription=var-lib-rancher mount unit\nBefore=local-fs.target k3s.service k3s-agent.service\n\n[Mount]\nWhat=/writable/var/lib/rancher\nWhere=/var/lib/rancher\nType=none\nOptions=bind\n\n[Install]\nWantedBy=local-fs.target\n",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rootPathMountStage(tt.args.rootPath)
			gomega.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
