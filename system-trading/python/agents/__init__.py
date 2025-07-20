"""
Multi-Asset Trading System - Python Analysis Agents (Monorepo)

This package contains all Python-based analysis agents in a monorepo structure
where each agent runs as an independent process but shares common infrastructure.

Agents (Process-Based):
- TechnicalAnalysisAgent: Processes market data to generate technical indicators
- NewsAnalysisAgent: Analyzes news articles for market sentiment
- SentimentAnalysisAgent: Performs NLP sentiment analysis
- MacroEconomicAgent: Analyzes macroeconomic indicators (Coming Soon)
- StrategyAgent: Makes trading decisions based on insights (Coming Soon)
- BacktestAgent: Runs historical simulations (Coming Soon)

Architecture:
- Monorepo: All agents in single repository with shared infrastructure
- Process Isolation: Each agent runs as independent process
- Shared Base: Common utilities, models, and base classes
- Universal Starter: Single script to launch any agent type
"""

__version__ = "0.2.0"
__author__ = "Trading System Team"
__architecture__ = "monorepo-multi-agent"

# Import agents that are implemented
from .technical_analysis import TechnicalAnalysisAgent, TechnicalConfig
from .news_analysis import NewsAnalysisAgent, NewsConfig
from .sentiment_analysis import SentimentAnalysisAgent, SentimentConfig

__all__ = [
    "TechnicalAnalysisAgent",
    "TechnicalConfig",
    "NewsAnalysisAgent", 
    "NewsConfig",
    "SentimentAnalysisAgent",
    "SentimentConfig",
]

# Registry of available agents (for universal starter)
AGENT_REGISTRY = {
    "technical_analysis": {
        "class": TechnicalAnalysisAgent,
        "config": TechnicalConfig,
        "runner_module": "agents.technical_analysis.runner",
        "status": "implemented"
    },
    "news_analysis": {
        "class": NewsAnalysisAgent,
        "config": NewsConfig,
        "runner_module": "agents.news_analysis.runner",
        "status": "implemented"
    },
    "sentiment_analysis": {
        "class": SentimentAnalysisAgent,
        "config": SentimentConfig,
        "runner_module": "agents.sentiment_analysis.runner",
        "status": "implemented"
    },
    "macro_economic": {
        "class": None,
        "config": None,
        "runner_module": "agents.macro_economic.runner",
        "status": "planned"
    },
    "strategy": {
        "class": None,
        "config": None,
        "runner_module": "agents.strategy.runner",
        "status": "planned"
    },
    "backtest": {
        "class": None,
        "config": None,
        "runner_module": "agents.backtest.runner",
        "status": "planned"
    }
}