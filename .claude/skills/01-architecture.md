# K3s Provider Architecture

## Overview

The provider-k3s is a Kairos/C3OS cluster provider that configures K3s (Lightweight Kubernetes) installations for edge deployments. K3s is optimized for resource-constrained environments with a small binary footprint and simplified architecture, making it ideal for edge computing, IoT devices, and CI/CD pipelines.

## Key Concepts

### Kubernetes Distribution: K3s

K3s is a certified Kubernetes distribution built for IoT and edge computing:

- **Lightweight**: Single binary under 100MB
- **Simplified**: Batteries-included with minimal external dependencies
- **Production-ready**: Fully conformant Kubernetes with CNCF certification
- **Embedded Components**: Includes containerd, CoreDNS, Flannel, and Traefik by default
- **Edge-optimized**: Low resource consumption, ARM support, air-gap capable

### Kairos/C3OS Integration

Provider-k3s integrates with Kairos immutable Linux distribution:

- **Cloud-init Configuration**: Declarative cluster setup via cluster section
- **Immutable OS**: A/B partition updates with atomic upgrades
- **Boot Stages**: Yip-based stage execution during boot.before phase
- **Service Management**: OpenRC and systemd service orchestration
- **Image Import**: Local container image preloading for air-gap deployments

### Component Architecture

```
┌─────────────────────────────────────────────────────┐
│              Cloud-Init (User Configuration)         │
│  cluster:                                           │
│    cluster_token: token123                         │
│    control_plane_host: 10.0.1.100                  │
│    role: init|controlplane|worker                  │
│    config: |                                       │
│      node-name: edge-node-1                        │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│          Provider-K3s (Cluster Plugin)              │
│  • Parse cluster configuration                      │
│  • Generate K3s config files                        │
│  • Configure proxy settings                         │
│  • Manage TLS SANs                                  │
│  • Handle role-specific setup                       │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│               Yip Stage Execution                   │
│  boot.before:                                       │
│    1. Disable swap                                  │
│    2. Install config files                          │
│    3. Import local images (optional)                │
│    4. Enable k3s/k3s-agent service                  │
│    5. Start K3s                                     │
└─────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────┐
│                K3s Installation                     │
│  ┌──────────────┐  ┌────────────────┐              │
│  │ K3s Server   │  │  K3s Agent     │              │
│  │ (Control     │  │  (Worker)      │              │
│  │  Plane)      │  │                │              │
│  │              │  │                │              │
│  │ • API Server │  │ • Kubelet      │              │
│  │ • Scheduler  │  │ • Kube-proxy   │              │
│  │ • Controller │  │ • Container    │              │
│  │ • etcd       │  │   Runtime      │              │
│  └──────────────┘  └────────────────┘              │
└─────────────────────────────────────────────────────┘
```

### Configuration Flow

1. **Cluster Definition**: User defines cluster configuration in cloud-init
2. **Provider Execution**: Provider-k3s processes configuration via ClusterProvider()
3. **Config Generation**: Creates K3s YAML configs in /etc/rancher/k3s/config.d/
4. **Config Merging**: jq merges multiple config files into /etc/rancher/k3s/config.yaml
5. **Service Start**: systemd/OpenRC starts k3s or k3s-agent service
6. **Cluster Join**: Node joins cluster using cluster_token and control_plane_host

### File Structure

```
/etc/rancher/k3s/
├── config.d/
│   ├── 90_userdata.yaml          # User-provided configuration
│   └── 99_userdata.yaml          # Provider-generated configuration
├── config.yaml                    # Merged final configuration
└── ...

/etc/default/
├── k3s                            # Server proxy environment variables
└── k3s-agent                      # Agent proxy environment variables

/var/log/
├── k3s-import-images.log         # Image import logs (if enabled)
└── ...
```

## Implementation Patterns

### Role-Based Configuration

The provider handles three distinct roles with different configurations:

**Init Role (Bootstrap First Node)**:
```go
// pkg/provider/provider.go
case clusterplugin.RoleInit:
    k3sConfig.ClusterInit = true
    k3sConfig.Server = ""  // No upstream server
    k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
    // Initializes embedded etcd cluster
```

**ControlPlane Role (Additional Control Plane)**:
```go
case clusterplugin.RoleControlPlane:
    k3sConfig.ClusterInit = false
    k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
    k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
    // Joins existing etcd cluster
```

**Worker Role (Agent Only)**:
```go
case clusterplugin.RoleWorker:
    systemName = "k3s-agent"
    k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
    // Runs kubelet and container runtime only
```

### Configuration File Generation

```go
// pkg/provider/provider.go:parseFiles()
files := []yip.File{
    {
        Path:        "/etc/rancher/k3s/config.d/90_userdata.yaml",
        Permissions: 0400,
        Content:     string(userOptions),  // User YAML config
    },
    {
        Path:        "/etc/rancher/k3s/config.d/99_userdata.yaml",
        Permissions: 0400,
        Content:     string(options),       // Provider-generated config
    },
}
```

Configuration merging happens via jq during boot:
```bash
jq -s 'def flatten: reduce .[] as $i([]; if $i | type == "array" then . + ($i | flatten) else . + [$i] end); [.[] | to_entries] | flatten | reduce .[] as $dot ({}; .[$dot.key] += $dot.value)' /etc/rancher/k3s/config.d/*.yaml > /etc/rancher/k3s/config.yaml
```

### Proxy Configuration

Provider-k3s handles HTTP/HTTPS proxy with automatic NO_PROXY calculation:

```go
// pkg/provider/provider.go:proxyEnv()
func proxyEnv(proxyOptions []byte, proxyMap map[string]string) string {
    // Extracts cluster-cidr and service-cidr from user config
    // Adds node CIDR automatically
    // Appends .svc,.svc.cluster,.svc.cluster.local
    // Generates /etc/default/k3s with CONTAINERD_HTTP_PROXY, etc.
}
```

Example generated proxy file:
```bash
HTTP_PROXY=http://proxy.example.com:8080
HTTPS_PROXY=http://proxy.example.com:8080
CONTAINERD_HTTP_PROXY=http://proxy.example.com:8080
CONTAINERD_HTTPS_PROXY=http://proxy.example.com:8080
NO_PROXY=10.42.0.0/16,10.43.0.0/16,192.168.1.0/24,.svc,.svc.cluster,.svc.cluster.local
CONTAINERD_NO_PROXY=10.42.0.0/16,10.43.0.0/16,192.168.1.0/24,.svc,.svc.cluster,.svc.cluster.local
```

### Image Import for Air-Gap

```go
// pkg/provider/provider.go:parseStages()
if cluster.ImportLocalImages {
    if cluster.LocalImagesPath == "" {
        cluster.LocalImagesPath = "/opt/content/images"
    }
    importStage := yip.Stage{
        Name: constants.ImportK3sImages,
        Commands: []string{
            "chmod +x /opt/k3s/scripts/import.sh",
            "/bin/sh /opt/k3s/scripts/import.sh /opt/content/images > /var/log/k3s-import-images.log",
        },
    }
}
```

## Common Pitfalls

### 1. Incorrect Cluster Token Format

K3s tokens must be at least 16 characters. Short tokens cause authentication failures:
```yaml
# BAD
cluster_token: abc123

# GOOD
cluster_token: K10abcdef1234567890abcdef1234567890
```

### 2. Control Plane Host Mismatch

Workers must use the same control_plane_host as the init node's IP:
```yaml
# Init node config (IP: 10.0.1.100)
cluster:
  role: init
  control_plane_host: 10.0.1.100

# Worker config - MUST match
cluster:
  role: worker
  control_plane_host: 10.0.1.100  # Same IP
```

### 3. TLS SAN Issues

If accessing cluster via hostname/load balancer, add to tls-san:
```yaml
cluster:
  config: |
    tls-san:
      - cluster.example.com
      - 10.0.1.100
      - load-balancer.local
```

### 4. Proxy Configuration Conflicts

When using proxy, ensure cluster and service CIDRs are in NO_PROXY:
```yaml
cluster:
  env:
    HTTP_PROXY: "http://proxy.example.com:8080"
    NO_PROXY: "10.0.0.0/8,192.168.0.0/16,.local"  # Include cluster networks
  config: |
    cluster-cidr: 10.42.0.0/16
    service-cidr: 10.43.0.0/16
```

### 5. Swap Not Disabled

K3s requires swap disabled. Provider handles this, but custom kernel configs may re-enable:
```bash
# Verify swap is off
swapon --show  # Should return empty

# Provider runs these commands:
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
swapoff -a
```

### 6. Port Conflicts

Ensure these ports are available:
- **6443**: Kubernetes API (server)
- **10250**: Kubelet metrics (all nodes)
- **2379-2380**: etcd (control plane only)
- **8472**: Flannel VXLAN (if using Flannel)
- **4789**: Flannel VXLAN alternative
- **51820**: Flannel WireGuard (if using WireGuard backend)

### 7. Config Merging Order

Provider configs override user configs (90 vs 99 prefix). To override provider settings, use ProviderOptions:
```yaml
cluster:
  provider_options:
    cluster-init: "no"  # Override provider's cluster-init setting
```

## Integration Points

### Stylus Integration

**Provider Selection**: Stylus edge agent selects provider-k3s for lightweight deployments:
- Default choice for resource-constrained edge devices
- Used when cluster spec specifies k3s distribution
- Automatically installed in Kairos image with provider-kairos package

**Edge Device Provisioning**:
1. Stylus generates cloud-init with cluster configuration
2. Device boots with Kairos + provider-k3s
3. Provider-k3s processes cluster section during boot.before stage
4. K3s starts and registers with cluster
5. Stylus monitors cluster health via Kubernetes API

**Device Registration**:
```yaml
# Generated by Stylus for edge device
cluster:
  cluster_token: "${STYLUS_CLUSTER_TOKEN}"
  control_plane_host: "${STYLUS_CONTROL_PLANE_IP}"
  role: worker
  config: |
    node-name: "${DEVICE_ID}"
    node-label:
      - "stylus.edge/site=${SITE_ID}"
      - "stylus.edge/region=${REGION}"
```

### Kairos Integration

**Provider Plugin System**:
```go
// main.go
plugin := clusterplugin.ClusterPlugin{
    Provider: provider.ClusterProvider,
}

plugin.Run(
    pluggable.FactoryPlugin{
        EventType:     clusterplugin.EventClusterReset,
        PluginHandler: handleClusterReset,
    },
)
```

**Cluster Reset Event**:
```go
// pkg/provider/reset.go
func handleClusterReset(event *pluggable.Event) pluggable.EventResponse {
    // Runs k3s-uninstall.sh or k3s-agent-uninstall.sh
    // Removes /etc/rancher/k3s
    // Cleans up /var/lib/rancher/k3s
}
```

**Yip Stage Execution**:
```go
// pkg/provider/provider.go:ClusterProvider()
cfg := yip.YipConfig{
    Name: "K3s Kairos Cluster Provider",
    Stages: map[string][]yip.Stage{
        "boot.before": parseStages(...),  // Executed before boot
    },
}
```

## Reference Examples

### Basic Single-Node Cluster

```yaml
#cloud-config
# /Users/rishi/work/src/provider-k3s/README.md (example)

cluster:
  cluster_token: K10my-secure-token-12345
  control_plane_host: 192.168.1.100
  role: init
  config: |
    write-kubeconfig-mode: "0644"
    node-name: edge-controller
```

### Multi-Node HA Cluster

```yaml
# Init node (first control plane)
cluster:
  cluster_token: K10shared-token-for-cluster
  control_plane_host: 10.0.1.100
  role: init
  config: |
    cluster-init: true
    tls-san:
      - k8s.example.com
      - 10.0.1.100
      - 10.0.1.101
      - 10.0.1.102

# Additional control plane nodes
cluster:
  cluster_token: K10shared-token-for-cluster
  control_plane_host: 10.0.1.100
  role: controlplane
  config: |
    server: https://10.0.1.100:6443

# Worker nodes
cluster:
  cluster_token: K10shared-token-for-cluster
  control_plane_host: 10.0.1.100
  role: worker
```

### Air-Gap Deployment with Local Images

```yaml
cluster:
  cluster_token: K10airgap-token-12345
  control_plane_host: 192.168.100.10
  role: init
  import_local_images: true
  local_images_path: "/opt/content/images"
  config: |
    airgap-extra-registry: registry.local:5000
```

### Proxy Environment

```yaml
cluster:
  cluster_token: K10proxy-token-12345
  control_plane_host: 10.10.1.50
  role: worker
  env:
    HTTP_PROXY: "http://proxy.corp.com:3128"
    HTTPS_PROXY: "http://proxy.corp.com:3128"
    NO_PROXY: "localhost,127.0.0.1,.local,.corp.com"
  config: |
    node-name: edge-worker-01
```

## Related Skills

- **02-configuration-patterns.md**: Detailed configuration examples and patterns
- **03-cluster-roles.md**: Deep dive into init, controlplane, and worker roles
- **04-stylus-integration.md**: How Stylus uses provider-k3s for edge deployments
- **provider-kubeadm/01-architecture.md**: Compare with upstream Kubernetes architecture
- **provider-rke2/01-architecture.md**: Compare with RKE2 architecture

## Documentation References

- **Provider README**: `/Users/rishi/work/src/provider-k3s/README.md`
- **Main Provider Logic**: `/Users/rishi/work/src/provider-k3s/pkg/provider/provider.go`
- **Configuration Types**: `/Users/rishi/work/src/provider-k3s/api/k3s_config.go`
- **Reset Handler**: `/Users/rishi/work/src/provider-k3s/pkg/provider/reset.go`
- **Constants**: `/Users/rishi/work/src/provider-k3s/pkg/constants/constants.go`
- **K3s Official Docs**: https://docs.k3s.io/
- **Kairos Documentation**: https://kairos.io/docs/
