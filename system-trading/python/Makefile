# Python Analysis Agents Makefile

.PHONY: help install test lint format clean docker-build docker-run docker-test

# Default target
help:
	@echo "Available targets:"
	@echo "  install      - Install dependencies"
	@echo "  test         - Run tests with coverage"
	@echo "  lint         - Run linting (flake8, mypy)"
	@echo "  format       - Format code (black, isort)"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-test  - Run tests in Docker"
	@echo ""
	@echo "🚀 Central Router & Agent Management:"
	@echo "  start-router         - Start Central Router service"
	@echo "  start-technical      - Start Technical Analysis Agent"
	@echo "  start-news           - Start News Analysis Agent"
	@echo "  start-sentiment      - Start Sentiment Analysis Agent"
	@echo "  start-all-agents     - Start all analysis agents"
	@echo "  stop-agents          - Stop all agent processes"
	@echo "  status               - Show agent status"

# Development setup
install:
	pip install --upgrade pip
	pip install -r requirements.txt
	pip install -e .

# Testing
test:
	pytest tests/ -v --cov=agents --cov-report=html --cov-report=term-missing

test-fast:
	pytest tests/ -v -x

test-watch:
	pytest-watch tests/ agents/

# Code quality
lint:
	flake8 agents/ tests/
	mypy agents/

format:
	black agents/ tests/
	isort agents/ tests/

format-check:
	black --check agents/ tests/
	isort --check-only agents/ tests/

# Cleanup
clean:
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} +
	rm -rf build/
	rm -rf dist/
	rm -rf .coverage
	rm -rf htmlcov/
	rm -rf .pytest_cache/
	rm -rf .mypy_cache/

# Docker operations
docker-build:
	docker-compose build

docker-run:
	docker-compose up

docker-run-detached:
	docker-compose up -d

docker-test:
	docker-compose run --rm technical-analysis pytest tests/ -v

docker-stop:
	docker-compose down

docker-clean:
	docker-compose down -v
	docker system prune -f

# Development workflow
dev-setup: install
	cp .env.example .env
	@echo "Development setup complete. Edit .env as needed."

dev-test: format lint test
	@echo "All development checks passed!"

# CI/CD targets
ci-install:
	pip install --upgrade pip
	pip install -r requirements.txt

ci-test: format-check lint test
	@echo "CI checks complete!"

# Monorepo Agent Management (Universal Starter)
list-agents:
	python3 scripts/start_agent.py list

# Single agent targets
run-technical:
	python3 scripts/start_agent.py technical

run-technical-debug:
	python3 scripts/start_agent.py technical --log-level DEBUG

run-technical-named:
	python3 scripts/start_agent.py technical --name technical-agent-1

run-technical-background:
	nohup python3 scripts/start_agent.py technical --name technical-bg > logs/technical.log 2>&1 &

# News Analysis Agent targets
run-news:
	python3 scripts/start_agent.py news

run-news-debug:
	python3 scripts/start_agent.py news --log-level DEBUG

run-news-background:
	nohup python3 scripts/start_agent.py news --name news-bg > logs/news.log 2>&1 &

# Sentiment Analysis Agent targets  
run-sentiment:
	python3 scripts/start_agent.py sentiment

run-sentiment-debug:
	python3 scripts/start_agent.py sentiment --log-level DEBUG

run-sentiment-background:
	nohup python3 scripts/start_agent.py sentiment --name sentiment-bg > logs/sentiment.log 2>&1 &

# Development presets
run-technical-dev:
	python3 scripts/start_agent.py technical --name dev-ta --log-level DEBUG --config-preset development

run-technical-prod:
	python3 scripts/start_agent.py technical --name prod-ta --config-preset production

run-news-dev:
	python3 scripts/start_agent.py news --name dev-news --log-level DEBUG --config-preset development

run-sentiment-dev:
	python3 scripts/start_agent.py sentiment --name dev-sentiment --log-level DEBUG --config-preset development

# Multi-agent scaling
run-technical-scaled:
	python3 scripts/start_agent.py technical --name ta-1 &
	python3 scripts/start_agent.py technical --name ta-2 &
	python3 scripts/start_agent.py technical --name ta-3 &

run-analysis-cluster:
	python3 scripts/start_agent.py technical --name ta-1 &
	python3 scripts/start_agent.py news --name news-1 &
	python3 scripts/start_agent.py sentiment --name sentiment-1 &

# Universal commands for any agent type
run-agent:
	@echo "Usage: make run-agent AGENT=<type> [NAME=<name>] [LEVEL=<log-level>]"
	@echo "Example: make run-agent AGENT=technical NAME=my-agent LEVEL=DEBUG"

run-agent-impl:
	python3 scripts/start_agent.py $(AGENT) $(if $(NAME),--name $(NAME)) $(if $(LEVEL),--log-level $(LEVEL))

# Dry run testing
dry-run-technical:
	python3 scripts/start_agent.py technical --dry-run --name test-agent

dry-run-news:
	python3 scripts/start_agent.py news --dry-run --name test-news

dry-run-sentiment:
	python3 scripts/start_agent.py sentiment --dry-run --name test-sentiment

dry-run-implemented:
	@echo "Testing all implemented agent types with dry run:"
	python3 scripts/start_agent.py technical --dry-run --name test-ta
	python3 scripts/start_agent.py news --dry-run --name test-news
	python3 scripts/start_agent.py sentiment --dry-run --name test-sentiment

dry-run-all:
	@echo "Testing all agent types with dry run:"
	python3 scripts/start_agent.py technical --dry-run --name test-ta
	python3 scripts/start_agent.py news --dry-run --name test-news
	python3 scripts/start_agent.py sentiment --dry-run --name test-sentiment
	python3 scripts/start_agent.py macro --dry-run --name test-macro
	python3 scripts/start_agent.py strategy --dry-run --name test-strategy
	python3 scripts/start_agent.py backtest --dry-run --name test-backtest

# Process management (legacy)
stop-technical:
	./scripts/stop_agent.sh

status-technical:
	./scripts/stop_agent.sh --status

# Integration testing
integration-test:
	@echo "Starting integration test environment..."
	docker-compose up -d nats
	sleep 5
	python -m pytest tests/ -k integration -v
	docker-compose down

# Performance testing
benchmark:
	python -m pytest tests/ -k benchmark -v --benchmark-only

# Documentation
docs:
	@echo "Generating documentation..."
	# Add documentation generation commands here

# 🚀 Central Router & Agent Management
NATS_URL ?= nats://localhost:4222
LOG_LEVEL ?= INFO

start-router:
	@echo "🔧 Starting Central Router..."
	python run_central_router.py --nats-url $(NATS_URL) --log-level $(LOG_LEVEL) &
	@echo "✅ Central Router started in background"

start-technical:
	@echo "📊 Starting Technical Analysis Agent..."
	cd agents/technical_analysis && python runner.py &
	@echo "✅ Technical Analysis Agent started"

start-news:
	@echo "📰 Starting News Analysis Agent..."
	cd agents/news_analysis && python runner.py &
	@echo "✅ News Analysis Agent started"

start-sentiment:
	@echo "💭 Starting Sentiment Analysis Agent..."
	cd agents/sentiment_analysis && python runner.py &
	@echo "✅ Sentiment Analysis Agent started"

start-all-agents: start-router
	@echo "🚀 Starting all analysis agents..."
	@sleep 2  # Wait for router to initialize
	$(MAKE) start-technical
	@sleep 1
	$(MAKE) start-news
	@sleep 1
	$(MAKE) start-sentiment
	@echo "✅ All agents started successfully"

stop-agents:
	@echo "🛑 Stopping all agent processes..."
	pkill -f "run_central_router.py" || true
	pkill -f "technical_analysis.*runner.py" || true
	pkill -f "news_analysis.*runner.py" || true
	pkill -f "sentiment_analysis.*runner.py" || true
	@echo "✅ All agents stopped"

status:
	@echo "📊 Agent Status:"
	@echo "Central Router:"
	@pgrep -f "run_central_router.py" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f run_central_router.py))" || echo "  ❌ Not running"
	@echo "Technical Analysis Agent:"
	@pgrep -f "technical_analysis.*runner.py" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f technical_analysis.*runner.py))" || echo "  ❌ Not running"
	@echo "News Analysis Agent:"
	@pgrep -f "news_analysis.*runner.py" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f news_analysis.*runner.py))" || echo "  ❌ Not running"
	@echo "Sentiment Analysis Agent:"
	@pgrep -f "sentiment_analysis.*runner.py" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f sentiment_analysis.*runner.py))" || echo "  ❌ Not running"

# Development helpers
dev-setup: install
	@echo "🔧 Setting up development environment..."
	@echo "export NATS_URL=nats://localhost:4222" > .env
	@echo "export LOG_LEVEL=DEBUG" >> .env
	@echo "export TECHNICAL_AGENT_NAME=technical-analysis-dev" >> .env
	@echo "✅ Development environment configured"

test-router:
	@echo "🧪 Testing Central Router..."
	python -c "
import asyncio
from shared.central_router import CentralRouter
async def test():
    router = CentralRouter()
    await router.start()
    print('✅ Router test passed')
    await router.stop()
asyncio.run(test())
	"

quick-demo: start-all-agents
	@echo "🎯 Quick demo setup completed!"
	@echo "📊 Router dashboard: http://localhost:8080/stats (if implemented)"
	@echo "🔍 To stop all agents: make stop-agents"
	@sleep 5
	$(MAKE) status

# Release targets
build:
	python -m build

release-test:
	python -m twine upload --repository testpypi dist/*

release:
	python -m twine upload dist/*