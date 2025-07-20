# Trading System Kubernetes Deployment

Complete Kubernetes manifests for deploying the Multi-Agent Trading System with Central Router A2A Communication.

## 🏗️ Architecture Overview

The trading system consists of the following components:

### Infrastructure Layer
- **NATS**: Message bus for A2A communication and event streaming
- **PostgreSQL**: Primary database for trading data and state
- **Redis**: Cache for fast data access and session storage
- **Prometheus**: Metrics collection and monitoring
- **Grafana**: Dashboards and visualization

### Application Layer
- **Trading Core** (Go): Core trading engine with execution, risk, and portfolio management
- **Central Router** (Python): Intelligent A2A communication routing and service discovery
- **Technical Analysis Agent** (Python): RSI, MACD, Bollinger Bands analysis
- **News Analysis Agent** (Python): AI-powered news impact analysis
- **Sentiment Analysis Agent** (Python): Market sentiment analysis

## 📁 Directory Structure

```
k8s/
├── namespace/                 # Namespace definition
├── secrets/                   # Sensitive configuration (passwords, tokens)
├── configmaps/               # Application configuration
├── storage/                  # Persistent volume claims
├── infrastructure/           # Infrastructure components
│   ├── nats/                # Message bus
│   ├── postgresql/          # Database
│   ├── redis/               # Cache
│   ├── prometheus/          # Monitoring
│   └── grafana/             # Dashboards
├── applications/             # Application deployments
│   ├── trading-core/        # Go trading engine
│   └── python-agents/       # Python analysis agents
├── ingress/                  # External access configuration
├── kustomization.yaml        # Kustomize configuration
├── deploy.sh                # Deployment script
└── README.md                # This file
```

## 🚀 Quick Start

### Prerequisites

1. **Kubernetes Cluster**: Version 1.20+
2. **kubectl**: Latest version
3. **kustomize**: Optional but recommended
4. **Storage Class**: Ensure you have a storage class named `fast-ssd` or update the PVCs

### One-Command Deployment

```bash
# Deploy the complete system
./k8s/deploy.sh

# Or using specific commands
./k8s/deploy.sh deploy
```

### Step-by-Step Deployment

```bash
# 1. Deploy infrastructure only
./k8s/deploy.sh infrastructure

# 2. Deploy applications
./k8s/deploy.sh applications

# 3. Deploy networking
./k8s/deploy.sh networking

# 4. Check status
./k8s/deploy.sh status

# 5. Get access information
./k8s/deploy.sh access
```

## 🔧 Configuration

### Before Deployment

1. **Update Image Names**: Edit `kustomization.yaml` to use your container registry:
   ```yaml
   images:
     - name: trading-core
       newName: your-registry/trading-core
       newTag: v1.0.0
     - name: python-agents
       newName: your-registry/python-agents
       newTag: v1.0.0
   ```

2. **Storage Class**: Update storage class in `storage/persistent-volumes.yaml`:
   ```yaml
   storageClassName: your-storage-class  # Change from fast-ssd
   ```

3. **Domain Names**: Update ingress hosts in `ingress/trading-system-ingress.yaml`:
   ```yaml
   rules:
     - host: trading.yourdomain.com  # Update to your domain
   ```

4. **Secrets**: Update base64 encoded secrets in `secrets/trading-secrets.yaml`:
   ```bash
   echo -n "your-password" | base64
   ```

### Environment Variables

Key configuration can be modified in the ConfigMaps:

- `configmaps/trading-core-config.yaml`: Trading core settings
- `configmaps/python-agents-config.yaml`: Agent-specific settings
- `configmaps/prometheus-config.yaml`: Monitoring configuration

## 📊 Monitoring and Observability

### Prometheus Metrics

The system exposes metrics on the following endpoints:
- Trading Core: `:8080/metrics`
- Python Agents: `:8080/metrics`
- NATS: `:8222/varz`
- Redis: `:9121/metrics`
- PostgreSQL: `:9187/metrics`

### Grafana Dashboards

Access Grafana at:
- NodePort: `http://NODE-IP:30030`
- Ingress: `https://grafana.yourdomain.com`

Default login: `admin/admin`

### Health Checks

All components include health checks:
```bash
# Check pod health
kubectl get pods -n trading-system

# Check specific component
kubectl describe pod -n trading-system POD-NAME
```

## 🔍 Accessing the System

### NodePort Access (Default)

```bash
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')

# Trading Core API
curl http://$NODE_IP:30080/health

# Grafana Dashboard
open http://$NODE_IP:30030

# Prometheus
open http://$NODE_IP:30090
```

### Port Forwarding

```bash
# Trading Core
kubectl port-forward -n trading-system svc/trading-core-service 8080:8080

# Grafana
kubectl port-forward -n trading-system svc/grafana-service 3000:3000

# Prometheus
kubectl port-forward -n trading-system svc/prometheus-service 9090:9090
```

### Ingress Access

Configure DNS to point to your ingress controller:
- `trading.yourdomain.com` → Trading Core API
- `grafana.yourdomain.com` → Grafana Dashboard
- `prometheus.yourdomain.com` → Prometheus

## 🔄 Auto-Scaling

The system includes Horizontal Pod Autoscalers (HPA):

```bash
# Check HPA status
kubectl get hpa -n trading-system

# Manual scaling
kubectl scale deployment trading-core -n trading-system --replicas=3
```

### Scaling Configuration

- **Technical Analysis**: 2-10 replicas based on CPU (70%) and Memory (80%)
- **News Analysis**: 1-5 replicas based on CPU (70%)
- **Sentiment Analysis**: 1-5 replicas based on CPU (70%)
- **Trading Core**: Manual scaling (default 2 replicas)

## 🛠️ Troubleshooting

### Common Issues

1. **Pods stuck in Pending**:
   ```bash
   kubectl describe pod POD-NAME -n trading-system
   # Usually storage class or resource constraints
   ```

2. **Database connection issues**:
   ```bash
   kubectl logs -n trading-system postgresql-0
   kubectl exec -it -n trading-system postgresql-0 -- psql -U postgres -d trading_system
   ```

3. **NATS connectivity**:
   ```bash
   kubectl port-forward -n trading-system svc/nats-service 4222:4222
   # Test with NATS CLI tools
   ```

### Useful Commands

```bash
# View all resources
kubectl get all -n trading-system

# Check logs
kubectl logs -f -n trading-system deployment/central-router

# Execute into containers
kubectl exec -it -n trading-system deployment/trading-core -- /bin/sh

# Delete and redeploy
./k8s/deploy.sh delete
./k8s/deploy.sh deploy
```

## 🔐 Security

### Network Policies

The deployment includes network policies that:
- Restrict inter-pod communication to same namespace
- Allow ingress from ingress controller
- Allow monitoring access from Prometheus

### Secrets Management

Sensitive data is stored in Kubernetes secrets:
- Database passwords
- JWT tokens
- API keys

Consider using external secret management like:
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault

## 📈 Performance Tuning

### Resource Requests and Limits

Current resource allocation:

| Component | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| Trading Core | 200m | 256Mi | 1000m | 1Gi |
| Central Router | 100m | 128Mi | 500m | 512Mi |
| Technical Analysis | 200m | 256Mi | 1000m | 1Gi |
| PostgreSQL | 200m | 256Mi | 1000m | 1Gi |
| Prometheus | 200m | 512Mi | 1000m | 2Gi |

### Storage Performance

- Use SSD-backed storage classes for databases
- Consider separate storage classes for different workloads
- Monitor disk I/O with Prometheus metrics

## 🔄 Updates and Rollbacks

### Rolling Updates

```bash
# Update image
kubectl set image deployment/trading-core trading-core=your-registry/trading-core:v1.1.0 -n trading-system

# Check rollout status
kubectl rollout status deployment/trading-core -n trading-system

# Rollback if needed
kubectl rollout undo deployment/trading-core -n trading-system
```

### Blue-Green Deployments

For zero-downtime deployments, consider using:
- Argo CD
- Flagger
- Istio with traffic splitting

## 📋 Maintenance

### Backup Strategy

1. **Database Backups**:
   ```bash
   kubectl exec -n trading-system postgresql-0 -- pg_dump -U postgres trading_system > backup.sql
   ```

2. **Configuration Backups**:
   ```bash
   kubectl get all,configmap,secret -n trading-system -o yaml > trading-system-backup.yaml
   ```

### Log Management

Logs are available through:
```bash
# Recent logs
kubectl logs -n trading-system --tail=100 deployment/trading-core

# Follow logs
kubectl logs -n trading-system -f deployment/central-router

# All container logs
kubectl logs -n trading-system --all-containers=true deployment/technical-analysis
```

## 🎯 Next Steps

1. **Set up CI/CD pipelines** for automated deployments
2. **Configure alerting** with AlertManager
3. **Implement backup automation**
4. **Set up log aggregation** with ELK stack or similar
5. **Add service mesh** (Istio/Linkerd) for advanced traffic management
6. **Implement GitOps** with ArgoCD or Flux

## 📞 Support

For issues and questions:
- Check the troubleshooting section above
- Review pod logs and events
- Consult the main project documentation
- Create issues in the project repository

---

**Note**: This deployment is production-ready but should be customized based on your specific infrastructure, security requirements, and operational procedures. 