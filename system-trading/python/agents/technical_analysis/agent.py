"""
Technical Analysis Agent

Subscribes to raw market data and generates technical indicators including:
- RSI (Relative Strength Index)
- MACD (Moving Average Convergence Divergence) 
- Bollinger Bands
- Moving Averages (SMA, EMA)
- Volume indicators

Publishes results to the 'insight.technical' topic as specified in CLAUDE.md.
"""

import asyncio
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

import numpy as np
import pandas as pd
import talib
from pydantic import BaseModel, Field

try:
    # Import from shared module for router integration
    from shared.base_agent import AgentConfig, BaseAgent
except ImportError:
    # Fallback for standalone testing
    from .base import AgentConfig, BaseAgent


class TechnicalConfig(AgentConfig):
    """Configuration specific to Technical Analysis Agent"""
    
    # Data window settings
    data_window_size: int = Field(default=200, description="Number of bars to keep in memory")
    min_bars_required: int = Field(default=50, description="Minimum bars needed for analysis")
    
    # Indicator settings
    rsi_period: int = Field(default=14, description="RSI calculation period")
    macd_fast: int = Field(default=12, description="MACD fast EMA period")
    macd_slow: int = Field(default=26, description="MACD slow EMA period") 
    macd_signal: int = Field(default=9, description="MACD signal line period")
    bb_period: int = Field(default=20, description="Bollinger Bands period")
    bb_std: float = Field(default=2.0, description="Bollinger Bands standard deviations")
    sma_short: int = Field(default=20, description="Short-term SMA period")
    sma_long: int = Field(default=50, description="Long-term SMA period")
    ema_short: int = Field(default=12, description="Short-term EMA period")
    ema_long: int = Field(default=26, description="Long-term EMA period")
    
    # Volume indicators
    volume_sma_period: int = Field(default=20, description="Volume SMA period")
    
    # Publishing settings
    publish_frequency: int = Field(default=1, description="Publish every N data points")


class MarketData(BaseModel):
    """Market data structure matching the Go entities"""
    
    symbol: str
    timestamp: str
    open_price: float
    high_price: float
    low_price: float
    close_price: float
    volume: float
    source: str


class TechnicalIndicators(BaseModel):
    """Technical analysis results structure"""
    
    symbol: str
    timestamp: str
    
    # Price-based indicators
    rsi: Optional[float] = None
    macd: Optional[float] = None
    macd_signal: Optional[float] = None
    macd_histogram: Optional[float] = None
    bb_upper: Optional[float] = None
    bb_middle: Optional[float] = None
    bb_lower: Optional[float] = None
    bb_width: Optional[float] = None
    bb_position: Optional[float] = None  # Where price is within bands (0-1)
    
    # Moving averages
    sma_short: Optional[float] = None
    sma_long: Optional[float] = None
    ema_short: Optional[float] = None
    ema_long: Optional[float] = None
    
    # Volume indicators
    volume_sma: Optional[float] = None
    volume_ratio: Optional[float] = None  # Current volume / average volume
    
    # Trend signals
    trend_signal: Optional[str] = None  # "bullish", "bearish", "neutral"
    momentum_signal: Optional[str] = None  # "overbought", "oversold", "neutral"
    
    # Meta information
    bars_analyzed: int
    confidence: float  # 0-1 based on data quality and completeness


class TechnicalAnalysisAgent(BaseAgent):
    """
    Technical Analysis Agent that processes market data and generates indicators.
    
    Follows the CLAUDE.md specification:
    - Subscribes to: raw.market_data.price
    - Publishes to: insight.technical
    """
    
    def __init__(self, config: TechnicalConfig) -> None:
        super().__init__(config)
        self.config: TechnicalConfig = config
        
        # Data storage - organized by symbol
        self.market_data: Dict[str, pd.DataFrame] = {}
        self.data_count = 0
        
        self.logger = self.logger.bind(component="technical_analysis")
    
    async def _setup_subscriptions(self) -> None:
        """Setup NATS subscriptions for market data"""
        if not self.nats_client:
            raise RuntimeError("NATS client not connected")
        
        # Subscribe to raw market data
        handler = await self._message_handler(
            "raw.market_data.price", 
            self._handle_market_data
        )
        
        subscription = await self.nats_client.subscribe(
            "raw.market_data.price", 
            cb=handler
        )
        
        self.subscriptions["market_data"] = subscription
        
        self.logger.info("Subscribed to market data topic")
    
    async def _initialize(self) -> None:
        """Initialize the technical analysis agent"""
        self.logger.info(
            "Technical Analysis Agent initialized",
            data_window=self.config.data_window_size,
            min_bars=self.config.min_bars_required
        )
    
    async def _cleanup(self) -> None:
        """Cleanup agent resources"""
        self.market_data.clear()
        self.logger.info("Technical Analysis Agent cleaned up")
    
    async def _handle_market_data(self, data: Dict[str, Any]) -> None:
        """Process incoming market data and generate technical indicators"""
        try:
            # Parse market data
            market_data = MarketData(**data)
            
            # Update data store
            self._update_market_data(market_data)
            
            # Check if we have enough data for analysis
            symbol_data = self.market_data.get(market_data.symbol)
            if symbol_data is None or len(symbol_data) < self.config.min_bars_required:
                self.logger.debug(
                    "Insufficient data for analysis",
                    symbol=market_data.symbol,
                    bars=len(symbol_data) if symbol_data is not None else 0,
                    required=self.config.min_bars_required
                )
                return
            
            # Generate technical indicators
            indicators = self._calculate_indicators(market_data.symbol, symbol_data)
            
            # Check if we should publish (based on frequency setting)
            self.data_count += 1
            if self.data_count % self.config.publish_frequency == 0:
                await self._publish_indicators(indicators)
            
        except Exception as e:
            self.logger.error(
                "Error processing market data", 
                error=str(e),
                data=data
            )
    
    def _update_market_data(self, market_data: MarketData) -> None:
        """Update the market data store for a symbol"""
        symbol = market_data.symbol
        
        # Create new DataFrame row
        new_row = pd.DataFrame({
            'timestamp': [pd.to_datetime(market_data.timestamp)],
            'open': [market_data.open_price],
            'high': [market_data.high_price], 
            'low': [market_data.low_price],
            'close': [market_data.close_price],
            'volume': [market_data.volume]
        })
        
        if symbol not in self.market_data:
            # Initialize new symbol data
            self.market_data[symbol] = new_row
        else:
            # Append to existing data
            self.market_data[symbol] = pd.concat(
                [self.market_data[symbol], new_row], 
                ignore_index=True
            )
            
            # Maintain window size
            if len(self.market_data[symbol]) > self.config.data_window_size:
                self.market_data[symbol] = self.market_data[symbol].tail(
                    self.config.data_window_size
                )
        
        # Sort by timestamp to ensure proper order
        self.market_data[symbol] = self.market_data[symbol].sort_values('timestamp')
        self.market_data[symbol].reset_index(drop=True, inplace=True)
        
        self.logger.debug(
            "Updated market data",
            symbol=symbol,
            total_bars=len(self.market_data[symbol])
        )
    
    def _calculate_indicators(self, symbol: str, data: pd.DataFrame) -> TechnicalIndicators:
        """Calculate technical indicators for the given data"""
        
        # Get latest timestamp and bar count
        latest_timestamp = data['timestamp'].iloc[-1].isoformat()
        bars_analyzed = len(data)
        
        # Extract price and volume arrays
        open_prices = data['open'].values
        high_prices = data['high'].values
        low_prices = data['low'].values
        close_prices = data['close'].values
        volumes = data['volume'].values
        
        # Initialize indicators with None values
        indicators = TechnicalIndicators(
            symbol=symbol,
            timestamp=latest_timestamp,
            bars_analyzed=bars_analyzed,
            confidence=0.0
        )
        
        try:
            # RSI
            if len(close_prices) >= self.config.rsi_period:
                rsi_values = talib.RSI(close_prices, timeperiod=self.config.rsi_period)
                indicators.rsi = float(rsi_values[-1]) if not np.isnan(rsi_values[-1]) else None
            
            # MACD
            if len(close_prices) >= max(self.config.macd_fast, self.config.macd_slow):
                macd, macd_signal, macd_hist = talib.MACD(
                    close_prices,
                    fastperiod=self.config.macd_fast,
                    slowperiod=self.config.macd_slow,
                    signalperiod=self.config.macd_signal
                )
                indicators.macd = float(macd[-1]) if not np.isnan(macd[-1]) else None
                indicators.macd_signal = float(macd_signal[-1]) if not np.isnan(macd_signal[-1]) else None
                indicators.macd_histogram = float(macd_hist[-1]) if not np.isnan(macd_hist[-1]) else None
            
            # Bollinger Bands
            if len(close_prices) >= self.config.bb_period:
                bb_upper, bb_middle, bb_lower = talib.BBANDS(
                    close_prices,
                    timeperiod=self.config.bb_period,
                    nbdevup=self.config.bb_std,
                    nbdevdn=self.config.bb_std
                )
                indicators.bb_upper = float(bb_upper[-1]) if not np.isnan(bb_upper[-1]) else None
                indicators.bb_middle = float(bb_middle[-1]) if not np.isnan(bb_middle[-1]) else None
                indicators.bb_lower = float(bb_lower[-1]) if not np.isnan(bb_lower[-1]) else None
                
                # Calculate band width and position
                if all(x is not None for x in [indicators.bb_upper, indicators.bb_lower]):
                    indicators.bb_width = indicators.bb_upper - indicators.bb_lower
                    current_price = close_prices[-1]
                    if indicators.bb_width > 0:
                        indicators.bb_position = (current_price - indicators.bb_lower) / indicators.bb_width
            
            # Moving Averages
            if len(close_prices) >= self.config.sma_short:
                sma_short = talib.SMA(close_prices, timeperiod=self.config.sma_short)
                indicators.sma_short = float(sma_short[-1]) if not np.isnan(sma_short[-1]) else None
            
            if len(close_prices) >= self.config.sma_long:
                sma_long = talib.SMA(close_prices, timeperiod=self.config.sma_long)
                indicators.sma_long = float(sma_long[-1]) if not np.isnan(sma_long[-1]) else None
            
            if len(close_prices) >= self.config.ema_short:
                ema_short = talib.EMA(close_prices, timeperiod=self.config.ema_short)
                indicators.ema_short = float(ema_short[-1]) if not np.isnan(ema_short[-1]) else None
            
            if len(close_prices) >= self.config.ema_long:
                ema_long = talib.EMA(close_prices, timeperiod=self.config.ema_long)
                indicators.ema_long = float(ema_long[-1]) if not np.isnan(ema_long[-1]) else None
            
            # Volume indicators
            if len(volumes) >= self.config.volume_sma_period:
                volume_sma = talib.SMA(volumes, timeperiod=self.config.volume_sma_period)
                indicators.volume_sma = float(volume_sma[-1]) if not np.isnan(volume_sma[-1]) else None
                
                if indicators.volume_sma and indicators.volume_sma > 0:
                    indicators.volume_ratio = volumes[-1] / indicators.volume_sma
            
            # Generate signals based on indicators
            indicators.trend_signal = self._determine_trend_signal(indicators)
            indicators.momentum_signal = self._determine_momentum_signal(indicators)
            
            # Calculate confidence based on data completeness
            indicators.confidence = self._calculate_confidence(indicators, bars_analyzed)
            
            self.logger.debug(
                "Calculated technical indicators",
                symbol=symbol,
                rsi=indicators.rsi,
                trend=indicators.trend_signal,
                momentum=indicators.momentum_signal,
                confidence=indicators.confidence
            )
            
        except Exception as e:
            self.logger.error(
                "Error calculating indicators",
                symbol=symbol,
                error=str(e)
            )
        
        return indicators
    
    def _determine_trend_signal(self, indicators: TechnicalIndicators) -> str:
        """Determine overall trend signal from multiple indicators"""
        bullish_signals = 0
        bearish_signals = 0
        total_signals = 0
        
        # MACD trend
        if indicators.macd is not None and indicators.macd_signal is not None:
            total_signals += 1
            if indicators.macd > indicators.macd_signal:
                bullish_signals += 1
            else:
                bearish_signals += 1
        
        # Moving average crossover
        if indicators.sma_short is not None and indicators.sma_long is not None:
            total_signals += 1
            if indicators.sma_short > indicators.sma_long:
                bullish_signals += 1
            else:
                bearish_signals += 1
        
        # EMA trend
        if indicators.ema_short is not None and indicators.ema_long is not None:
            total_signals += 1
            if indicators.ema_short > indicators.ema_long:
                bullish_signals += 1
            else:
                bearish_signals += 1
        
        if total_signals == 0:
            return "neutral"
        
        # Determine overall trend
        bullish_ratio = bullish_signals / total_signals
        if bullish_ratio >= 0.67:
            return "bullish"
        elif bullish_ratio <= 0.33:
            return "bearish"
        else:
            return "neutral"
    
    def _determine_momentum_signal(self, indicators: TechnicalIndicators) -> str:
        """Determine momentum signal primarily from RSI"""
        if indicators.rsi is None:
            return "neutral"
        
        if indicators.rsi >= 70:
            return "overbought"
        elif indicators.rsi <= 30:
            return "oversold"
        else:
            return "neutral"
    
    def _calculate_confidence(self, indicators: TechnicalIndicators, bars_analyzed: int) -> float:
        """Calculate confidence score based on data quality and completeness"""
        
        # Base confidence on number of bars relative to requirements
        data_confidence = min(1.0, bars_analyzed / (self.config.min_bars_required * 2))
        
        # Count available indicators
        available_indicators = 0
        total_indicators = 0
        
        # Price indicators
        for field in ['rsi', 'macd', 'macd_signal', 'bb_upper', 'bb_lower', 
                     'sma_short', 'sma_long', 'ema_short', 'ema_long']:
            total_indicators += 1
            if getattr(indicators, field) is not None:
                available_indicators += 1
        
        # Volume indicators
        for field in ['volume_sma', 'volume_ratio']:
            total_indicators += 1
            if getattr(indicators, field) is not None:
                available_indicators += 1
        
        indicator_confidence = available_indicators / total_indicators if total_indicators > 0 else 0
        
        # Overall confidence is the average of data and indicator confidence
        return (data_confidence + indicator_confidence) / 2
    
    async def _publish_indicators(self, indicators: TechnicalIndicators) -> None:
        """Publish technical indicators to the insight.technical topic"""
        try:
            message = indicators.model_dump()
            await self.publish("insight.technical", message)
            
            self.logger.info(
                "Published technical indicators",
                symbol=indicators.symbol,
                trend=indicators.trend_signal,
                momentum=indicators.momentum_signal,
                confidence=indicators.confidence
            )
            
            # A2A Communication: Notify other agents if significant signals detected
            await self._check_and_notify_significant_signals(indicators)
            
        except Exception as e:
            self.logger.error(
                "Failed to publish indicators",
                symbol=indicators.symbol,
                error=str(e)
            )
    
    async def _check_and_notify_significant_signals(self, indicators: TechnicalIndicators) -> None:
        """Check for significant signals and notify relevant agents"""
        try:
            # Detect significant RSI levels
            if indicators.rsi is not None:
                if indicators.rsi >= 75:  # Extreme overbought
                    await self._notify_risk_management_agent(
                        "extreme_overbought", 
                        indicators
                    )
                elif indicators.rsi <= 25:  # Extreme oversold
                    await self._notify_strategy_agent(
                        "potential_buy_opportunity", 
                        indicators
                    )
            
            # Detect strong trend changes
            if indicators.trend_signal == "bullish" and indicators.confidence > 0.8:
                await self._notify_strategy_agent("strong_bullish_trend", indicators)
            elif indicators.trend_signal == "bearish" and indicators.confidence > 0.8:
                await self._notify_strategy_agent("strong_bearish_trend", indicators)
            
            # High volume alerts
            if indicators.volume_ratio is not None and indicators.volume_ratio > 2.0:
                await self._broadcast_high_volume_alert(indicators)
            
        except Exception as e:
            self.logger.error("Failed to check significant signals", error=str(e))
    
    async def _notify_risk_management_agent(self, alert_type: str, indicators: TechnicalIndicators) -> None:
        """Send alert to Risk Management Agent"""
        message = {
            "alert_type": alert_type,
            "symbol": indicators.symbol,
            "timestamp": indicators.timestamp,
            "rsi": indicators.rsi,
            "confidence": indicators.confidence,
            "details": {
                "trend_signal": indicators.trend_signal,
                "momentum_signal": indicators.momentum_signal
            }
        }
        
        success = await self.route_message_to_agent(
            message=message,
            destination_type="risk_management",
            strategy="least_loaded"
        )
        
        if success:
            self.logger.info(
                "Notified Risk Management Agent",
                alert_type=alert_type,
                symbol=indicators.symbol
            )
    
    async def _notify_strategy_agent(self, signal_type: str, indicators: TechnicalIndicators) -> None:
        """Send trading signal to Strategy Agent"""
        message = {
            "signal_type": signal_type,
            "symbol": indicators.symbol,
            "timestamp": indicators.timestamp,
            "indicators": indicators.model_dump(),
            "priority": "high" if indicators.confidence > 0.8 else "medium"
        }
        
        success = await self.route_message_to_agent(
            message=message,
            destination_type="strategy",
            strategy="round_robin"
        )
        
        if success:
            self.logger.info(
                "Notified Strategy Agent",
                signal_type=signal_type,
                symbol=indicators.symbol,
                confidence=indicators.confidence
            )
    
    async def _broadcast_high_volume_alert(self, indicators: TechnicalIndicators) -> None:
        """Broadcast high volume alert to all relevant agents"""
        message = {
            "alert_type": "high_volume",
            "symbol": indicators.symbol,
            "timestamp": indicators.timestamp,
            "volume_ratio": indicators.volume_ratio,
            "volume_sma": indicators.volume_sma,
            "technical_context": {
                "trend_signal": indicators.trend_signal,
                "momentum_signal": indicators.momentum_signal,
                "rsi": indicators.rsi,
                "confidence": indicators.confidence
            }
        }
        
        # Broadcast to strategy and risk management agents
        success = await self.broadcast_message(
            message=message,
            agent_types=["strategy", "risk_management", "portfolio_management"]
        )
        
        if success:
            self.logger.info(
                "Broadcasted high volume alert",
                symbol=indicators.symbol,
                volume_ratio=indicators.volume_ratio
            )
    
    # A2A Communication Handlers
    
    async def _handle_a2a_message(self, message: Dict[str, Any]) -> None:
        """Handle Agent-to-Agent communication messages"""
        try:
            message_type = message.get("message_type", "unknown")
            sender_id = message.get("sender_id", "unknown")
            
            self.logger.info(
                "Received A2A message",
                message_type=message_type,
                sender=sender_id
            )
            
            if message_type == "indicator_request":
                await self._handle_indicator_request(message)
            elif message_type == "validation_request":
                await self._handle_validation_request(message)
            elif message_type == "configuration_update":
                await self._handle_configuration_update(message)
            else:
                self.logger.warning(
                    "Unknown A2A message type",
                    message_type=message_type,
                    sender=sender_id
                )
                
        except Exception as e:
            self.logger.error("Failed to handle A2A message", error=str(e))
    
    async def _handle_indicator_request(self, message: Dict[str, Any]) -> None:
        """Handle request for specific indicators"""
        try:
            symbol = message.get("symbol")
            requested_indicators = message.get("indicators", [])
            requester_id = message.get("sender_id")
            
            if not symbol or symbol not in self.market_data:
                self.logger.warning(
                    "Indicator request for unavailable symbol",
                    symbol=symbol,
                    requester=requester_id
                )
                return
            
            # Get latest indicators for the symbol
            data = self.market_data[symbol]
            indicators = self._calculate_indicators(symbol, data)
            
            # Filter to requested indicators only
            response_data = {}
            for indicator in requested_indicators:
                if hasattr(indicators, indicator):
                    response_data[indicator] = getattr(indicators, indicator)
            
            # Send response back to requester
            response_message = {
                "message_type": "indicator_response",
                "symbol": symbol,
                "indicators": response_data,
                "timestamp": indicators.timestamp,
                "confidence": indicators.confidence
            }
            
            await self.route_message_to_agent(
                message=response_message,
                destination_id=requester_id
            )
            
            self.logger.info(
                "Responded to indicator request",
                symbol=symbol,
                requester=requester_id,
                indicators=requested_indicators
            )
            
        except Exception as e:
            self.logger.error("Failed to handle indicator request", error=str(e))
    
    async def _handle_validation_request(self, message: Dict[str, Any]) -> None:
        """Handle cross-validation request from other agents"""
        try:
            symbol = message.get("symbol")
            external_signal = message.get("signal")
            requester_id = message.get("sender_id")
            
            if not symbol or symbol not in self.market_data:
                return
            
            # Get our technical analysis for the symbol
            data = self.market_data[symbol]
            our_indicators = self._calculate_indicators(symbol, data)
            
            # Compare signals and provide validation
            validation_result = self._validate_external_signal(external_signal, our_indicators)
            
            response_message = {
                "message_type": "validation_response",
                "symbol": symbol,
                "validation_result": validation_result,
                "our_signal": {
                    "trend": our_indicators.trend_signal,
                    "momentum": our_indicators.momentum_signal,
                    "confidence": our_indicators.confidence
                },
                "correlation_id": message.get("correlation_id")
            }
            
            await self.route_message_to_agent(
                message=response_message,
                destination_id=requester_id
            )
            
            self.logger.info(
                "Provided signal validation",
                symbol=symbol,
                requester=requester_id,
                validation=validation_result
            )
            
        except Exception as e:
            self.logger.error("Failed to handle validation request", error=str(e))
    
    def _validate_external_signal(self, external_signal: Dict[str, Any], our_indicators: TechnicalIndicators) -> Dict[str, Any]:
        """Validate an external signal against our technical analysis"""
        
        external_trend = external_signal.get("trend", "neutral")
        external_strength = external_signal.get("strength", 0.5)
        
        # Check trend alignment
        trend_agreement = external_trend == our_indicators.trend_signal
        
        # Calculate confidence adjustment based on our analysis
        confidence_adjustment = 0.0
        
        if trend_agreement:
            confidence_adjustment += 0.1  # Boost confidence for agreeing signals
            
            # Additional boost for strong technical confirmation
            if our_indicators.confidence > 0.8:
                confidence_adjustment += 0.05
                
            # RSI confirmation
            if our_indicators.rsi is not None:
                if (external_trend == "bullish" and our_indicators.rsi < 70) or \
                   (external_trend == "bearish" and our_indicators.rsi > 30):
                    confidence_adjustment += 0.05
        else:
            confidence_adjustment -= 0.1  # Reduce confidence for conflicting signals
        
        return {
            "trend_agreement": trend_agreement,
            "confidence_adjustment": confidence_adjustment,
            "technical_support": our_indicators.confidence > 0.6,
            "risk_warning": our_indicators.momentum_signal in ["overbought", "oversold"],
            "additional_context": {
                "rsi": our_indicators.rsi,
                "trend_signal": our_indicators.trend_signal,
                "momentum_signal": our_indicators.momentum_signal
            }
        }
    
    async def _handle_configuration_update(self, message: Dict[str, Any]) -> None:
        """Handle dynamic configuration updates"""
        try:
            config_updates = message.get("config_updates", {})
            
            # Apply safe configuration updates
            if "publish_frequency" in config_updates:
                self.config.publish_frequency = config_updates["publish_frequency"]
                
            if "rsi_period" in config_updates:
                new_rsi_period = config_updates["rsi_period"]
                if 5 <= new_rsi_period <= 50:  # Reasonable bounds
                    self.config.rsi_period = new_rsi_period
            
            self.logger.info(
                "Updated configuration",
                updates=config_updates,
                sender=message.get("sender_id")
            )
            
        except Exception as e:
            self.logger.error("Failed to handle configuration update", error=str(e))


async def main() -> None:
    """Main entry point for the Technical Analysis Agent"""
    import os
    import sys
    from dotenv import load_dotenv
    
    # Load environment variables
    load_dotenv()
    
    # Create configuration from environment
    config = TechnicalConfig(
        agent_name=os.getenv("TECHNICAL_AGENT_NAME", "technical-analysis-agent"),
        nats_url=os.getenv("NATS_URL", "nats://localhost:4222"),
        log_level=os.getenv("LOG_LEVEL", "INFO"),
        
        # Technical analysis specific configuration
        data_window_size=int(os.getenv("TECHNICAL_DATA_WINDOW_SIZE", "200")),
        min_bars_required=int(os.getenv("TECHNICAL_MIN_BARS_REQUIRED", "50")),
        publish_frequency=int(os.getenv("TECHNICAL_PUBLISH_FREQUENCY", "1")),
        
        # Indicator configuration
        rsi_period=int(os.getenv("TECHNICAL_RSI_PERIOD", "14")),
        macd_fast=int(os.getenv("TECHNICAL_MACD_FAST", "12")),
        macd_slow=int(os.getenv("TECHNICAL_MACD_SLOW", "26")),
        macd_signal=int(os.getenv("TECHNICAL_MACD_SIGNAL", "9")),
        bb_period=int(os.getenv("TECHNICAL_BB_PERIOD", "20")),
        bb_std=float(os.getenv("TECHNICAL_BB_STD", "2.0")),
        sma_short=int(os.getenv("TECHNICAL_SMA_SHORT", "20")),
        sma_long=int(os.getenv("TECHNICAL_SMA_LONG", "50")),
        ema_short=int(os.getenv("TECHNICAL_EMA_SHORT", "12")),
        ema_long=int(os.getenv("TECHNICAL_EMA_LONG", "26")),
        volume_sma_period=int(os.getenv("TECHNICAL_VOLUME_SMA_PERIOD", "20"))
    )
    
    # Create and run agent
    agent = TechnicalAnalysisAgent(config)
    
    try:
        await agent.run()
    except KeyboardInterrupt:
        agent.logger.info("Received shutdown signal, stopping agent")
        sys.exit(0)
    except Exception as e:
        agent.logger.error("Agent failed with error", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())