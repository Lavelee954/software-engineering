# Process-Based Agent Architecture

## Overview

The Python analysis agents are designed to run as **independent processes**, following microservices best practices for isolation, scalability, and fault tolerance. Each agent runs in its own process space with dedicated resources and lifecycle management.

## Benefits of Process-Based Architecture

### 🔒 **Process Isolation**
- Memory isolation prevents agents from affecting each other
- CPU resource allocation per agent
- Independent failure domains
- Security boundaries between agents

### 📈 **Horizontal Scalability**
- Run multiple instances of the same agent type
- Load distribution across agent instances
- Independent scaling per agent type
- Resource optimization based on workload

### 🛡️ **Fault Tolerance**
- Agent failure doesn't affect other agents
- Independent restart and recovery
- Circuit breaker patterns per agent
- Graceful degradation

### 🔧 **Operational Excellence**
- Independent deployment and updates
- Per-agent monitoring and logging
- Granular resource management
- Process-level health checks

## Architecture Components

### 1. Standalone Process Wrapper

Each agent has a dedicated process wrapper that handles:

```python
class TechnicalAnalysisProcess:
    - Signal handling (SIGTERM, SIGINT)
    - Environment configuration
    - Graceful startup and shutdown
    - Error handling and recovery
    - Resource management
```

### 2. Process Management Scripts

**Start Script** (`scripts/start_agent.sh`):
- Environment validation
- Dependency checks (NATS connectivity)
- Process startup with proper configuration
- Background execution support

**Stop Script** (`scripts/stop_agent.sh`):
- Graceful shutdown with SIGTERM
- Timeout-based force kill fallback
- Process status monitoring
- Bulk operations for multiple instances

### 3. Service Management Integration

**SystemD Service** (`systemd/technical-analysis.service`):
```ini
[Unit]
Description=Technical Analysis Agent
After=network.target nats.service

[Service]
Type=simple
ExecStart=/opt/trading-system/python/venv/bin/python run_technical_analysis.py
Restart=always
RestartSec=10
```

**Supervisor Configuration** (`supervisor/technical-analysis.conf`):
```ini
[program:technical-analysis-agent]
command=/opt/trading-system/python/venv/bin/python run_technical_analysis.py
autostart=true
autorestart=true
```

### 4. Container Orchestration

**Docker Compose**:
```yaml
services:
  technical-analysis:
    command: ["python", "run_technical_analysis.py"]
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
```

**Kubernetes Deployment**:
```yaml
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: technical-analysis
        command: ["python", "run_technical_analysis.py"]
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Process Deployment Patterns

### 1. Single Instance Deployment

**Local Development**:
```bash
# Direct execution
python run_technical_analysis.py

# Script-based execution
./scripts/start_agent.sh --name ta-agent-1 --log-level DEBUG
```

**Production with SystemD**:
```bash
sudo systemctl start technical-analysis
sudo systemctl enable technical-analysis
```

### 2. Multi-Instance Deployment

**Multiple Agents (Load Distribution)**:
```bash
# Start 3 instances for load balancing
TECHNICAL_AGENT_NAME=ta-agent-1 ./scripts/start_agent.sh &
TECHNICAL_AGENT_NAME=ta-agent-2 ./scripts/start_agent.sh &
TECHNICAL_AGENT_NAME=ta-agent-3 ./scripts/start_agent.sh &
```

**Docker Compose Scaling**:
```bash
# Scale to 3 instances
docker-compose up --scale technical-analysis=3

# With resource profiles
docker-compose --profile scaling up
```

**Kubernetes Horizontal Scaling**:
```bash
# Scale deployment
kubectl scale deployment technical-analysis-agent --replicas=5

# Auto-scaling with HPA
kubectl apply -f k8s/technical-analysis-deployment.yaml
```

### 3. Mixed Deployment (Development + Production)

**Development Environment**:
```bash
# Local development with debugging
LOG_LEVEL=DEBUG python run_technical_analysis.py

# Background development
make run-technical-background
```

**Production Environment**:
```bash
# SystemD managed
sudo systemctl start technical-analysis

# Supervisor managed
supervisorctl start technical-analysis-agent

# Kubernetes managed
kubectl apply -f k8s/technical-analysis-deployment.yaml
```

## Configuration Management

### Environment-Based Configuration

Each process loads configuration from environment variables:

```bash
# Core configuration
export NATS_URL="nats://localhost:4222"
export LOG_LEVEL="INFO"
export TECHNICAL_AGENT_NAME="ta-agent-1"

# Technical analysis specific
export TECHNICAL_DATA_WINDOW_SIZE="200"
export TECHNICAL_MIN_BARS_REQUIRED="50"
export TECHNICAL_RSI_PERIOD="14"
```

### Configuration Hierarchy

1. **Environment Variables** (highest priority)
2. **.env File** (development)
3. **Default Values** (fallback)

### Multi-Instance Configuration

```bash
# Instance 1 - High frequency trading
export TECHNICAL_AGENT_NAME="ta-hft-1"
export TECHNICAL_PUBLISH_FREQUENCY="1"
export TECHNICAL_MIN_BARS_REQUIRED="20"

# Instance 2 - Long-term analysis
export TECHNICAL_AGENT_NAME="ta-longterm-1"
export TECHNICAL_PUBLISH_FREQUENCY="5"
export TECHNICAL_MIN_BARS_REQUIRED="100"
```

## Process Monitoring and Health

### Process Health Checks

**Script-Based Monitoring**:
```bash
# Check process status
./scripts/stop_agent.sh --status

# Health check via Make
make status-technical
```

**SystemD Status**:
```bash
systemctl status technical-analysis
journalctl -u technical-analysis -f
```

**Container Health Checks**:
```yaml
healthcheck:
  test: ["CMD", "python", "-c", "import sys; sys.exit(0)"]
  interval: 30s
  timeout: 10s
  retries: 3
```

### Process Metrics

Each process provides:
- **CPU Usage**: Process-level CPU consumption
- **Memory Usage**: RSS, VSS memory metrics
- **Message Processing**: Throughput and latency
- **Error Rates**: Process-level error tracking

### Log Management

**Per-Process Logging**:
```bash
# Individual log files
logs/technical-analysis-agent-1.log
logs/technical-analysis-agent-2.log
logs/technical-analysis-agent-3.log

# Centralized with identifiers
{
  "timestamp": "2023-12-07T10:30:00Z",
  "level": "INFO",
  "agent": "technical-analysis-agent-1",
  "message": "Processing market data",
  "pid": 12345
}
```

## Resource Management

### Memory Management

**Per-Process Memory Allocation**:
```yaml
resources:
  requests:
    memory: "256Mi"
  limits:
    memory: "512Mi"
```

**Data Window Management**:
- Configurable window sizes per process
- Automatic memory cleanup
- Process-level memory monitoring

### CPU Management

**CPU Allocation**:
```yaml
resources:
  requests:
    cpu: "200m"    # 0.2 CPU cores
  limits:
    cpu: "500m"    # 0.5 CPU cores
```

**CPU Scaling**:
- Vertical scaling: Increase CPU per process
- Horizontal scaling: Add more processes

## Failure Handling and Recovery

### Process Restart Strategies

**SystemD Restart**:
```ini
Restart=always
RestartSec=10
StartLimitBurst=5
StartLimitIntervalSec=300
```

**Supervisor Restart**:
```ini
autorestart=true
startretries=3
exitcodes=0,2
```

**Kubernetes Restart**:
```yaml
restartPolicy: Always
terminationGracePeriodSeconds: 30
```

### Graceful Shutdown

**Signal Handling**:
```python
def _signal_handler(self, signum, frame):
    """Handle shutdown signals"""
    self.logger.info(f"Received signal {signum}")
    self.running = False
    asyncio.create_task(self.agent.stop())
```

**Cleanup Process**:
1. Stop processing new messages
2. Complete current operations
3. Close NATS connections
4. Flush logs and metrics
5. Exit with appropriate code

## Performance Characteristics

### Single Process Performance

- **Startup Time**: ~2-5 seconds
- **Memory Footprint**: ~50MB base + ~1MB per symbol
- **CPU Usage**: ~5-10% per 1000 msg/sec
- **Message Latency**: ~10-50ms per message

### Multi-Process Performance

- **Load Distribution**: Linear scaling up to CPU cores
- **Memory Isolation**: No shared memory conflicts
- **Fault Isolation**: Independent failure domains
- **Resource Efficiency**: Optimal resource utilization

## Production Deployment Examples

### 1. High-Availability Setup

**3-Node Kubernetes Cluster**:
```yaml
spec:
  replicas: 6  # 2 per node
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
            - key: app
              operator: In
              values:
              - technical-analysis-agent
          topologyKey: kubernetes.io/hostname
```

### 2. Load-Based Scaling

**Auto-scaling Configuration**:
```yaml
minReplicas: 2
maxReplicas: 10
metrics:
- type: Resource
  resource:
    name: cpu
    target:
      type: Utilization
      averageUtilization: 70
```

### 3. Multi-Environment Deployment

**Development**:
```bash
# Single instance for development
python run_technical_analysis.py
```

**Staging**:
```bash
# 2 instances for testing
docker-compose up --scale technical-analysis=2
```

**Production**:
```bash
# Kubernetes with auto-scaling
kubectl apply -f k8s/technical-analysis-deployment.yaml
```

## Best Practices

### ✅ **Do This**

1. **Use environment variables** for configuration
2. **Implement graceful shutdown** with signal handling
3. **Monitor process health** with appropriate tools
4. **Scale horizontally** rather than vertically when possible
5. **Use resource limits** to prevent resource starvation
6. **Implement proper logging** with process identifiers
7. **Test scaling scenarios** before production deployment

### ❌ **Avoid This**

1. **Shared state between processes** (use message bus instead)
2. **Hardcoded configuration** (use environment variables)
3. **Ignoring resource limits** (can cause OOM kills)
4. **No health checks** (prevents proper orchestration)
5. **Blocking shutdown** (implement proper signal handling)
6. **Single points of failure** (always run multiple instances)

## Migration from Monolithic to Process-Based

### Step 1: Extract Process Wrapper
```python
# Before: Single application
agents = [TechnicalAnalysisAgent(), NewsAnalysisAgent()]
for agent in agents:
    await agent.start()

# After: Individual processes
# run_technical_analysis.py
agent = TechnicalAnalysisAgent(config)
await agent.run()
```

### Step 2: Update Deployment
```bash
# Before: Single container
docker run trading-system/agents

# After: Individual containers
docker run trading-system/technical-analysis
docker run trading-system/news-analysis
```

### Step 3: Scale Independently
```bash
# Scale only technical analysis
kubectl scale deployment technical-analysis-agent --replicas=5

# Keep news analysis at baseline
kubectl scale deployment news-analysis-agent --replicas=2
```

The process-based architecture provides a robust, scalable, and maintainable foundation for the Python analysis agents, enabling independent development, deployment, and scaling while maintaining the event-driven architecture principles from CLAUDE.md.