# EdgeLink Sidecar DaemonSet

This Helm chart deploys EdgeLink as a Kubernetes DaemonSet, enabling node-to-node VPN connectivity across your cluster.

## Overview

The EdgeLink Sidecar runs on every node in your Kubernetes cluster, creating a WireGuard-based mesh network between all nodes. This enables:

- **Pod-to-Pod communication** across nodes without overlay network overhead
- **Node-to-Node VPN tunnels** for secure inter-node communication
- **Hybrid cloud connectivity** connecting on-premises and cloud Kubernetes clusters
- **Edge computing** connecting edge devices to central Kubernetes infrastructure

## Prerequisites

- Kubernetes 1.20+
- Helm 3.0+
- WireGuard kernel module installed on nodes (or use userspace mode)
- EdgeLink Control Plane deployed and accessible
- Pre-shared key from EdgeLink Control Plane

## Installation

### 1. Add Helm repository (if published)

```bash
helm repo add edgelink https://charts.edgelink.io
helm repo update
```

### 2. Create namespace

```bash
kubectl create namespace edgelink-system
```

### 3. Install the chart

```bash
helm install edgelink-sidecar edgelink/edgelink-sidecar \
  --namespace edgelink-system \
  --set edgelink.serverUrl=https://api.edgelink.example.com \
  --set edgelink.preSharedKey=your-psk-here
```

### 4. Verify deployment

```bash
# Check DaemonSet status
kubectl get daemonset -n edgelink-system

# Check pod status on each node
kubectl get pods -n edgelink-system -o wide

# View logs
kubectl logs -n edgelink-system -l app.kubernetes.io/name=edgelink-sidecar
```

## Configuration

### Basic Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `namespace` | Namespace for deployment | `edgelink-system` |
| `image.repository` | Container image repository | `edgelink/edgelink-sidecar` |
| `image.tag` | Container image tag | `0.1.0` |
| `edgelink.serverUrl` | EdgeLink control plane URL | `https://api.edgelink.example.com` |
| `edgelink.preSharedKey` | Pre-shared key for registration | `change-me-in-production` |
| `edgelink.logLevel` | Log level | `info` |

### Advanced Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `hostNetwork` | Use host network namespace | `true` |
| `securityContext.privileged` | Run in privileged mode | `true` |
| `resources.limits.cpu` | CPU limit | `200m` |
| `resources.limits.memory` | Memory limit | `256Mi` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `tolerations` | Tolerations for pod assignment | See values.yaml |

### Example: Production Configuration

```yaml
# values-production.yaml
edgelink:
  serverUrl: "https://api.edgelink.production.com"
  preSharedKey: "your-production-psk"
  logLevel: "warn"

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 256Mi

nodeSelector:
  edgelink.io/enabled: "true"

# Use userspace WireGuard for better security
securityContext:
  privileged: false
  capabilities:
    add:
    - NET_ADMIN
```

Install with production values:

```bash
helm install edgelink-sidecar edgelink/edgelink-sidecar \
  --namespace edgelink-system \
  --values values-production.yaml
```

## Security Considerations

### Privileged Mode

By default, the DaemonSet runs in **privileged mode** to access the WireGuard kernel module. For enhanced security:

1. **Use WireGuard userspace mode** (wireguard-go):
   ```yaml
   securityContext:
     privileged: false
     capabilities:
       add:
       - NET_ADMIN
   ```

2. **Restrict node selection**:
   ```yaml
   nodeSelector:
     edgelink.io/sidecar: "enabled"
   ```

3. **Use Kubernetes secrets** for PSK:
   ```bash
   kubectl create secret generic edgelink-psk \
     --from-literal=psk=your-secret-key \
     -n edgelink-system
   ```

### Network Policies

Apply network policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: edgelink-sidecar-policy
  namespace: edgelink-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: edgelink-sidecar
  policyTypes:
  - Egress
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  # Allow EdgeLink control plane
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 443
  # Allow WireGuard UDP
  - ports:
    - protocol: UDP
      port: 51820
```

## Troubleshooting

### Check WireGuard Kernel Module

```bash
# On each node
lsmod | grep wireguard
```

If not loaded:
```bash
modprobe wireguard
```

### View Device Registration Status

```bash
# Get device status from any sidecar pod
kubectl exec -n edgelink-system -it <pod-name> -- edgelink-lite --status
```

### Common Issues

**Issue**: Pods stuck in CrashLoopBackOff

**Solution**: Check if WireGuard kernel module is available:
```bash
kubectl logs -n edgelink-system <pod-name>
```

**Issue**: Registration fails with authentication error

**Solution**: Verify pre-shared key:
```bash
kubectl get secret -n edgelink-system edgelink-sidecar-psk -o jsonpath='{.data.psk}' | base64 -d
```

**Issue**: Nodes cannot communicate

**Solution**: Verify firewall rules allow UDP port 51820:
```bash
# On each node
sudo ufw allow 51820/udp
# or
sudo iptables -A INPUT -p udp --dport 51820 -j ACCEPT
```

### Debug Mode

Enable debug logging:

```bash
helm upgrade edgelink-sidecar edgelink/edgelink-sidecar \
  --namespace edgelink-system \
  --set edgelink.logLevel=debug \
  --reuse-values
```

## Uninstallation

```bash
# Remove Helm release
helm uninstall edgelink-sidecar -n edgelink-system

# Remove namespace (optional)
kubectl delete namespace edgelink-system
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │   Node 1     │         │   Node 2     │                  │
│  │              │         │              │                  │
│  │  ┌────────┐  │         │  ┌────────┐  │                  │
│  │  │EdgeLink│◄─┼─────────┼─►│EdgeLink│  │                  │
│  │  │Sidecar │  │ WireGuard │Sidecar │  │                  │
│  │  │  Pod   │  │  Tunnel   │  Pod   │  │                  │
│  │  └───┬────┘  │         │  └───┬────┘  │                  │
│  │      │       │         │      │       │                  │
│  │  ┌───▼────┐  │         │  ┌───▼────┐  │                  │
│  │  │App Pods│  │         │  │App Pods│  │                  │
│  │  └────────┘  │         │  └────────┘  │                  │
│  └──────────────┘         └──────────────┘                  │
│         │                        │                           │
│         │  Registration &        │                           │
│         │  Configuration         │                           │
│         └────────┬───────────────┘                           │
│                  │                                            │
└──────────────────┼────────────────────────────────────────────┘
                   │
                   ▼
         ┌─────────────────┐
         │  EdgeLink       │
         │  Control Plane  │
         │  (External)     │
         └─────────────────┘
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/edgelink/edge-link/issues
- Documentation: https://docs.edgelink.io
- Email: support@edgelink.com
