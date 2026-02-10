---
skill: K3s Provider Configuration Patterns
description: Skill documentation for provider-k3s
type: general
repository: provider-k3s
team: edge
topics: [kubernetes, provider, edge, cluster]
difficulty: intermediate
last_updated: 2026-02-09
related_skills: []
memory_references: []
---

# K3s Provider Configuration Patterns

## Overview

This guide covers configuration patterns for provider-k3s, including cloud-init structure, custom configuration injection, network settings, and advanced scenarios for production edge deployments.

## Key Concepts

### Cloud-Init Cluster Section

Provider-k3s reads configuration from the `cluster` section of cloud-init:

```yaml
cluster:
  cluster_token: string          # Required: Shared secret for cluster membership
  control_plane_host: string     # Required: Control plane IP/hostname
  role: init|controlplane|worker # Required: Node role
  config: |                      # Optional: K3s-specific YAML configuration
    key: value
  env:                           # Optional: Environment variables (proxy, etc.)
    HTTP_PROXY: string
  import_local_images: bool      # Optional: Import container images on boot
  local_images_path: string      # Optional: Path to local images
  provider_options:              # Optional: Provider-specific overrides
    key: value
```

### Configuration File Hierarchy

K3s configuration is merged from multiple sources in order:

1. **Provider Defaults** (lowest priority)
2. **User config** (90_userdata.yaml - from cluster.config)
3. **Provider overrides** (99_userdata.yaml - from provider logic)
4. **Merged final** (/etc/rancher/k3s/config.yaml)

### K3s Configuration Format

K3s uses YAML configuration compatible with CLI flags:

```yaml
# Server options (control plane)
cluster-init: true
server: https://control-plane:6443
token: secrettoken
tls-san:
  - hostname.example.com
write-kubeconfig-mode: "0644"
node-name: my-node

# Network options
cluster-cidr: 10.42.0.0/16
service-cidr: 10.43.0.0/16
cluster-dns: 10.43.0.10

# Agent options (worker)
node-label:
  - "region=us-west"
  - "environment=production"
kubelet-arg:
  - "max-pods=50"
```

## Implementation Patterns

### Basic Init Node Configuration

```yaml
#cloud-config
hostname: control-plane-01

cluster:
  cluster_token: K10f3e1a2b3c4d5e6f7g8h9i0j1k2l3m4
  control_plane_host: 192.168.1.100
  role: init
  config: |
    write-kubeconfig-mode: "0644"
    disable:
      - traefik  # Disable default ingress controller
    tls-san:
      - k8s-api.example.com
      - 192.168.1.100
```

**What happens**:
1. Provider sets `cluster-init: true` automatically
2. K3s bootstraps embedded etcd cluster
3. TLS certificates include control_plane_host + user tls-san entries
4. Kubeconfig written to /etc/rancher/k3s/k3s.yaml (mode 0644)
5. Traefik ingress controller disabled per user config

### Control Plane Join Configuration

```yaml
#cloud-config
hostname: control-plane-02

cluster:
  cluster_token: K10f3e1a2b3c4d5e6f7g8h9i0j1k2l3m4
  control_plane_host: 192.168.1.100
  role: controlplane
  config: |
    write-kubeconfig-mode: "0644"
    tls-san:
      - k8s-api.example.com
      - 192.168.1.100
      - 192.168.1.101
```

**What happens**:
1. Provider sets `server: https://192.168.1.100:6443`
2. Node joins existing etcd cluster using cluster_token
3. Becomes full control plane with API server, scheduler, controller
4. Shares etcd data with other control plane nodes

### Worker Node Configuration

```yaml
#cloud-config
hostname: worker-01

cluster:
  cluster_token: K10f3e1a2b3c4d5e6f7g8h9i0j1k2l3m4
  control_plane_host: 192.168.1.100
  role: worker
  config: |
    node-name: edge-worker-site-a-01
    node-label:
      - "site=site-a"
      - "zone=az1"
      - "workload=general"
    kubelet-arg:
      - "max-pods=110"
      - "eviction-hard=memory.available<500Mi"
```

**What happens**:
1. Provider configures k3s-agent service (not k3s server)
2. Kubelet connects to control plane API at 192.168.1.100:6443
3. Node registers with custom name and labels
4. Kubelet uses custom max-pods and eviction settings

### Custom Network Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10network-custom-token
  control_plane_host: 10.0.0.10
  role: init
  config: |
    cluster-cidr: 10.52.0.0/16       # Pod network
    service-cidr: 10.53.0.0/16       # Service network
    cluster-dns: 10.53.0.10          # CoreDNS ClusterIP
    flannel-backend: wireguard       # Use WireGuard instead of VXLAN
```

**Use case**: Custom network ranges to avoid conflicts with existing infrastructure.

**Important**: If using proxy, ensure these CIDRs are in NO_PROXY.

### Disable Built-in Components

```yaml
#cloud-config
cluster:
  cluster_token: K10minimal-cluster-token
  control_plane_host: 192.168.10.10
  role: init
  config: |
    disable:
      - traefik          # Disable Traefik ingress
      - servicelb        # Disable Klipper service LB
      - local-storage    # Disable local-path provisioner
      - metrics-server   # Disable metrics server
    flannel-backend: none  # Disable Flannel (use custom CNI)
```

**Use case**: Minimal cluster for custom networking (Calico, Cilium) or ingress (NGINX, Istio).

### Node Taints and Labels

```yaml
#cloud-config
cluster:
  cluster_token: K10taints-example-token
  control_plane_host: 192.168.1.100
  role: worker
  config: |
    node-name: gpu-worker-01
    node-label:
      - "gpu=nvidia-t4"
      - "accelerator=true"
      - "workload=ml-inference"
    node-taint:
      - "gpu=true:NoSchedule"  # Only pods with toleration can schedule
```

**Use case**: Dedicated nodes for specific workloads (GPU, high-memory, storage).

### Kubelet Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10kubelet-config-token
  control_plane_host: 192.168.1.100
  role: worker
  config: |
    kubelet-arg:
      - "max-pods=200"
      - "pod-max-pids=4096"
      - "eviction-hard=memory.available<1Gi,nodefs.available<10%"
      - "eviction-soft=memory.available<2Gi,nodefs.available<15%"
      - "eviction-soft-grace-period=memory.available=2m,nodefs.available=2m"
      - "image-gc-high-threshold=85"
      - "image-gc-low-threshold=80"
```

**Use case**: Fine-tune kubelet for high-density or resource-constrained nodes.

### Private Registry Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10registry-example-token
  control_plane_host: 192.168.1.100
  role: worker
  config: |
    node-name: worker-private-registry

# Containerd registry config (separate from cluster section)
write_files:
  - path: /etc/rancher/k3s/registries.yaml
    permissions: "0644"
    content: |
      mirrors:
        docker.io:
          endpoint:
            - "https://registry.example.com"
        registry.example.com:
          endpoint:
            - "https://registry.example.com"
      configs:
        "registry.example.com":
          auth:
            username: myuser
            password: mypassword
          tls:
            cert_file: /etc/ssl/certs/registry.crt
            key_file: /etc/ssl/certs/registry.key
            ca_file: /etc/ssl/certs/ca.crt
```

**Use case**: Pull images from private registry or mirror public registries.

### Air-Gap Deployment

```yaml
#cloud-config
cluster:
  cluster_token: K10airgap-deployment-token
  control_plane_host: 192.168.100.10
  role: init
  import_local_images: true
  local_images_path: "/opt/airgap/images"
  config: |
    disable-cloud-controller: true
    node-name: airgap-control-01

stages:
  boot:
    - name: "Import K3s Images"
      commands:
        - "k3s ctr images import /opt/airgap/k3s-images.tar"
        - "k3s ctr images import /opt/airgap/app-images.tar"
```

**Use case**: Edge deployments without internet connectivity.

**Prerequisites**:
1. K3s images tarball at /opt/airgap/images/
2. Import script at /opt/k3s/scripts/import.sh
3. Provider automatically runs import during boot.before stage

### HA Cluster with External etcd

K3s supports external etcd for true HA (not embedded etcd):

```yaml
#cloud-config
cluster:
  cluster_token: K10external-etcd-token
  control_plane_host: 192.168.1.100
  role: init
  config: |
    datastore-endpoint: "https://etcd1.example.com:2379,https://etcd2.example.com:2379,https://etcd3.example.com:2379"
    datastore-cafile: /etc/etcd/ca.pem
    datastore-certfile: /etc/etcd/client.pem
    datastore-keyfile: /etc/etcd/client-key.pem
```

**Use case**: Separation of control plane and data plane for large clusters.

### Proxy Environment Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10proxy-environment-token
  control_plane_host: 10.10.1.100
  role: worker
  env:
    HTTP_PROXY: "http://proxy.corp.com:3128"
    HTTPS_PROXY: "http://proxy.corp.com:3128"
    NO_PROXY: "localhost,127.0.0.1,.local,.corp.com,10.0.0.0/8"
  config: |
    node-name: worker-behind-proxy
    cluster-cidr: 10.42.0.0/16
    service-cidr: 10.43.0.0/16
```

**Generated proxy file** (/etc/default/k3s-agent):
```bash
HTTP_PROXY=http://proxy.corp.com:3128
HTTPS_PROXY=http://proxy.corp.com:3128
CONTAINERD_HTTP_PROXY=http://proxy.corp.com:3128
CONTAINERD_HTTPS_PROXY=http://proxy.corp.com:3128
NO_PROXY=10.42.0.0/16,10.43.0.0/16,10.10.1.0/24,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.local,.corp.com,10.0.0.0/8
CONTAINERD_NO_PROXY=10.42.0.0/16,10.43.0.0/16,10.10.1.0/24,.svc,.svc.cluster,.svc.cluster.local,localhost,127.0.0.1,.local,.corp.com,10.0.0.0/8
```

**Provider automatically adds**:
- cluster-cidr
- service-cidr
- Node CIDR (detected from network interface)
- .svc, .svc.cluster, .svc.cluster.local

### Custom kube-proxy Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10kube-proxy-config-token
  control_plane_host: 192.168.1.100
  role: init
  config: |
    kube-proxy-arg:
      - "proxy-mode=ipvs"           # Use IPVS instead of iptables
      - "ipvs-scheduler=rr"         # Round-robin scheduling
      - "ipvs-strict-arp=true"      # Enable strict ARP (for MetalLB)
      - "metrics-bind-address=0.0.0.0:10249"
```

**Use case**: IPVS mode for better performance with many services, or MetalLB integration.

### Custom DNS Configuration

```yaml
#cloud-config
cluster:
  cluster_token: K10dns-config-token
  control_plane_host: 192.168.1.100
  role: init
  config: |
    cluster-dns: 10.43.0.10
    cluster-domain: cluster.local
    resolv-conf: /etc/k3s-resolv.conf  # Custom resolv.conf for kubelet

write_files:
  - path: /etc/k3s-resolv.conf
    permissions: "0644"
    content: |
      nameserver 8.8.8.8
      nameserver 8.8.4.4
      search cluster.local svc.cluster.local corp.example.com
```

**Use case**: Custom DNS resolution for hybrid cloud/on-prem environments.

### Secrets Encryption at Rest

```yaml
#cloud-config
cluster:
  cluster_token: K10secrets-encryption-token
  control_plane_host: 192.168.1.100
  role: init
  config: |
    secrets-encryption: true  # Enable AES-CBC encryption for secrets

stages:
  boot:
    - name: "Verify secrets encryption"
      commands:
        - "k3s kubectl get secrets -A -o json | jq '.items[] | select(.type==\"Opaque\")' | head -1"
```

**Use case**: Compliance requirements for encrypting secrets at rest in etcd.

### Advanced: Multi-Interface Node

```yaml
#cloud-config
cluster:
  cluster_token: K10multi-interface-token
  control_plane_host: 192.168.1.100
  role: worker
  config: |
    node-ip: 192.168.1.50              # Primary interface for k8s traffic
    node-external-ip: 203.0.113.50     # Public IP for external access
    flannel-iface: eth1                 # Interface for Flannel overlay
    bind-address: 0.0.0.0               # Listen on all interfaces
```

**Use case**: Nodes with multiple network interfaces (public/private, management/data).

## Common Pitfalls

### 1. Config Merge Order Confusion

User config (90_userdata.yaml) is overridden by provider config (99_userdata.yaml):

```yaml
# User sets this in cluster.config
cluster-init: false

# Provider generates 99_userdata.yaml with
cluster-init: true  # This wins!
```

**Solution**: Use provider_options to override provider settings:
```yaml
cluster:
  provider_options:
    cluster-init: "no"
```

### 2. Missing Token Requirements

K3s token must be valid bootstrap token format or 16+ character string:

```yaml
# BAD - too short
cluster_token: abc

# BAD - invalid format
cluster_token: "contains spaces"

# GOOD - random string 16+ chars
cluster_token: K10abcdef1234567890

# GOOD - bootstrap token format
cluster_token: abcdef.0123456789abcdef
```

### 3. TLS SAN Not Including All Access Methods

If accessing API via multiple hostnames/IPs, all must be in tls-san:

```yaml
# Init node accessible via:
# - 192.168.1.100 (primary IP)
# - 10.0.1.100 (VPN IP)
# - k8s.example.com (DNS)
# - load-balancer.local (load balancer)

cluster:
  control_plane_host: 192.168.1.100  # Automatically added to tls-san
  config: |
    tls-san:
      - 10.0.1.100
      - k8s.example.com
      - load-balancer.local
```

### 4. Proxy Not Excluding Cluster Networks

Forgetting to exclude cluster CIDRs from proxy causes pod networking failures:

```yaml
# BAD - cluster CIDRs proxied
env:
  HTTP_PROXY: http://proxy:3128
  NO_PROXY: localhost  # Missing cluster networks!

# GOOD - provider auto-adds cluster CIDRs
env:
  HTTP_PROXY: http://proxy:3128
  NO_PROXY: localhost,127.0.0.1  # Provider adds cluster-cidr, service-cidr, node CIDR
```

### 5. Disabling Flannel Without Alternative CNI

```yaml
# BAD - no CNI, pods won't communicate
config: |
  flannel-backend: none

# GOOD - install alternative CNI
config: |
  flannel-backend: none

stages:
  boot:
    - name: "Install Calico CNI"
      commands:
        - "kubectl apply -f /opt/calico/calico.yaml"
```

### 6. Worker Node with cluster-init

```yaml
# BAD - workers can't bootstrap etcd
cluster:
  role: worker
  config: |
    cluster-init: true  # ERROR: workers don't have etcd

# GOOD - workers join existing cluster
cluster:
  role: worker
  # No cluster-init needed, provider sets server URL automatically
```

## Integration Points

### Stylus Integration

**Generated Configuration**: Stylus dynamically generates cluster config for edge devices:

```yaml
# Template used by Stylus
cluster:
  cluster_token: "{{.ClusterToken}}"
  control_plane_host: "{{.ControlPlaneEndpoint}}"
  role: "{{.NodeRole}}"
  config: |
    node-name: "{{.DeviceID}}"
    node-label:
      - "stylus.edge/site={{.SiteID}}"
      - "stylus.edge/region={{.Region}}"
      - "stylus.edge/device-type={{.DeviceType}}"
    {{- if .CustomConfig }}
    {{.CustomConfig}}
    {{- end}}
```

**Stylus-Specific Labels**: Device metadata encoded as node labels for scheduling:

```yaml
node-label:
  - "stylus.edge/site=factory-01"
  - "stylus.edge/region=us-west"
  - "stylus.edge/device-type=gateway"
  - "stylus.edge/firmware-version=2.1.0"
```

### Kairos Integration

**Provider Options**: Kairos-specific options passed via provider_options:

```yaml
cluster:
  provider_options:
    cluster-root-path: "/run/initramfs/live"  # Custom root path for scripts
```

**File Structure** (/Users/rishi/work/src/provider-k3s/pkg/provider/provider.go):
```go
func parseFiles(cluster clusterplugin.Cluster, systemName string) []yip.File {
    options, proxyOptions, userOptions := parseOptions(cluster)

    files := []yip.File{
        {
            Path:        filepath.Join(configurationPath, "90_userdata.yaml"),
            Permissions: 0400,
            Content:     string(userOptions),  // From cluster.config
        },
        {
            Path:        filepath.Join(configurationPath, "99_userdata.yaml"),
            Permissions: 0400,
            Content:     string(options),      // Provider-generated
        },
    }
    // ... proxy files
}
```

## Reference Examples

### Production Edge Deployment

```yaml
#cloud-config
# Production-grade edge node configuration
hostname: edge-{{.SiteID}}-{{.NodeIndex}}

users:
  - name: admin
    groups: [sudo]
    ssh_authorized_keys:
      - ssh-rsa AAAA...

cluster:
  cluster_token: K10prod-edge-cluster-secure-token-12345
  control_plane_host: 10.0.1.10
  role: worker
  env:
    HTTP_PROXY: "http://proxy.corp:3128"
    HTTPS_PROXY: "http://proxy.corp:3128"
  config: |
    node-name: edge-site-{{.SiteID}}-{{.NodeIndex}}
    node-label:
      - "site={{.SiteID}}"
      - "region={{.Region}}"
      - "environment=production"
      - "workload=edge-inference"
    node-taint:
      - "edge=true:NoSchedule"
    kubelet-arg:
      - "max-pods=50"
      - "eviction-hard=memory.available<256Mi,nodefs.available<10%"
      - "system-reserved=cpu=500m,memory=512Mi"
      - "kube-reserved=cpu=500m,memory=512Mi"

write_files:
  - path: /etc/rancher/k3s/registries.yaml
    content: |
      mirrors:
        docker.io:
          endpoint:
            - "https://registry.{{.Region}}.corp.com"

stages:
  boot:
    - name: "Configure monitoring"
      commands:
        - "systemctl enable node-exporter"
        - "systemctl start node-exporter"
```

### Configuration File Locations

- **User Config**: `/etc/rancher/k3s/config.d/90_userdata.yaml`
- **Provider Config**: `/etc/rancher/k3s/config.d/99_userdata.yaml`
- **Merged Config**: `/etc/rancher/k3s/config.yaml`
- **Proxy Config**: `/etc/default/k3s` or `/etc/default/k3s-agent`
- **Registry Config**: `/etc/rancher/k3s/registries.yaml`
- **Kubeconfig**: `/etc/rancher/k3s/k3s.yaml`

## Related Skills

- **01-architecture.md**: Provider architecture and component overview
- **03-cluster-roles.md**: Role-specific configurations (init, controlplane, worker)
- **04-stylus-integration.md**: Stylus-specific configuration patterns
- **provider-rke2/02-configuration-patterns.md**: RKE2 configuration comparison

## Documentation References

- **Provider Code**: `/Users/rishi/work/src/provider-k3s/pkg/provider/provider.go`
- **Configuration Types**: `/Users/rishi/work/src/provider-k3s/api/k3s_config.go`
- **Agent Config**: `/Users/rishi/work/src/provider-k3s/api/agent_config.go`
- **K3s Server Docs**: https://docs.k3s.io/reference/server-config
- **K3s Agent Docs**: https://docs.k3s.io/reference/agent-config
- **K3s Networking**: https://docs.k3s.io/networking
