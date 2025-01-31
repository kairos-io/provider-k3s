package provider

import (
	"bytes"
	_ "embed"
	"reflect"
	"testing"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"gopkg.in/yaml.v3"
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
			expectedUserOptions:  []byte(`{}`),
		},
		{
			name: "Init: Standard",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "init",
			},
			expectedOptions:      []byte(`{"tls-san":["localhost"],"token":"token","cluster-init":true}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`{}`),
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
			expectedOptions:      []byte(`{"cluster-init":false,"tls-san":["localhost"],"token":"token","datastore-endpoint":"localhost:2379"}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`{}`),
		},
		{
			name: "Control Plane",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "controlplane",
			},
			expectedOptions:      []byte(`{"tls-san":["localhost"],"token":"token","server":"https://localhost:6443"}`),
			expectedProxyOptions: []byte(`null`),
			expectedUserOptions:  []byte(`{}`),
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
		{
			name: "Control Plane: With Options",
			cluster: clusterplugin.Cluster{
				ClusterToken:     "token",
				ControlPlaneHost: "localhost",
				Role:             "controlplane",
				Options: `disable-apiserver-lb: true
enable-pprof: true`,
			},
			expectedOptions:      []byte(`{"tls-san":["localhost"],"token":"token","server":"https://localhost:6443"}`),
			expectedProxyOptions: []byte(`{"disable-apiserver-lb":true,"enable-pprof":true}`),
			expectedUserOptions:  []byte(`{"enable-pprof":true}`),
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

func Test_unmarshall(t *testing.T) {
	yamlfile := `test: "xyz,zyx"
`
	type Xyz struct {
		Test string `json:"test,omitempty" yaml:"test,omitempty"`
	}
	x := Xyz{}
	err := yaml.Unmarshal([]byte(yamlfile), &x)
	if err != nil {
		t.Errorf("unmarshall() error = %v", err)
	}

	t.Logf("%+v", x)
	t.Logf("%s\n", x.Test)
	u, err := yaml.Marshal(x)
	if err != nil {
		t.Errorf("unmarshall() error = %v", err)
	}

	t.Logf("%s\n", string(u))
}

func Test_decodeOptions(t *testing.T) {
	type args struct {
		in map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "One list of interfaces",
			args: args{
				in: map[string]interface{}{
					"test": []string{"xyz", "zyx"},
				},
			},
			want: map[string]interface{}{
				"test": []string{"xyz", "zyx"},
			},
		},
		{
			name: "One list of strings",
			args: args{
				in: map[string]interface{}{
					"test": []interface{}{"xyz", "zyx"},
				},
			},
			want: map[string]interface{}{
				"test": []interface{}{"xyz", "zyx"},
			},
		},
		{
			name: "One list of interfaces and one list of strings",
			args: args{
				in: map[string]interface{}{
					"test":  []interface{}{"xyz", "zyx"},
					"test2": []string{"abc", "cba"},
				},
			},
			want: map[string]interface{}{
				"test":  []interface{}{"xyz", "zyx"},
				"test2": []string{"abc", "cba"},
			},
		},
		{
			name: "One string and one comma separated string",
			args: args{
				in: map[string]interface{}{
					"test":         "xyz,zyx",
					"test2":        []string{"abc", "cba"},
					"test3":        "xyz",
					"cluster-cidr": "192.168.0.1/24",
				},
			},
			want: map[string]interface{}{
				"test":         []string{"xyz", "zyx"},
				"test2":        []string{"abc", "cba"},
				"test3":        "xyz",
				"cluster-cidr": []string{"192.168.0.1/24"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeOptions(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
