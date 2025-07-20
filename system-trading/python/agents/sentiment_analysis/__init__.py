"""
Sentiment Analysis Agent

Performs advanced NLP sentiment analysis on news articles and social media
content to generate market sentiment insights.

This agent follows the CLAUDE.md specification:
- Subscribes to: raw.news.article, raw.social.post
- Publishes to: insight.sentiment

Architecture:
- Process-based agent in monorepo structure  
- Uses advanced NLP models for sentiment classification
- Tracks sentiment trends over time
"""

from .agent import SentimentAnalysisAgent, SentimentConfig

__all__ = ["SentimentAnalysisAgent", "SentimentConfig"]