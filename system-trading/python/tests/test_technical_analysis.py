"""
Tests for Technical Analysis Agent

Comprehensive test suite covering:
- Agent initialization and lifecycle
- Market data processing
- Technical indicator calculations
- Message handling and publishing
- Error scenarios
"""

import asyncio
import json
from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, patch

import numpy as np
import pandas as pd
import pytest

from agents.technical_analysis import (
    MarketData,
    TechnicalAnalysisAgent,
    TechnicalConfig,
    TechnicalIndicators,
)


@pytest.fixture
def technical_config():
    """Create test configuration for Technical Analysis Agent"""
    return TechnicalConfig(
        agent_name="test-technical-agent",
        nats_url="nats://localhost:4222",
        log_level="DEBUG",
        data_window_size=100,
        min_bars_required=20,
        publish_frequency=1
    )


@pytest.fixture
def mock_nats_client():
    """Create mock NATS client"""
    client = AsyncMock()
    client.connect = AsyncMock()
    client.close = AsyncMock()
    client.publish = AsyncMock()
    client.subscribe = AsyncMock()
    return client


@pytest.fixture
def sample_market_data():
    """Generate sample market data for testing"""
    base_time = datetime.now()
    data_points = []
    
    # Generate 50 bars of sample OHLCV data
    for i in range(50):
        timestamp = base_time + timedelta(minutes=i)
        # Simple trending data
        base_price = 100 + i * 0.5 + np.random.normal(0, 0.5)
        
        data_point = MarketData(
            symbol="AAPL",
            timestamp=timestamp.isoformat(),
            open_price=base_price + np.random.normal(0, 0.1),
            high_price=base_price + abs(np.random.normal(0.2, 0.1)),
            low_price=base_price - abs(np.random.normal(0.2, 0.1)),
            close_price=base_price + np.random.normal(0, 0.1),
            volume=1000000 + np.random.randint(-100000, 100000),
            source="test"
        )
        data_points.append(data_point)
    
    return data_points


class TestTechnicalAnalysisAgent:
    """Test suite for Technical Analysis Agent"""
    
    @pytest.mark.asyncio
    async def test_agent_initialization(self, technical_config):
        """Test agent initialization"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        assert agent.config.agent_name == "test-technical-agent"
        assert agent.config.min_bars_required == 20
        assert len(agent.market_data) == 0
        assert agent.data_count == 0
        assert not agent.running
    
    @pytest.mark.asyncio
    async def test_agent_start_stop(self, technical_config, mock_nats_client):
        """Test agent start and stop lifecycle"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        with patch('nats.connect', return_value=mock_nats_client):
            await agent.start()
            
            assert agent.running
            assert agent.nats_client == mock_nats_client
            assert "market_data" in agent.subscriptions
            
            await agent.stop()
            
            assert not agent.running
            assert len(agent.subscriptions) == 0
    
    @pytest.mark.asyncio
    async def test_market_data_processing(self, technical_config, sample_market_data):
        """Test market data processing and storage"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Process several data points
        for i, data_point in enumerate(sample_market_data[:10]):
            agent._update_market_data(data_point)
            
            # Check data is stored correctly
            assert "AAPL" in agent.market_data
            assert len(agent.market_data["AAPL"]) == i + 1
        
        # Verify data structure
        aapl_data = agent.market_data["AAPL"]
        assert list(aapl_data.columns) == ['timestamp', 'open', 'high', 'low', 'close', 'volume']
        assert len(aapl_data) == 10
        
        # Test data window management
        for data_point in sample_market_data[10:]:
            agent._update_market_data(data_point)
        
        assert len(agent.market_data["AAPL"]) == len(sample_market_data)
    
    @pytest.mark.asyncio
    async def test_technical_indicators_calculation(self, technical_config, sample_market_data):
        """Test technical indicator calculations"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Load enough data for calculations
        for data_point in sample_market_data:
            agent._update_market_data(data_point)
        
        # Calculate indicators
        symbol_data = agent.market_data["AAPL"]
        indicators = agent._calculate_indicators("AAPL", symbol_data)
        
        # Check that indicators are calculated
        assert indicators.symbol == "AAPL"
        assert indicators.bars_analyzed == len(sample_market_data)
        assert indicators.confidence > 0
        
        # Check specific indicators (with sufficient data)
        assert indicators.rsi is not None
        assert 0 <= indicators.rsi <= 100
        
        assert indicators.macd is not None
        assert indicators.macd_signal is not None
        assert indicators.macd_histogram is not None
        
        assert indicators.bb_upper is not None
        assert indicators.bb_middle is not None  
        assert indicators.bb_lower is not None
        assert indicators.bb_width is not None
        assert indicators.bb_position is not None
        
        assert indicators.sma_short is not None
        assert indicators.sma_long is not None
        assert indicators.ema_short is not None
        assert indicators.ema_long is not None
        
        assert indicators.volume_sma is not None
        assert indicators.volume_ratio is not None
        
        # Check signals
        assert indicators.trend_signal in ["bullish", "bearish", "neutral"]
        assert indicators.momentum_signal in ["overbought", "oversold", "neutral"]
    
    @pytest.mark.asyncio
    async def test_insufficient_data_handling(self, technical_config):
        """Test handling of insufficient data"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Create minimal data (less than min_bars_required)
        minimal_data = [
            MarketData(
                symbol="AAPL",
                timestamp=datetime.now().isoformat(),
                open_price=100.0,
                high_price=101.0,
                low_price=99.0,
                close_price=100.5,
                volume=1000000,
                source="test"
            )
        ]
        
        # Process insufficient data
        agent._update_market_data(minimal_data[0])
        
        # Should have data stored but not enough for analysis
        assert "AAPL" in agent.market_data
        assert len(agent.market_data["AAPL"]) == 1
        
        # Verify we can't calculate indicators with insufficient data
        symbol_data = agent.market_data["AAPL"]
        indicators = agent._calculate_indicators("AAPL", symbol_data)
        
        # Most indicators should be None due to insufficient data
        assert indicators.rsi is None
        assert indicators.macd is None
        assert indicators.confidence < 1.0
    
    @pytest.mark.asyncio
    async def test_trend_signal_determination(self, technical_config):
        """Test trend signal determination logic"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Test bullish scenario
        bullish_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0,
            macd=1.5,
            macd_signal=1.0,
            sma_short=105.0,
            sma_long=100.0,
            ema_short=106.0,
            ema_long=101.0
        )
        
        trend = agent._determine_trend_signal(bullish_indicators)
        assert trend == "bullish"
        
        # Test bearish scenario
        bearish_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0,
            macd=-1.5,
            macd_signal=-1.0,
            sma_short=95.0,
            sma_long=100.0,
            ema_short=94.0,
            ema_long=99.0
        )
        
        trend = agent._determine_trend_signal(bearish_indicators)
        assert trend == "bearish"
        
        # Test neutral scenario
        neutral_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0
        )
        
        trend = agent._determine_trend_signal(neutral_indicators)
        assert trend == "neutral"
    
    @pytest.mark.asyncio
    async def test_momentum_signal_determination(self, technical_config):
        """Test momentum signal determination logic"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Test overbought
        overbought_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0,
            rsi=75.0
        )
        
        momentum = agent._determine_momentum_signal(overbought_indicators)
        assert momentum == "overbought"
        
        # Test oversold
        oversold_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0,
            rsi=25.0
        )
        
        momentum = agent._determine_momentum_signal(oversold_indicators)
        assert momentum == "oversold"
        
        # Test neutral
        neutral_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=1.0,
            rsi=50.0
        )
        
        momentum = agent._determine_momentum_signal(neutral_indicators)
        assert momentum == "neutral"
    
    @pytest.mark.asyncio
    async def test_confidence_calculation(self, technical_config):
        """Test confidence score calculation"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Test with full indicators
        full_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=100,  # 5x min_bars_required (20)
            confidence=0.0,  # Will be calculated
            rsi=50.0,
            macd=1.0,
            macd_signal=0.5,
            bb_upper=105.0,
            bb_lower=95.0,
            sma_short=102.0,
            sma_long=98.0,
            ema_short=103.0,
            ema_long=97.0,
            volume_sma=1000000.0,
            volume_ratio=1.2
        )
        
        confidence = agent._calculate_confidence(full_indicators, 100)
        assert confidence > 0.8  # Should be high with complete data
        
        # Test with minimal indicators
        minimal_indicators = TechnicalIndicators(
            symbol="TEST",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=20,  # Exactly min_bars_required
            confidence=0.0,
            rsi=50.0  # Only RSI available
        )
        
        confidence = agent._calculate_confidence(minimal_indicators, 20)
        assert 0.1 < confidence < 0.5  # Lower confidence with minimal data
    
    @pytest.mark.asyncio
    async def test_message_publishing(self, technical_config, mock_nats_client):
        """Test message publishing functionality"""
        agent = TechnicalAnalysisAgent(technical_config)
        agent.nats_client = mock_nats_client
        
        # Create test indicators
        indicators = TechnicalIndicators(
            symbol="AAPL",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=0.9,
            rsi=65.0,
            trend_signal="bullish",
            momentum_signal="neutral"
        )
        
        # Test publishing
        await agent._publish_indicators(indicators)
        
        # Verify publish was called
        mock_nats_client.publish.assert_called_once()
        
        # Check the published message
        call_args = mock_nats_client.publish.call_args
        topic, data = call_args[0]
        
        assert topic == "insight.technical"
        
        # Decode and verify message content
        message = json.loads(data.decode())
        assert message["symbol"] == "AAPL"
        assert message["rsi"] == 65.0
        assert message["trend_signal"] == "bullish"
        assert message["momentum_signal"] == "neutral"
    
    @pytest.mark.asyncio
    async def test_error_handling(self, technical_config, mock_nats_client):
        """Test error handling in various scenarios"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Test invalid market data
        invalid_data = {
            "symbol": "AAPL",
            "timestamp": "invalid-timestamp",
            # Missing required fields
        }
        
        # Should handle gracefully without crashing
        await agent._handle_market_data(invalid_data)
        
        # Test publishing error
        agent.nats_client = mock_nats_client
        mock_nats_client.publish.side_effect = Exception("Connection failed")
        
        indicators = TechnicalIndicators(
            symbol="AAPL",
            timestamp=datetime.now().isoformat(),
            bars_analyzed=50,
            confidence=0.9
        )
        
        # Should handle publish error gracefully
        await agent._publish_indicators(indicators)
    
    @pytest.mark.asyncio
    async def test_data_window_management(self, technical_config):
        """Test data window size management"""
        # Set small window for testing
        technical_config.data_window_size = 10
        agent = TechnicalAnalysisAgent(technical_config)
        
        # Add more data than window size
        base_time = datetime.now()
        for i in range(15):
            data_point = MarketData(
                symbol="AAPL",
                timestamp=(base_time + timedelta(minutes=i)).isoformat(),
                open_price=100.0,
                high_price=101.0,
                low_price=99.0,
                close_price=100.0,
                volume=1000000,
                source="test"
            )
            agent._update_market_data(data_point)
        
        # Should maintain window size
        assert len(agent.market_data["AAPL"]) == 10
        
        # Should keep the most recent data
        latest_timestamp = agent.market_data["AAPL"]["timestamp"].iloc[-1]
        expected_timestamp = pd.to_datetime((base_time + timedelta(minutes=14)).isoformat())
        assert latest_timestamp == expected_timestamp
    
    @pytest.mark.asyncio
    async def test_multiple_symbols(self, technical_config):
        """Test handling multiple symbols simultaneously"""
        agent = TechnicalAnalysisAgent(technical_config)
        
        symbols = ["AAPL", "GOOGL", "MSFT"]
        base_time = datetime.now()
        
        # Add data for multiple symbols
        for symbol in symbols:
            for i in range(5):
                data_point = MarketData(
                    symbol=symbol,
                    timestamp=(base_time + timedelta(minutes=i)).isoformat(),
                    open_price=100.0 + i,
                    high_price=101.0 + i,
                    low_price=99.0 + i,
                    close_price=100.5 + i,
                    volume=1000000,
                    source="test"
                )
                agent._update_market_data(data_point)
        
        # All symbols should be stored separately
        assert len(agent.market_data) == 3
        for symbol in symbols:
            assert symbol in agent.market_data
            assert len(agent.market_data[symbol]) == 5


if __name__ == "__main__":
    pytest.main([__file__, "-v"])