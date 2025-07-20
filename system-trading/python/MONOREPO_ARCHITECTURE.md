# Multi-Agent System Monorepo Architecture

## Overview

The trading system follows a **monorepo pattern** where all agents are developed, tested, and deployed from a single repository while running as independent processes. This approach provides the benefits of both unified development and process isolation.

## Monorepo Benefits for Multi-Agent Systems

### 🏗️ **Unified Development**
- Shared code, utilities, and interfaces
- Consistent tooling and development workflow
- Atomic changes across multiple agents
- Simplified dependency management

### 🔄 **Independent Processes**
- Each agent runs in its own process space
- Independent scaling and resource allocation
- Fault isolation between agents
- Independent deployment lifecycle

### 📦 **Shared Infrastructure**
- Common base classes and utilities
- Shared message formats and protocols
- Unified configuration management
- Common testing and CI/CD pipelines

## Repository Structure

```
system-trading/
├── go/                           # Go-based infrastructure agents
│   ├── cmd/
│   ├── internal/
│   └── ...
├── python/                       # Python-based analysis agents (MONOREPO)
│   ├── agents/                   # All agent implementations
│   │   ├── __init__.py
│   │   ├── base.py               # Shared base agent class
│   │   ├── technical_analysis/   # Technical Analysis Agent
│   │   │   ├── __init__.py
│   │   │   ├── agent.py          # Main agent implementation
│   │   │   ├── indicators.py     # Technical indicator calculations
│   │   │   ├── config.py         # Agent-specific configuration
│   │   │   └── runner.py         # Process runner
│   │   ├── news_analysis/        # News Analysis Agent
│   │   │   ├── __init__.py
│   │   │   ├── agent.py
│   │   │   ├── nlp.py
│   │   │   ├── config.py
│   │   │   └── runner.py
│   │   ├── sentiment_analysis/   # Sentiment Analysis Agent
│   │   │   └── ...
│   │   ├── macro_economic/       # Macro Economic Agent
│   │   │   └── ...
│   │   ├── strategy/             # Strategy Agent
│   │   │   └── ...
│   │   └── backtest/             # Backtest Agent
│   │       └── ...
│   ├── shared/                   # Shared utilities and libraries
│   │   ├── __init__.py
│   │   ├── message_bus.py        # NATS integration
│   │   ├── models.py             # Shared data models
│   │   ├── utils.py              # Common utilities
│   │   ├── logging.py            # Shared logging configuration
│   │   └── testing.py            # Test utilities
│   ├── scripts/                  # Management scripts
│   │   ├── start_agent.py        # Universal agent starter
│   │   ├── stop_agents.py        # Stop all or specific agents
│   │   ├── health_check.py       # Health monitoring
│   │   └── deploy.py             # Deployment automation
│   ├── tests/                    # Comprehensive test suite
│   │   ├── unit/                 # Unit tests for each agent
│   │   ├── integration/          # Integration tests
│   │   ├── e2e/                  # End-to-end tests
│   │   └── fixtures/             # Test data and fixtures
│   ├── configs/                  # Configuration management
│   │   ├── base.yaml             # Base configuration
│   │   ├── development.yaml      # Development overrides
│   │   ├── production.yaml       # Production overrides
│   │   └── agents/               # Agent-specific configs
│   │       ├── technical_analysis.yaml
│   │       ├── news_analysis.yaml
│   │       └── ...
│   ├── deployment/               # Deployment configurations
│   │   ├── docker/               # Docker configurations
│   │   │   ├── Dockerfile.technical
│   │   │   ├── Dockerfile.news
│   │   │   └── docker-compose.yml
│   │   ├── k8s/                  # Kubernetes manifests
│   │   │   ├── namespace.yaml
│   │   │   ├── technical-analysis.yaml
│   │   │   └── ...
│   │   ├── systemd/              # SystemD services
│   │   └── supervisor/           # Supervisor configurations
│   ├── docs/                     # Documentation
│   │   ├── agents/               # Per-agent documentation
│   │   ├── development.md        # Development guide
│   │   └── deployment.md         # Deployment guide
│   ├── pyproject.toml            # Python project configuration
│   ├── requirements.txt          # Dependencies
│   ├── Makefile                  # Development automation
│   └── README.md                 # Main documentation
```

## Agent Implementation Pattern

### 1. Shared Base Infrastructure

**`agents/base.py`** - Common functionality for all agents:
```python
class BaseAgent(ABC):
    """Base class for all analysis agents"""
    
    def __init__(self, config: AgentConfig):
        self.config = config
        self.logger = setup_logging(config)
        self.message_bus = MessageBus(config.nats_url)
        self.running = False
    
    @abstractmethod
    async def process_message(self, topic: str, data: dict) -> None:
        """Agent-specific message processing"""
        pass
    
    async def start(self) -> None:
        """Common startup logic"""
        await self.message_bus.connect()
        await self.setup_subscriptions()
        self.running = True
    
    async def stop(self) -> None:
        """Common shutdown logic"""
        self.running = False
        await self.message_bus.disconnect()
```

### 2. Agent-Specific Implementation

**`agents/technical_analysis/agent.py`**:
```python
from ..base import BaseAgent
from .config import TechnicalAnalysisConfig
from .indicators import IndicatorCalculator

class TechnicalAnalysisAgent(BaseAgent):
    """Technical Analysis Agent implementation"""
    
    def __init__(self, config: TechnicalAnalysisConfig):
        super().__init__(config)
        self.calculator = IndicatorCalculator(config)
        self.data_store = MarketDataStore(config.window_size)
    
    async def setup_subscriptions(self):
        await self.message_bus.subscribe(
            "raw.market_data.price", 
            self.process_market_data
        )
    
    async def process_market_data(self, data: dict):
        # Agent-specific processing logic
        indicators = self.calculator.calculate(data)
        await self.message_bus.publish("insight.technical", indicators)
```

### 3. Process Runner Pattern

**`agents/technical_analysis/runner.py`**:
```python
#!/usr/bin/env python3
"""
Technical Analysis Agent Process Runner
"""

import asyncio
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parents[2]))

from agents.technical_analysis.agent import TechnicalAnalysisAgent
from agents.technical_analysis.config import load_config
from shared.process_manager import ProcessManager

async def main():
    """Run Technical Analysis Agent as independent process"""
    config = load_config()
    agent = TechnicalAnalysisAgent(config)
    
    process_manager = ProcessManager(
        agent_name="technical-analysis",
        agent_instance=agent
    )
    
    await process_manager.run()

if __name__ == "__main__":
    asyncio.run(main())
```

## Universal Agent Management

### 1. Universal Agent Starter

**`scripts/start_agent.py`**:
```python
#!/usr/bin/env python3
"""
Universal Agent Starter

Starts any agent as an independent process with proper configuration.
"""

import argparse
import asyncio
import importlib
import sys
from pathlib import Path

AVAILABLE_AGENTS = {
    'technical': 'agents.technical_analysis.runner',
    'news': 'agents.news_analysis.runner', 
    'sentiment': 'agents.sentiment_analysis.runner',
    'macro': 'agents.macro_economic.runner',
    'strategy': 'agents.strategy.runner',
    'backtest': 'agents.backtest.runner'
}

async def start_agent(agent_type: str, **kwargs):
    """Start specified agent"""
    if agent_type not in AVAILABLE_AGENTS:
        raise ValueError(f"Unknown agent type: {agent_type}")
    
    module_path = AVAILABLE_AGENTS[agent_type]
    module = importlib.import_module(module_path)
    
    # Run the agent's main function
    await module.main()

def main():
    parser = argparse.ArgumentParser(description="Start trading system agent")
    parser.add_argument("agent", choices=AVAILABLE_AGENTS.keys(), 
                       help="Agent type to start")
    parser.add_argument("--config", help="Configuration file path")
    parser.add_argument("--name", help="Agent instance name")
    parser.add_argument("--log-level", choices=['DEBUG', 'INFO', 'WARN', 'ERROR'])
    
    args = parser.parse_args()
    
    # Set environment variables based on arguments
    if args.name:
        os.environ[f"{args.agent.upper()}_AGENT_NAME"] = args.name
    if args.log_level:
        os.environ["LOG_LEVEL"] = args.log_level
    
    # Start the specified agent
    asyncio.run(start_agent(args.agent))

if __name__ == "__main__":
    main()
```

### 2. Multi-Agent Process Manager

**`scripts/process_manager.py`**:
```python
#!/usr/bin/env python3
"""
Multi-Agent Process Manager

Manages multiple agent processes from a single interface.
"""

import asyncio
import subprocess
import yaml
from pathlib import Path
from typing import Dict, List

class MultiAgentManager:
    """Manages multiple agent processes"""
    
    def __init__(self, config_path: str):
        self.config = self.load_config(config_path)
        self.processes: Dict[str, subprocess.Popen] = {}
    
    def load_config(self, config_path: str) -> dict:
        """Load multi-agent configuration"""
        with open(config_path) as f:
            return yaml.safe_load(f)
    
    async def start_all_agents(self):
        """Start all configured agents"""
        for agent_config in self.config['agents']:
            await self.start_agent(agent_config)
    
    async def start_agent(self, agent_config: dict):
        """Start a single agent process"""
        agent_type = agent_config['type']
        agent_name = agent_config['name']
        
        cmd = [
            sys.executable, 
            "scripts/start_agent.py",
            agent_type,
            "--name", agent_name
        ]
        
        # Add environment variables
        env = os.environ.copy()
        env.update(agent_config.get('environment', {}))
        
        process = subprocess.Popen(cmd, env=env)
        self.processes[agent_name] = process
        
        print(f"✅ Started {agent_type} agent: {agent_name} (PID: {process.pid})")
    
    async def stop_all_agents(self):
        """Stop all running agents"""
        for name, process in self.processes.items():
            process.terminate()
            print(f"🛑 Stopped agent: {name}")
        
        # Wait for graceful shutdown
        await asyncio.sleep(5)
        
        # Force kill if necessary
        for name, process in self.processes.items():
            if process.poll() is None:
                process.kill()
                print(f"⚡ Force killed agent: {name}")
    
    def get_status(self) -> Dict[str, str]:
        """Get status of all agents"""
        status = {}
        for name, process in self.processes.items():
            if process.poll() is None:
                status[name] = "running"
            else:
                status[name] = f"stopped (exit code: {process.poll()})"
        return status

# Example configuration file
EXAMPLE_CONFIG = """
agents:
  - type: technical
    name: technical-analysis-1
    environment:
      LOG_LEVEL: INFO
      TECHNICAL_DATA_WINDOW_SIZE: 200
  
  - type: technical  
    name: technical-analysis-2
    environment:
      LOG_LEVEL: INFO
      TECHNICAL_DATA_WINDOW_SIZE: 100
  
  - type: news
    name: news-analysis-1
    environment:
      LOG_LEVEL: INFO
      NEWS_SOURCE: reuters
  
  - type: strategy
    name: strategy-agent-1
    environment:
      LOG_LEVEL: DEBUG
      STRATEGY_TYPE: ml_ensemble
"""
```

## Development Workflow

### 1. Single Agent Development

```bash
# Develop and test single agent
cd python/

# Run specific agent for development
python -m agents.technical_analysis.runner

# Or use universal starter
python scripts/start_agent.py technical --name dev-ta-1 --log-level DEBUG

# Run tests for specific agent
pytest tests/unit/test_technical_analysis.py

# Run agent-specific integration tests
pytest tests/integration/test_technical_analysis_integration.py
```

### 2. Multi-Agent Development

```bash
# Start multiple agents for integration testing
python scripts/process_manager.py --config configs/development.yaml

# Run full integration test suite
pytest tests/integration/

# Run end-to-end tests with all agents
pytest tests/e2e/
```

### 3. Shared Code Development

```bash
# Test shared utilities
pytest tests/unit/test_shared/

# Update shared models
# - Edit shared/models.py
# - Run tests across all agents: make test-all
# - Verify no breaking changes
```

## Configuration Management

### 1. Hierarchical Configuration

**`configs/base.yaml`**:
```yaml
# Base configuration for all agents
message_bus:
  nats_url: "nats://localhost:4222"
  reconnect_attempts: 5

logging:
  level: INFO
  format: json

health:
  check_interval: 30
  timeout: 10
```

**`configs/agents/technical_analysis.yaml`**:
```yaml
# Technical Analysis specific configuration
extends: ../base.yaml

agent:
  name: technical-analysis-agent
  data_window_size: 200
  min_bars_required: 50
  
indicators:
  rsi_period: 14
  macd_fast: 12
  macd_slow: 26
  bb_period: 20
```

### 2. Environment-Specific Overrides

**`configs/production.yaml`**:
```yaml
# Production overrides
message_bus:
  nats_url: "nats://nats-cluster:4222"

logging:
  level: WARN

monitoring:
  enabled: true
  prometheus_url: "http://prometheus:9090"
```

## Deployment Strategies

### 1. Single Repository Deployment

```bash
# Build all agents from monorepo
docker build -t trading-system/agents .

# Deploy with different entry points
docker run trading-system/agents python scripts/start_agent.py technical
docker run trading-system/agents python scripts/start_agent.py news
docker run trading-system/agents python scripts/start_agent.py strategy
```

### 2. Kubernetes Deployment

**`deployment/k8s/technical-analysis.yaml`**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: technical-analysis-agent
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: technical-analysis
        image: trading-system/agents:latest
        command: ["python", "scripts/start_agent.py"]
        args: ["technical", "--name", "$(POD_NAME)"]
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
```

### 3. Multi-Agent Orchestration

**`deployment/docker/docker-compose.yml`**:
```yaml
version: '3.8'
services:
  technical-analysis-1:
    build: ../../
    command: ["python", "scripts/start_agent.py", "technical"]
    environment:
      - TECHNICAL_AGENT_NAME=technical-analysis-1
  
  technical-analysis-2:
    build: ../../  
    command: ["python", "scripts/start_agent.py", "technical"]
    environment:
      - TECHNICAL_AGENT_NAME=technical-analysis-2
  
  news-analysis:
    build: ../../
    command: ["python", "scripts/start_agent.py", "news"]
    environment:
      - NEWS_AGENT_NAME=news-analysis-1
  
  strategy-agent:
    build: ../../
    command: ["python", "scripts/start_agent.py", "strategy"]
    environment:
      - STRATEGY_AGENT_NAME=strategy-agent-1
```

## Testing Strategy

### 1. Unit Tests (Per Agent)

```bash
# Test individual agents
pytest tests/unit/test_technical_analysis.py
pytest tests/unit/test_news_analysis.py

# Test shared components
pytest tests/unit/test_shared/
```

### 2. Integration Tests (Cross-Agent)

```bash
# Test agent interactions
pytest tests/integration/test_technical_to_strategy.py
pytest tests/integration/test_news_to_strategy.py

# Test full pipeline
pytest tests/integration/test_data_to_execution_pipeline.py
```

### 3. End-to-End Tests

```bash
# Test complete system
pytest tests/e2e/test_complete_trading_workflow.py

# Test scaling scenarios  
pytest tests/e2e/test_multi_agent_scaling.py
```

## Benefits of Monorepo Multi-Agent Architecture

### ✅ **Development Benefits**
- **Unified Tooling**: Single set of development tools and scripts
- **Shared Code**: Common utilities, models, and interfaces
- **Atomic Changes**: Update multiple agents in single commit
- **Consistent Testing**: Uniform test strategies across agents

### ✅ **Operational Benefits**
- **Process Isolation**: Independent failure domains
- **Independent Scaling**: Scale agents based on load
- **Resource Optimization**: Optimize resources per agent type
- **Deployment Flexibility**: Deploy agents independently

### ✅ **Maintenance Benefits**
- **Single Source of Truth**: All code in one repository
- **Simplified Dependencies**: Unified dependency management
- **Cross-Agent Refactoring**: Easy to update shared interfaces
- **Comprehensive Testing**: Full system testing capabilities

This monorepo approach provides the perfect balance between unified development and independent process execution, enabling efficient development while maintaining the operational benefits of microservices architecture.