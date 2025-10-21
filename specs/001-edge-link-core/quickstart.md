# Quickstart Guide: Edge-Link Core System

**Feature**: Edge-Link Core System
**Branch**: 001-edge-link-core
**Date**: 2025-10-19

## Purpose

This guide provides step-by-step instructions for deploying Edge-Link control plane, connecting clients, and validating the system. Target audience: DevOps engineers and system administrators.

---

## Prerequisites

### Infrastructure Requirements

- **Kubernetes Cluster**: 1.25+ with 3+ nodes (for HA control plane)
- **PostgreSQL**: 14+ with 100GB storage, 4 vCPU, 16GB RAM minimum
- **Redis**: 7+ with Sentinel HA (3 instances recommended)
- **S3-Compatible Storage**: MinIO or AWS S3 for diagnostic bundles
- **STUN/TURN Servers**: 1+ STUN server, 1+ TURN server (coturn recommended)
- **Domain Names**: Wildcard DNS for `*.edgelink.example.com` pointing to ingress
- **TLS Certificates**: Valid certificates for API endpoints (Let's Encrypt or corporate CA)

### Local Tools

- `kubectl` 1.25+
- `helm` 3.10+
- `docker` 20.10+ (for local development)
- `openssl` for certificate management

---

## Step 1: Deploy Infrastructure Components

### 1.1 PostgreSQL

**Option A: Kubernetes (using Bitnami Helm chart)**

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

helm install edge-link-postgres bitnami/postgresql \
  --namespace edge-link --create-namespace \
  --set auth.username=edgelink \
  --set auth.password=CHANGE_ME \
  --set auth.database=edgelink \
  --set primary.persistence.size=100Gi \
  --set primary.resources.requests.cpu=2 \
  --set primary.resources.requests.memory=8Gi
```

**Option B: Managed Service (AWS RDS, GCP Cloud SQL)**

Export connection string for next steps:

```bash
export DATABASE_URL="postgresql://edgelink:password@postgres-endpoint:5432/edgelink?sslmode=require"
```

### 1.2 Redis

**Kubernetes deployment with Sentinel:**

```bash
helm install edge-link-redis bitnami/redis \
  --namespace edge-link \
  --set architecture=replication \
  --set auth.password=CHANGE_ME \
  --set sentinel.enabled=true \
  --set replica.replicaCount=2
```

Export Redis connection:

```bash
export REDIS_URL="redis://:password@edge-link-redis-master.edge-link.svc.cluster.local:6379"
```

### 1.3 S3 Storage (MinIO)

```bash
helm install edge-link-minio bitnami/minio \
  --namespace edge-link \
  --set auth.rootUser=admin \
  --set auth.rootPassword=CHANGE_ME \
  --set persistence.size=500Gi

# Create bucket for diagnostics
kubectl run -i --tty minio-client --image=minio/mc --restart=Never -- \
  mc alias set edgelink http://edge-link-minio:9000 admin CHANGE_ME && \
  mc mb edgelink/diagnostics
```

### 1.4 STUN/TURN Server (coturn)

Deploy coturn with custom configuration:

```yaml
# coturn-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coturn-config
  namespace: edge-link
data:
  turnserver.conf: |
    listening-port=3478
    relay-ip=<EXTERNAL_IP>
    external-ip=<EXTERNAL_IP>
    realm=edgelink.example.com
    server-name=turn1.edgelink.example.com
    lt-cred-mech
    user=edgelink:CHANGE_ME
    no-multicast-peers
    no-cli
    log-file=stdout

---
apiVersion: v1
kind: Service
metadata:
  name: coturn
  namespace: edge-link
spec:
  type: LoadBalancer
  ports:
    - name: turn-udp
      port: 3478
      protocol: UDP
    - name: turn-tcp
      port: 3478
      protocol: TCP
  selector:
    app: coturn

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coturn
  namespace: edge-link
spec:
  replicas: 2
  selector:
    matchLabels:
      app: coturn
  template:
    metadata:
      labels:
        app: coturn
    spec:
      containers:
      - name: coturn
        image: coturn/coturn:4.6
        ports:
        - containerPort: 3478
          protocol: UDP
        - containerPort: 3478
          protocol: TCP
        volumeMounts:
        - name: config
          mountPath: /etc/coturn/turnserver.conf
          subPath: turnserver.conf
      volumes:
      - name: config
        configMap:
          name: coturn-config
```

```bash
kubectl apply -f coturn-config.yaml
```

---

## Step 2: Deploy Edge-Link Control Plane

### 2.1 Configure Helm Values

Create `values-prod.yaml`:

```yaml
# Global configuration
global:
  domain: edgelink.example.com
  environment: production

# Database configuration
database:
  host: edge-link-postgres-postgresql.edge-link.svc.cluster.local
  port: 5432
  username: edgelink
  password: CHANGE_ME
  database: edgelink
  sslMode: require

# Redis configuration
redis:
  host: edge-link-redis-master.edge-link.svc.cluster.local
  port: 6379
  password: CHANGE_ME

# S3 configuration
storage:
  endpoint: http://edge-link-minio:9000
  accessKey: admin
  secretKey: CHANGE_ME
  bucket: diagnostics
  region: us-east-1

# STUN/TURN servers
natTraversal:
  stunServers:
    - stun1.edgelink.example.com:3478
  turnServers:
    - url: turn1.edgelink.example.com:3478
      username: edgelink
      credential: CHANGE_ME

# Service configurations
apiGateway:
  replicas: 3
  resources:
    requests:
      cpu: 1
      memory: 2Gi
    limits:
      cpu: 2
      memory: 4Gi

deviceService:
  replicas: 3
  resources:
    requests:
      cpu: 500m
      memory: 1Gi

topologyService:
  replicas: 2
  resources:
    requests:
      cpu: 500m
      memory: 1Gi

natCoordinator:
  replicas: 2
  resources:
    requests:
      cpu: 1
      memory: 2Gi

alertService:
  replicas: 2
  resources:
    requests:
      cpu: 250m
      memory: 512Mi
  smtp:
    host: smtp.example.com
    port: 587
    username: alerts@edgelink.example.com
    password: CHANGE_ME
    from: alerts@edgelink.example.com

backgroundWorker:
  replicas: 1
  resources:
    requests:
      cpu: 250m
      memory: 512Mi

# Ingress configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
    - hosts:
        - api.edgelink.example.com
      secretName: edgelink-tls

# Monitoring
prometheus:
  enabled: true
  serviceMonitor:
    enabled: true
```

### 2.2 Install Control Plane

```bash
# Add Edge-Link Helm repository (assume custom repo)
helm repo add edgelink https://helm.edgelink.example.com
helm repo update

# Install control plane
helm install edge-link-control-plane edgelink/edge-link-control-plane \
  --namespace edge-link \
  --values values-prod.yaml \
  --wait --timeout 10m

# Verify deployment
kubectl get pods -n edge-link
```

Expected output:
```
NAME                                  READY   STATUS    RESTARTS   AGE
api-gateway-xxxxxxxxxx-xxxxx          1/1     Running   0          2m
device-service-xxxxxxxxxx-xxxxx       1/1     Running   0          2m
topology-service-xxxxxxxxxx-xxxxx     1/1     Running   0          2m
nat-coordinator-xxxxxxxxxx-xxxxx      1/1     Running   0          2m
alert-service-xxxxxxxxxx-xxxxx        1/1     Running   0          2m
background-worker-xxxxxxxxxx-xxxxx    1/1     Running   0          2m
```

### 2.3 Run Database Migrations

```bash
# Get migration job pod name
kubectl get jobs -n edge-link

# Check migration logs
kubectl logs -n edge-link job/edge-link-migrations

# Verify tables created
kubectl exec -it deployment/edge-link-postgres-postgresql -n edge-link -- \
  psql -U edgelink -d edgelink -c "\dt"
```

---

## Step 3: Create Organization and Virtual Network

### 3.1 Access Management UI

```bash
# Port-forward to API Gateway (development)
kubectl port-forward -n edge-link svc/api-gateway 8080:80

# Or visit production URL
open https://ui.edgelink.example.com
```

### 3.2 Create Organization via API (or UI)

```bash
# Generate admin token (OIDC flow in production, simplified for quickstart)
export ADMIN_TOKEN="your-oidc-token-here"

# Create organization
curl -X POST https://api.edgelink.example.com/api/v1/organizations \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACME Corporation",
    "slug": "acme-corp",
    "max_devices": 1000,
    "max_virtual_networks": 10
  }'

# Response:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "name": "ACME Corporation",
#   "slug": "acme-corp",
#   "created_at": "2025-10-19T10:00:00Z"
# }
```

### 3.3 Create Virtual Network

```bash
export ORG_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X POST https://api.edgelink.example.com/api/v1/virtual-networks \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": "'$ORG_ID'",
    "name": "Engineering VPN",
    "cidr": "10.100.0.0/16",
    "gateway_ip": "10.100.0.1",
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  }'

# Save virtual_network_id from response
export VNET_ID="<returned-uuid>"
```

### 3.4 Generate Pre-Shared Key

```bash
curl -X POST https://api.edgelink.example.com/api/v1/pre-shared-keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": "'$ORG_ID'",
    "name": "Office Devices",
    "max_uses": 100,
    "expires_at": "2026-10-19T00:00:00Z"
  }'

# Save key from response (shown once only)
export PSK="psk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

---

## Step 4: Connect Desktop Client

### 4.1 Install Client

**Linux (Debian/Ubuntu)**:

```bash
wget https://releases.edgelink.example.com/edgelink-cli_1.0.0_amd64.deb
sudo dpkg -i edgelink-cli_1.0.0_amd64.deb
```

**macOS (Homebrew)**:

```bash
brew tap edgelink/tap
brew install edgelink-cli
```

**Windows (Installer)**:

Download and run `EdgeLink-1.0.0-Setup.exe` from releases page.

### 4.2 Register Device

```bash
# Initialize configuration
edgelink-cli init \
  --server https://api.edgelink.example.com \
  --psk $PSK \
  --virtual-network $VNET_ID \
  --name "Dev Laptop"

# Expected output:
# ✓ Device registered successfully
# Device ID: 123e4567-e89b-12d3-a456-426614174000
# Virtual IP: 10.100.1.42
# Configuration saved to: ~/.config/edgelink/config.yaml
```

### 4.3 Start Daemon

```bash
# Start in foreground (testing)
sudo edgelink-daemon --log-level debug

# Or install as systemd service (Linux)
sudo edgelink-cli install-service
sudo systemctl start edgelink
sudo systemctl enable edgelink

# Check status
sudo systemctl status edgelink
```

### 4.4 Verify Connectivity

```bash
# Check WireGuard interface
sudo wg show

# Expected output:
# interface: wg0
#   public key: <device-public-key>
#   private key: (hidden)
#   listening port: 51820
#
# peer: <peer-public-key>
#   endpoint: 203.0.113.42:51820
#   allowed ips: 10.100.1.43/32
#   latest handshake: 15 seconds ago
#   transfer: 1.24 MiB received, 845.12 KiB sent

# Ping peer device
ping 10.100.1.43
```

---

## Step 5: Monitoring and Operations

### 5.1 Access Grafana Dashboards

```bash
# Port-forward Grafana (if deployed with monitoring stack)
kubectl port-forward -n monitoring svc/grafana 3000:80

# Visit http://localhost:3000
# Default credentials: admin / admin (change immediately)
```

**Pre-built Dashboards**:
- **Control Plane Overview**: API latency, request rate, error rate
- **Device Health**: Online/offline devices, NAT type distribution, P2P success rate
- **Tunnel Metrics**: Active sessions, bandwidth, latency distribution
- **Alerts Dashboard**: Active alerts by severity, resolution time

### 5.2 Query Metrics via Prometheus

```bash
# Top 10 devices by bandwidth
curl 'http://prometheus:9090/api/v1/query' --data-urlencode 'query=topk(10, rate(wireguard_bytes_sent_total[5m]))'

# Device registration rate (last hour)
curl 'http://prometheus:9090/api/v1/query' --data-urlencode 'query=rate(device_registrations_total[1h])'
```

### 5.3 View Audit Logs

```bash
# Query via API
curl 'https://api.edgelink.example.com/api/v1/audit-logs?organization_id='$ORG_ID'&start_date=2025-10-19T00:00:00Z' \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Or query PostgreSQL directly
kubectl exec -it deployment/edge-link-postgres-postgresql -n edge-link -- \
  psql -U edgelink -d edgelink -c \
  "SELECT created_at, action, resource_type, actor_id FROM audit_logs ORDER BY created_at DESC LIMIT 10;"
```

---

## Step 6: Troubleshooting

### 6.1 Device Registration Fails

**Symptom**: `401 Unauthorized` during registration

**Checks**:
```bash
# Verify PSK is valid and not expired
curl 'https://api.edgelink.example.com/api/v1/pre-shared-keys/'$PSK_ID \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Check PSK usage count vs max_uses
# Response should show: "used_count": X, "max_uses": Y where X < Y
```

**Solution**: Generate new PSK if expired or exhausted.

### 6.2 Tunnel Fails to Establish

**Symptom**: Devices registered but cannot ping each other

**Checks**:
```bash
# Check NAT type on both devices
edgelink-cli status --json | jq '.nat_type'

# If both symmetric NAT, verify TURN server is reachable
nc -zvu turn1.edgelink.example.com 3478

# Check WireGuard handshake
sudo wg show wg0 latest-handshakes
```

**Solution**: If handshake timestamp stale (> 3 minutes), check firewall rules allow UDP 51820.

### 6.3 High Latency

**Symptom**: Latency > 200ms between devices

**Checks**:
```bash
# Check if using TURN relay (higher latency expected)
edgelink-cli status --sessions

# If P2P direct, check baseline latency without WireGuard
ping -c 5 <peer-public-ip>
```

**Solution**: If baseline latency high, devices may be geographically distant. Consider regional control plane deployment.

### 6.4 Control Plane Unresponsive

**Checks**:
```bash
# Check pod health
kubectl get pods -n edge-link

# Check database connectivity
kubectl logs -n edge-link deployment/device-service --tail=50 | grep -i "database"

# Check Redis connectivity
kubectl logs -n edge-link deployment/api-gateway --tail=50 | grep -i "redis"
```

**Solution**: Restart affected services if database connection lost:
```bash
kubectl rollout restart deployment/device-service -n edge-link
```

---

## Step 7: Next Steps

### Production Hardening

1. **Enable RBAC**: Configure OIDC/SAML for management UI authentication
2. **Backup Strategy**: Setup automated PostgreSQL backups (pg_dump or cloud provider snapshots)
3. **TLS Certificates**: Rotate certificates via cert-manager before expiry
4. **Rate Limiting**: Configure API rate limits per organization (default: 1000 req/min)
5. **Security Scan**: Run `trivy` on container images, `gosec` on Go code

### Scaling

1. **Horizontal Scaling**: Increase replicas for high-traffic services:
   ```bash
   kubectl scale deployment/device-service -n edge-link --replicas=5
   ```
2. **Database Sharding**: If exceeding 1M devices, consider Citus or manual sharding by organization
3. **Multi-Region**: Deploy control plane in multiple regions with geo-DNS routing

### Advanced Features

1. **Custom Alerting**: Configure alert-service webhooks to integrate with PagerDuty, Slack, Teams
2. **Log Aggregation**: Ship logs to Loki/ELK for centralized search and analysis
3. **Traffic Shaping**: Implement QoS rules in WireGuard for bandwidth prioritization

---

## Summary

You now have a functional Edge-Link deployment with:
- ✅ Control plane services running in Kubernetes
- ✅ PostgreSQL database with schema migrations
- ✅ Redis cache for session state
- ✅ STUN/TURN servers for NAT traversal
- ✅ Organization and virtual network created
- ✅ At least one device registered and connected
- ✅ Monitoring via Prometheus and Grafana

**Next**: Connect additional devices, configure alerts, and customize for your organization's needs.

For detailed API documentation, see [contracts/control-plane-api-v1.yaml](./contracts/control-plane-api-v1.yaml).
