"""
Shared utilities and components for the multi-agent trading system.

This module contains common functionality used across all agents including:
- Base agent classes
- Message bus integration
- Common data models
- Logging utilities
- Testing helpers
"""

from .base_agent import BaseAgent, AgentConfig

__all__ = ["BaseAgent", "AgentConfig"]