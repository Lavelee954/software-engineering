"""
Technical Analysis Agent Configuration

Configuration management for the Technical Analysis Agent with
environment-based loading and validation.
"""

import os
from typing import Optional

# Import from shared config base (using local import path for now)
try:
    from shared.base_agent import AgentConfig
except ImportError:
    # Fallback for standalone testing
    from pydantic import BaseModel, Field
    
    class AgentConfig(BaseModel):
        agent_name: str = Field(..., description="Unique name for this agent instance")
        nats_url: str = Field(default="nats://localhost:4222", description="NATS server URL")
        log_level: str = Field(default="INFO", description="Logging level")
        shutdown_timeout: int = Field(default=30, description="Graceful shutdown timeout in seconds")


class TechnicalConfig(AgentConfig):
    """Configuration specific to Technical Analysis Agent"""
    
    # Agent capabilities for router discovery
    capabilities: list = Field(
        default=["technical_analysis", "indicator_calculation", "signal_generation"],
        description="Technical analysis capabilities"
    )
    
    # Data management settings
    data_window_size: int = Field(default=200, description="Number of bars to keep in memory")
    min_bars_required: int = Field(default=50, description="Minimum bars needed for analysis")
    publish_frequency: int = Field(default=1, description="Publish every N data points")
    
    # RSI settings
    rsi_period: int = Field(default=14, description="RSI calculation period")
    
    # MACD settings
    macd_fast: int = Field(default=12, description="MACD fast EMA period")
    macd_slow: int = Field(default=26, description="MACD slow EMA period")
    macd_signal: int = Field(default=9, description="MACD signal line period")
    
    # Bollinger Bands settings
    bb_period: int = Field(default=20, description="Bollinger Bands period")
    bb_std: float = Field(default=2.0, description="Bollinger Bands standard deviations")
    
    # Moving averages settings
    sma_short: int = Field(default=20, description="Short-term SMA period")
    sma_long: int = Field(default=50, description="Long-term SMA period")
    ema_short: int = Field(default=12, description="Short-term EMA period")
    ema_long: int = Field(default=26, description="Long-term EMA period")
    
    # Volume indicators settings
    volume_sma_period: int = Field(default=20, description="Volume SMA period")


def load_config() -> TechnicalConfig:
    """
    Load Technical Analysis Agent configuration from environment variables
    with sensible defaults and validation.
    """
    
    # Load environment variables with .env file support
    try:
        from dotenv import load_dotenv
        load_dotenv()
    except ImportError:
        # dotenv not available, use environment variables directly
        pass
    
    return TechnicalConfig(
        # Base agent configuration
        agent_name=os.getenv("TECHNICAL_AGENT_NAME", "technical-analysis-agent"),
        nats_url=os.getenv("NATS_URL", "nats://localhost:4222"),
        log_level=os.getenv("LOG_LEVEL", "INFO"),
        shutdown_timeout=int(os.getenv("SHUTDOWN_TIMEOUT", "30")),
        
        # Data management
        data_window_size=int(os.getenv("TECHNICAL_DATA_WINDOW_SIZE", "200")),
        min_bars_required=int(os.getenv("TECHNICAL_MIN_BARS_REQUIRED", "50")),
        publish_frequency=int(os.getenv("TECHNICAL_PUBLISH_FREQUENCY", "1")),
        
        # RSI configuration
        rsi_period=int(os.getenv("TECHNICAL_RSI_PERIOD", "14")),
        
        # MACD configuration
        macd_fast=int(os.getenv("TECHNICAL_MACD_FAST", "12")),
        macd_slow=int(os.getenv("TECHNICAL_MACD_SLOW", "26")),
        macd_signal=int(os.getenv("TECHNICAL_MACD_SIGNAL", "9")),
        
        # Bollinger Bands configuration
        bb_period=int(os.getenv("TECHNICAL_BB_PERIOD", "20")),
        bb_std=float(os.getenv("TECHNICAL_BB_STD", "2.0")),
        
        # Moving averages configuration
        sma_short=int(os.getenv("TECHNICAL_SMA_SHORT", "20")),
        sma_long=int(os.getenv("TECHNICAL_SMA_LONG", "50")),
        ema_short=int(os.getenv("TECHNICAL_EMA_SHORT", "12")),
        ema_long=int(os.getenv("TECHNICAL_EMA_LONG", "26")),
        
        # Volume indicators
        volume_sma_period=int(os.getenv("TECHNICAL_VOLUME_SMA_PERIOD", "20"))
    )


def get_config_summary(config: TechnicalConfig) -> str:
    """Get a formatted summary of the configuration"""
    return f"""
Technical Analysis Agent Configuration:
======================================
Agent Name: {config.agent_name}
NATS URL: {config.nats_url}
Log Level: {config.log_level}

Data Management:
- Window Size: {config.data_window_size}
- Min Bars: {config.min_bars_required}
- Publish Frequency: {config.publish_frequency}

Indicators:
- RSI Period: {config.rsi_period}
- MACD: {config.macd_fast}/{config.macd_slow}/{config.macd_signal}
- Bollinger Bands: {config.bb_period} periods, {config.bb_std} std dev
- SMA: {config.sma_short}/{config.sma_long}
- EMA: {config.ema_short}/{config.ema_long}
- Volume SMA: {config.volume_sma_period}
"""


def validate_config(config: TechnicalConfig) -> bool:
    """
    Validate configuration for logical consistency
    
    Returns:
        True if configuration is valid, raises ValueError if invalid
    """
    
    # Validate periods are positive
    if config.rsi_period <= 0:
        raise ValueError("RSI period must be positive")
    
    if config.macd_fast >= config.macd_slow:
        raise ValueError("MACD fast period must be less than slow period")
    
    if config.sma_short >= config.sma_long:
        raise ValueError("Short SMA period must be less than long SMA period")
    
    if config.ema_short >= config.ema_long:
        raise ValueError("Short EMA period must be less than long EMA period")
    
    if config.bb_std <= 0:
        raise ValueError("Bollinger Bands standard deviation must be positive")
    
    # Validate data management settings
    if config.data_window_size <= config.min_bars_required:
        raise ValueError("Data window size must be larger than minimum bars required")
    
    if config.min_bars_required < max(config.sma_long, config.ema_long, config.bb_period):
        raise ValueError("Minimum bars required must be at least as large as the longest indicator period")
    
    return True


# Configuration presets for different use cases
DEVELOPMENT_CONFIG = {
    "log_level": "DEBUG",
    "data_window_size": 100,
    "min_bars_required": 30,
    "publish_frequency": 1
}

PRODUCTION_CONFIG = {
    "log_level": "INFO", 
    "data_window_size": 500,
    "min_bars_required": 100,
    "publish_frequency": 1
}

HIGH_FREQUENCY_CONFIG = {
    "log_level": "WARN",
    "data_window_size": 50,
    "min_bars_required": 20,
    "publish_frequency": 1,
    "rsi_period": 7,
    "macd_fast": 5,
    "macd_slow": 10,
    "bb_period": 10,
    "sma_short": 5,
    "sma_long": 20
}

LONG_TERM_CONFIG = {
    "log_level": "INFO",
    "data_window_size": 1000,
    "min_bars_required": 200,
    "publish_frequency": 5,
    "rsi_period": 21,
    "macd_fast": 26,
    "macd_slow": 52,
    "bb_period": 50,
    "sma_short": 50,
    "sma_long": 200
}


def load_preset_config(preset: str) -> TechnicalConfig:
    """Load a preset configuration"""
    base_config = load_config()
    
    presets = {
        "development": DEVELOPMENT_CONFIG,
        "production": PRODUCTION_CONFIG,
        "high_frequency": HIGH_FREQUENCY_CONFIG,
        "long_term": LONG_TERM_CONFIG
    }
    
    if preset not in presets:
        raise ValueError(f"Unknown preset: {preset}. Available: {list(presets.keys())}")
    
    # Update base config with preset values
    preset_values = presets[preset]
    config_dict = base_config.model_dump()
    config_dict.update(preset_values)
    
    return TechnicalConfig(**config_dict)