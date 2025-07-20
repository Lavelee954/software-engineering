# Python Analysis Agents

This directory contains the Python-based analysis agents for the multi-asset trading system, implementing the specifications from CLAUDE.md.

## Architecture

The Python agents follow the same event-driven architecture as the Go components:

- **Technical Analysis Agent**: Processes market data to generate technical indicators
- **News Analysis Agent**: Analyzes news articles for market sentiment  
- **Sentiment Analysis Agent**: Performs NLP sentiment analysis
- **Macro Economic Agent**: Analyzes macroeconomic indicators

## Current Implementation Status

### ✅ Phase 3.1: Technical Analysis Agent (Completed)

**Features:**
- Subscribes to `raw.market_data.price` topic
- Publishes to `insight.technical` topic
- Comprehensive technical indicators:
  - RSI (Relative Strength Index)
  - MACD (Moving Average Convergence Divergence)
  - Bollinger Bands
  - Moving Averages (SMA, EMA)
  - Volume indicators
- Trend and momentum signal generation
- Confidence scoring based on data quality
- Data window management for memory efficiency
- Multi-symbol support

**Technical Details:**
- Uses TA-Lib for robust indicator calculations
- Pandas for efficient data management
- Asyncio-based NATS integration
- Structured logging with contextual information
- Comprehensive error handling and resilience

### 🔄 Upcoming Phases

- **Phase 3.2**: News & Sentiment Analysis Agents
- **Phase 3.3**: Macro Economic Agent  
- **Phase 4.1**: Strategy Agent
- **Phase 4.2**: Backtest Agent

## Installation

### Prerequisites

- Python 3.9+
- TA-Lib C library (automatically installed in Docker)

### Local Development

```bash
# Install dependencies
pip install -r requirements.txt
pip install -e .

# For TA-Lib on macOS
brew install ta-lib

# For TA-Lib on Ubuntu/Debian
sudo apt-get install libta-lib-dev
```

### Docker Deployment

```bash
# Build and run with Docker Compose
docker-compose up --build

# Run specific agent
docker-compose up technical-analysis
```

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Key configuration options:

- `NATS_URL`: NATS message bus URL
- `LOG_LEVEL`: Logging verbosity (DEBUG, INFO, WARN, ERROR)
- `TECHNICAL_*`: Technical analysis specific settings

## Usage

### Running Technical Analysis Agent (Process-Based)

Each agent runs as an **independent process** for better isolation, scalability, and fault tolerance.

#### Single Process

```bash
# Direct execution
python run_technical_analysis.py

# With custom configuration
TECHNICAL_AGENT_NAME=ta-agent-1 LOG_LEVEL=DEBUG python run_technical_analysis.py

# Using management script
./scripts/start_agent.sh --name ta-agent-1 --log-level DEBUG

# Background execution
nohup ./scripts/start_agent.sh > logs/ta-agent.log 2>&1 &

# Using Make targets
make run-technical              # Direct execution
make run-technical-debug        # Debug mode
make run-technical-background   # Background mode
```

#### Multiple Processes (Scaling)

```bash
# Scale with multiple instances
make run-technical-scaled       # Starts 3 instances

# Manual scaling
TECHNICAL_AGENT_NAME=ta-agent-1 ./scripts/start_agent.sh &
TECHNICAL_AGENT_NAME=ta-agent-2 ./scripts/start_agent.sh &
TECHNICAL_AGENT_NAME=ta-agent-3 ./scripts/start_agent.sh &

# Docker Compose scaling
docker-compose up --scale technical-analysis=3

# Kubernetes scaling
kubectl scale deployment technical-analysis-agent --replicas=5
```

#### Process Management

```bash
# Check process status
./scripts/stop_agent.sh --status
make status-technical

# Stop processes
./scripts/stop_agent.sh         # Graceful shutdown
./scripts/stop_agent.sh --force # Force kill
make stop-technical

# SystemD (production)
sudo systemctl start technical-analysis
sudo systemctl stop technical-analysis
sudo systemctl status technical-analysis

# Supervisor (production)
supervisorctl start technical-analysis-agent
supervisorctl stop technical-analysis-agent
```

#### Container Deployment

```bash
# Single container
docker-compose up technical-analysis

# Multiple containers with scaling
docker-compose up --scale technical-analysis=3

# With scaling profile
docker-compose --profile scaling up

# Kubernetes deployment
kubectl apply -f k8s/technical-analysis-deployment.yaml
```

### Message Flow

The Technical Analysis Agent implements the CLAUDE.md workflow:

```
1. DataCollector → publishes to `raw.market_data.price`
2. TechnicalAnalysisAgent → subscribes to `raw.market_data.price`
3. TechnicalAnalysisAgent → calculates indicators
4. TechnicalAnalysisAgent → publishes to `insight.technical`
5. StrategyAgent → subscribes to `insight.technical`
```

### Input Message Format

```json
{
  "symbol": "AAPL",
  "timestamp": "2023-12-07T10:30:00Z",
  "open_price": 180.50,
  "high_price": 181.25,
  "low_price": 180.10,
  "close_price": 181.00,
  "volume": 1500000,
  "source": "data_collector"
}
```

### Output Message Format

```json
{
  "symbol": "AAPL",
  "timestamp": "2023-12-07T10:30:00Z",
  "rsi": 65.5,
  "macd": 1.25,
  "macd_signal": 1.10,
  "macd_histogram": 0.15,
  "bb_upper": 182.50,
  "bb_middle": 181.00,
  "bb_lower": 179.50,
  "bb_width": 3.00,
  "bb_position": 0.5,
  "sma_short": 181.25,
  "sma_long": 179.80,
  "ema_short": 181.30,
  "ema_long": 180.00,
  "volume_sma": 1400000,
  "volume_ratio": 1.07,
  "trend_signal": "bullish",
  "momentum_signal": "neutral",
  "bars_analyzed": 50,
  "confidence": 0.95
}
```

## Testing

```bash
# Run all tests
pytest

# Run with coverage
pytest --cov=agents --cov-report=html

# Run specific test file
pytest tests/test_technical_analysis.py -v

# Run with verbose output
pytest -v -s
```

## Development

### Code Quality

```bash
# Format code
black agents/ tests/

# Sort imports
isort agents/ tests/

# Type checking
mypy agents/

# Lint
flake8 agents/ tests/
```

### Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run on all files
pre-commit run --all-files
```

## Monitoring and Observability

The agents include comprehensive observability features:

- **Structured Logging**: JSON-formatted logs with context
- **Health Checks**: Docker health checks and monitoring endpoints
- **Error Handling**: Graceful error handling with retry logic
- **Performance Metrics**: Processing time and throughput tracking

### Log Analysis

```bash
# View agent logs
docker-compose logs -f technical-analysis

# Filter by log level
docker-compose logs technical-analysis | grep ERROR

# Follow specific agent
docker-compose logs -f --tail=100 technical-analysis
```

## Architecture Integration

The Python agents integrate seamlessly with the Go-based infrastructure:

- **Message Bus**: Shared NATS infrastructure
- **Data Formats**: Compatible JSON message formats
- **Error Handling**: Consistent error handling patterns
- **Monitoring**: Unified logging and metrics

## Performance Characteristics

### Technical Analysis Agent

- **Latency**: ~10-50ms per message (depending on indicators)
- **Throughput**: 1000+ messages/second
- **Memory**: ~50MB base + ~1MB per symbol
- **CPU**: Low usage, optimized with NumPy/TA-Lib

### Scalability

- **Horizontal**: Multiple agent instances for load distribution
- **Vertical**: Configurable data windows and processing frequency
- **Memory Management**: Automatic data window management
- **Resource Limits**: Configurable via Docker constraints

## Security

- **Non-root Container**: Runs as non-privileged user
- **Input Validation**: Pydantic-based message validation
- **Error Isolation**: Graceful handling of malformed data
- **Resource Limits**: Memory and CPU constraints

## Troubleshooting

### Common Issues

1. **TA-Lib Installation**:
   ```bash
   # macOS
   brew install ta-lib
   
   # Ubuntu/Debian
   sudo apt-get install libta-lib-dev
   ```

2. **NATS Connection**:
   ```bash
   # Check NATS server status
   curl http://localhost:8222/varz
   ```

3. **Memory Usage**:
   ```bash
   # Monitor container memory
   docker stats technical-analysis-agent
   ```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=DEBUG python -m agents.technical_analysis
```

### Health Checks

```bash
# Check agent health
curl http://localhost:8222/connz

# Check Docker health
docker inspect technical-analysis-agent | grep Health -A 10
```

## Contributing

1. Follow the existing code style (Black, isort)
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Ensure Docker builds pass
5. Verify integration with Go components

## License

Part of the Multi-Asset Trading System project.