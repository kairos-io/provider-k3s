---
skill: K3s Provider Cluster Roles
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

# K3s Provider Cluster Roles

## Overview

Provider-k3s supports three cluster roles: init (bootstrap first etcd node), controlplane (additional control plane nodes), and worker (agent-only nodes). Each role has distinct responsibilities, configuration requirements, and operational characteristics for building resilient edge Kubernetes clusters.

## Key Concepts

### Role Hierarchy

```
┌──────────────────────────────────────────────────────┐
│  Init Role (role: init)                              │
│  • First node in cluster                             │
│  • Bootstraps embedded etcd cluster                  │
│  • Becomes first control plane                        │
│  • Generates cluster certificates                     │
│  • ONE per cluster                                   │
└──────────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  ControlPlane Role (role: controlplane)              │
│  • Joins existing etcd cluster                       │
│  • Runs API server, scheduler, controller            │
│  • Participates in etcd consensus                    │
│  • Can be promoted to handle API traffic             │
│  • MULTIPLE for HA (typically 3 or 5 total)          │
└──────────────────────────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│  Worker Role (role: worker)                          │
│  • Runs k3s-agent service only                       │
│  • No etcd, API server, or controllers               │
│  • Runs kubelet and container runtime                │
│  • Connects to control plane for API requests        │
│  • MULTIPLE for workload capacity                    │
└──────────────────────────────────────────────────────┘
```

### Init Role Specifics

**Purpose**: Bootstrap the first node and initialize the etcd cluster.

**Components Running**:
- etcd (cluster datastore)
- kube-apiserver (Kubernetes API)
- kube-scheduler (pod scheduling)
- kube-controller-manager (reconciliation loops)
- kubelet (node agent)
- containerd (container runtime)
- CoreDNS (cluster DNS)
- Flannel (CNI - if not disabled)
- Traefik (ingress - if not disabled)

**Provider Configuration Logic**:
```go
// /Users/rishi/work/src/provider-k3s/pkg/provider/provider.go
case clusterplugin.RoleInit:
    k3sConfig.ClusterInit = true     // Bootstrap etcd
    k3sConfig.Server = ""            // No upstream server
    k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
    systemName = "k3s"               // Server service
```

**Key Configuration Options**:
- `cluster-init: true` - Automatically set by provider
- No `server` field - This is the first node
- TLS SANs include control_plane_host for certificate

### ControlPlane Role Specifics

**Purpose**: Add additional control plane nodes for high availability.

**Components Running**: Same as init role (etcd, API server, scheduler, controller, kubelet, etc.)

**Provider Configuration Logic**:
```go
case clusterplugin.RoleControlPlane:
    k3sConfig.ClusterInit = false    // Join existing cluster
    k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
    k3sConfig.TLSSan = []string{cluster.ControlPlaneHost}
    systemName = "k3s"               // Server service
```

**Key Configuration Options**:
- `cluster-init: false` - Join mode, not bootstrap
- `server: https://<init-node>:6443` - Points to init node
- Uses cluster_token to authenticate and join etcd

**Joining Process**:
1. Contacts init node API server using cluster_token
2. Downloads cluster CA and certificates
3. Joins etcd cluster as new member
4. Starts API server, scheduler, controller
5. Begins serving API requests

### Worker Role Specifics

**Purpose**: Run application workloads without control plane overhead.

**Components Running**:
- kubelet (node agent)
- kube-proxy (service networking)
- containerd (container runtime)
- Flannel agent (if CNI not disabled)

**Components NOT Running**:
- etcd (no datastore)
- kube-apiserver (no API serving)
- kube-scheduler (no scheduling logic)
- kube-controller-manager (no reconciliation)

**Provider Configuration Logic**:
```go
case clusterplugin.RoleWorker:
    systemName = "k3s-agent"         // Agent service (not k3s)
    k3sConfig.Server = fmt.Sprintf("https://%s:6443", cluster.ControlPlaneHost)
    // No ClusterInit or TLSSan needed
```

**Key Configuration Options**:
- `server: https://<control-plane>:6443` - API server endpoint
- Uses cluster_token to authenticate kubelet
- Significantly lower resource consumption

## Implementation Patterns

### Single-Node Development Cluster

```yaml
#cloud-config
# Simple single-node cluster for testing

cluster:
  cluster_token: K10dev-single-node-token
  control_plane_host: 192.168.1.100
  role: init
  config: |
    write-kubeconfig-mode: "0644"
    node-name: dev-all-in-one
```

**Use case**: Development, testing, CI/CD environments
**Pros**: Simple, low resource usage
**Cons**: No HA, single point of failure

### 3-Node HA Control Plane Cluster

```yaml
# Node 1 (init - bootstrap)
cluster:
  cluster_token: K10ha-cluster-shared-token
  control_plane_host: 10.0.1.10
  role: init
  config: |
    cluster-init: true
    tls-san:
      - 10.0.1.10
      - 10.0.1.11
      - 10.0.1.12
      - k8s-api.example.com

# Node 2 (controlplane - join)
cluster:
  cluster_token: K10ha-cluster-shared-token
  control_plane_host: 10.0.1.10
  role: controlplane
  config: |
    tls-san:
      - 10.0.1.10
      - 10.0.1.11
      - 10.0.1.12
      - k8s-api.example.com

# Node 3 (controlplane - join)
cluster:
  cluster_token: K10ha-cluster-shared-token
  control_plane_host: 10.0.1.10
  role: controlplane
  config: |
    tls-san:
      - 10.0.1.10
      - 10.0.1.11
      - 10.0.1.12
      - k8s-api.example.com
```

**Why 3 nodes**: etcd requires (n/2)+1 for quorum. 3 nodes tolerate 1 failure.

**Use case**: Production edge sites requiring HA
**Pros**: Tolerates 1 node failure, distributed API load
**Cons**: Higher resource usage, more complex

### Multi-Worker Edge Deployment

```yaml
# Control plane (init)
cluster:
  cluster_token: K10edge-deployment-token
  control_plane_host: 10.0.1.10
  role: init

# Workers (multiple)
cluster:
  cluster_token: K10edge-deployment-token
  control_plane_host: 10.0.1.10
  role: worker
  config: |
    node-name: edge-worker-{{.DeviceID}}
    node-label:
      - "site={{.SiteID}}"
      - "region={{.Region}}"
      - "workload=edge-inference"
```

**Use case**: Edge sites with centralized control plane, distributed workers
**Pros**: Scalable workload capacity, efficient resource use
**Cons**: Workers depend on control plane connectivity

### 5-Node HA with Dedicated Workers

```yaml
# 3 control plane nodes (init + 2 controlplane)
# ... (as in 3-node HA example)

# Worker nodes (dedicated for workloads)
cluster:
  cluster_token: K10ha-dedicated-workers-token
  control_plane_host: 10.0.1.10
  role: worker
  config: |
    node-name: worker-{{.Index}}
    node-label:
      - "node-role.kubernetes.io/worker=true"
    kubelet-arg:
      - "max-pods=200"
```

**Why separate workers**: Keep control plane lightweight, scale workload capacity independently

**Use case**: Production clusters with heavy workloads
**Pros**: Control plane isolation, independent scaling
**Cons**: More nodes, higher operational complexity

### Edge Hub-and-Spoke Model

```yaml
# Hub site (3-node HA control plane)
# Nodes 1-3: init + 2x controlplane

# Spoke sites (single control plane + workers)
# Site A:
cluster:
  cluster_token: K10spoke-site-a-token
  control_plane_host: 10.10.1.10
  role: init  # Single control plane at spoke

# Site A workers:
cluster:
  cluster_token: K10spoke-site-a-token
  control_plane_host: 10.10.1.10
  role: worker
```

**Use case**: Multi-site edge deployments with isolated clusters per site
**Pros**: Site autonomy, reduced WAN dependency
**Cons**: Per-site cluster management overhead

## Common Pitfalls

### 1. Multiple Init Nodes

**Problem**: Only ONE init node per cluster. Multiple init nodes create separate clusters.

```yaml
# BAD - Node 1
role: init

# BAD - Node 2 (creates second cluster!)
role: init

# GOOD - Node 1
role: init

# GOOD - Node 2
role: controlplane
```

### 2. Worker Node Trying to Init

**Problem**: Workers can't bootstrap etcd or run control plane.

```yaml
# BAD
cluster:
  role: worker
  config: |
    cluster-init: true  # ERROR: workers don't have etcd
```

### 3. ControlPlane Without Init

**Problem**: ControlPlane nodes need existing cluster to join.

```bash
# Deploy sequence:
# WRONG: Deploy controlplane first
kubectl apply -f controlplane-node.yaml  # Fails - no cluster yet

# RIGHT: Deploy init first, then controlplane
kubectl apply -f init-node.yaml          # Bootstraps cluster
# Wait for cluster ready
kubectl apply -f controlplane-node.yaml  # Joins successfully
```

### 4. Inconsistent cluster_token

**Problem**: All nodes must use same cluster_token.

```yaml
# Init node
cluster_token: K10token-abc123

# ControlPlane node - BAD
cluster_token: K10token-xyz789  # Different token - won't join!

# Worker node - GOOD
cluster_token: K10token-abc123  # Same token - joins successfully
```

### 5. Wrong control_plane_host for Workers

**Problem**: Workers must point to actual control plane IP, not their own IP.

```yaml
# Control plane
cluster:
  control_plane_host: 10.0.1.10
  role: init

# Worker - BAD
cluster:
  control_plane_host: 10.0.1.50  # Worker's own IP - wrong!
  role: worker

# Worker - GOOD
cluster:
  control_plane_host: 10.0.1.10  # Control plane IP - correct!
  role: worker
```

### 6. etcd Quorum Loss in HA

**Problem**: Need (n/2)+1 nodes for quorum. Losing majority breaks cluster.

```
3-node cluster: Tolerates 1 failure (2/3 = quorum)
5-node cluster: Tolerates 2 failures (3/5 = quorum)
```

**Avoid**:
- Even-numbered control plane (4 nodes = same fault tolerance as 3)
- Deploying control planes without network redundancy

### 7. Role Confusion in Config

**Problem**: Setting wrong role for node's intended purpose.

```yaml
# Node should be worker but configured as controlplane
# Results in unexpected etcd/API server running
cluster:
  role: controlplane  # Should be: worker
  config: |
    node-label:
      - "node-role.kubernetes.io/worker=true"  # Contradicts role!
```

## Role-Specific Configuration

### Init Node Best Practices

```yaml
cluster:
  role: init
  config: |
    # Required: TLS SANs for all control plane IPs
    tls-san:
      - 10.0.1.10
      - 10.0.1.11
      - 10.0.1.12
      - api.k8s.example.com

    # Recommended: Kubeconfig access
    write-kubeconfig-mode: "0644"

    # Optional: Disable unneeded components
    disable:
      - traefik

    # Optional: Custom network CIDRs
    cluster-cidr: 10.42.0.0/16
    service-cidr: 10.43.0.0/16
```

### ControlPlane Node Best Practices

```yaml
cluster:
  role: controlplane
  config: |
    # Required: Must match init node TLS SANs
    tls-san:
      - 10.0.1.10
      - 10.0.1.11
      - 10.0.1.12
      - api.k8s.example.com

    # Recommended: Kubeconfig access
    write-kubeconfig-mode: "0644"

    # Note: cluster-cidr and service-cidr inherited from init node
```

### Worker Node Best Practices

```yaml
cluster:
  role: worker
  config: |
    # Recommended: Custom node name
    node-name: worker-{{.SiteID}}-{{.Index}}

    # Recommended: Node labels for scheduling
    node-label:
      - "node-role.kubernetes.io/worker=true"
      - "workload=general"
      - "region={{.Region}}"

    # Optional: Node taints for dedicated workloads
    node-taint:
      - "workload=ml:NoSchedule"

    # Optional: Kubelet tuning
    kubelet-arg:
      - "max-pods=110"
      - "eviction-hard=memory.available<500Mi"
```

## Multi-Node Cluster Setup Patterns

### Pattern 1: Rolling Deployment

```bash
# Step 1: Deploy init node
deploy init-node.yaml
wait_for_api_ready

# Step 2: Deploy controlplane nodes (one at a time)
deploy controlplane-1.yaml
wait_for_node_ready
deploy controlplane-2.yaml
wait_for_node_ready

# Step 3: Deploy workers (parallel or sequential)
deploy worker-1.yaml
deploy worker-2.yaml
deploy worker-3.yaml
```

### Pattern 2: Phased Deployment

```bash
# Phase 1: Bootstrap control plane HA
deploy init-node.yaml
deploy controlplane-1.yaml
deploy controlplane-2.yaml
wait_for_cluster_stable

# Phase 2: Add workers in batches
for batch in $(seq 1 10); do
  deploy worker-batch-${batch}.yaml
  wait_for_batch_ready
done
```

### Pattern 3: Site-Local Deployment

```bash
# Each edge site gets own cluster
for site in site-a site-b site-c; do
  deploy ${site}-init.yaml
  wait_for_api_ready

  for worker in $(seq 1 5); do
    deploy ${site}-worker-${worker}.yaml
  done
done
```

## Integration Points

### Stylus Integration

**Role Assignment**: Stylus determines node role based on device profile:

```go
// Stylus edge controller logic (conceptual)
func assignRole(device Device) string {
    if device.IsFirstInSite() {
        return "init"
    } else if device.HasControlPlaneCapability() && needsHA() {
        return "controlplane"
    } else {
        return "worker"
    }
}
```

**Dynamic Role Configuration**:
```yaml
# Stylus-generated config template
cluster:
  cluster_token: "{{.ClusterToken}}"
  control_plane_host: "{{.ControlPlaneIP}}"
  role: "{{.AssignedRole}}"  # Dynamically determined
  config: |
    node-name: "{{.DeviceID}}"
```

### Kairos Integration

**Service Management** (/Users/rishi/work/src/provider-k3s/pkg/provider/provider.go):

```go
// Init and ControlPlane roles
systemName := "k3s"  // Server service

// Worker role
systemName := "k3s-agent"  // Agent service

// Service enable/start stages
yip.Stage{
    Name: "Enable Systemd Services",
    If:   "[ -x /bin/systemctl ]",
    Commands: []string{
        fmt.Sprintf("systemctl enable %s", systemName),
        fmt.Sprintf("systemctl restart %s", systemName),
    },
}
```

## Related Skills

- **01-architecture.md**: Overall provider architecture and components
- **02-configuration-patterns.md**: Configuration examples for each role
- **04-stylus-integration.md**: How Stylus assigns and manages roles
- **provider-kubeadm/03-cluster-roles.md**: Compare kubeadm role management

## Documentation References

- **Provider Logic**: `/Users/rishi/work/src/provider-k3s/pkg/provider/provider.go`
- **Role Handling**: Lines 36-96 in provider.go (parseOptions function)
- **K3s Server Docs**: https://docs.k3s.io/architecture#server-nodes
- **K3s Agent Docs**: https://docs.k3s.io/architecture#agent-nodes
- **K3s HA Setup**: https://docs.k3s.io/datastore/ha-embedded
