"""
News Analysis Agent

Analyzes news articles for market-relevant information and publishes insights
to the message bus for consumption by strategy agents.

This agent follows the CLAUDE.md specification:
- Subscribes to: raw.news.article
- Publishes to: insight.news

Architecture:
- Process-based agent in monorepo structure
- Uses NLP techniques to extract key information from news
- Maintains news history for trend analysis
"""

from .agent import NewsAnalysisAgent, NewsConfig

__all__ = ["NewsAnalysisAgent", "NewsConfig"]