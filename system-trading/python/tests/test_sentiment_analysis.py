"""
Tests for Sentiment Analysis Agent

Tests the sentiment analysis capabilities including lexicon-based analysis,
emotion detection, and trend analysis.
"""

import asyncio
import json
import pytest
from datetime import datetime, timedelta
from unittest.mock import Mock, AsyncMock
from collections import deque

# Import test dependencies
try:
    from agents.sentiment_analysis.agent import (
        SentimentAnalysisAgent, SentimentConfig, SentimentAnalysis, SentimentTrend,
        SentimentLabel, EmotionLabel
    )
    from shared.base_agent import AgentConfig
    DEPENDENCIES_AVAILABLE = True
except ImportError:
    DEPENDENCIES_AVAILABLE = False


@pytest.mark.skipif(not DEPENDENCIES_AVAILABLE, reason="Agent dependencies not available")
class TestSentimentAnalysisAgent:
    """Test cases for Sentiment Analysis Agent"""

    @pytest.fixture
    def config(self):
        """Create test configuration"""
        return SentimentConfig(
            agent_name="test-sentiment-agent",
            nats_url="nats://localhost:4222",
            log_level="DEBUG",
            sentiment_window_minutes=15,
            confidence_threshold=0.6,
            min_content_length=10,
            max_content_age_hours=48,
            use_lexicon_analysis=True,
            use_ml_models=False,
            track_market_emotions=True
        )

    @pytest.fixture
    def sentiment_agent(self, config):
        """Create Sentiment Analysis Agent instance"""
        agent = SentimentAnalysisAgent(config)
        agent.nats_client = Mock()
        agent.logger = Mock()
        return agent

    @pytest.fixture
    def sample_news_content(self):
        """Create sample news content message"""
        return {
            "title": "Market Rally Continues",
            "content": "The stock market continued its bullish rally today with strong gains across technology stocks. Investors showed confidence in the growth outlook.",
            "source": "MarketWatch",
            "published_at": datetime.now().isoformat()
        }

    @pytest.fixture
    def sample_social_content(self):
        """Create sample social media content"""
        return {
            "text": "Feeling very optimistic about $AAPL earnings! Strong growth expected 🚀",
            "source": "twitter",
            "timestamp": datetime.now().isoformat()
        }

    def test_config_initialization(self, config):
        """Test configuration initialization"""
        assert config.agent_name == "test-sentiment-agent"
        assert config.sentiment_window_minutes == 15
        assert config.confidence_threshold == 0.6
        assert config.use_lexicon_analysis is True
        assert config.track_market_emotions is True

    def test_agent_initialization(self, sentiment_agent):
        """Test agent initialization"""
        assert sentiment_agent.config.agent_name == "test-sentiment-agent"
        assert len(sentiment_agent.sentiment_lexicon) > 0
        assert len(sentiment_agent.emotion_lexicon) > 0
        assert len(sentiment_agent.market_lexicon) > 0

    def test_sentiment_lexicon_loaded(self, sentiment_agent):
        """Test that sentiment lexicon is properly loaded"""
        lexicon = sentiment_agent.sentiment_lexicon
        
        # Test positive words
        assert "excellent" in lexicon
        assert lexicon["excellent"] > 0.5
        assert "bullish" in lexicon
        assert lexicon["bullish"] > 0.5
        
        # Test negative words
        assert "terrible" in lexicon
        assert lexicon["terrible"] < -0.5
        assert "bearish" in lexicon
        assert lexicon["bearish"] < -0.5
        
        # Test neutral words
        assert "neutral" in lexicon
        assert abs(lexicon["neutral"]) < 0.2

    def test_emotion_lexicon_loaded(self, sentiment_agent):
        """Test that emotion lexicon is properly loaded"""
        emotions = sentiment_agent.emotion_lexicon
        
        assert "fear" in emotions
        assert "confidence" in emotions
        assert "greed" in emotions
        
        # Check that each emotion has associated words
        for emotion, words in emotions.items():
            assert len(words) > 0
            for word, score in words.items():
                assert 0 <= score <= 1

    def test_extract_text_content_news(self, sentiment_agent, sample_news_content):
        """Test extracting text from news content"""
        text = sentiment_agent._extract_text_content(sample_news_content, "news")
        
        assert "Market Rally Continues" in text
        assert "bullish rally" in text
        assert len(text) > 10

    def test_extract_text_content_social(self, sentiment_agent, sample_social_content):
        """Test extracting text from social content"""
        text = sentiment_agent._extract_text_content(sample_social_content, "social")
        
        assert "optimistic about $AAPL" in text
        assert "Strong growth expected" in text

    def test_generate_content_id(self, sentiment_agent, sample_news_content):
        """Test content ID generation"""
        content_id = sentiment_agent._generate_content_id(sample_news_content, "news")
        
        assert content_id is not None
        assert len(content_id) == 16
        
        # Same content should generate same ID
        content_id2 = sentiment_agent._generate_content_id(sample_news_content, "news")
        assert content_id == content_id2

    def test_lexicon_sentiment_analysis_positive(self, sentiment_agent):
        """Test lexicon-based sentiment analysis for positive content"""
        positive_text = "excellent bullish strong growth outstanding performance"
        sentiment, confidence = sentiment_agent._lexicon_sentiment_analysis(positive_text)
        
        assert sentiment > 0.3
        assert confidence > 0.1

    def test_lexicon_sentiment_analysis_negative(self, sentiment_agent):
        """Test lexicon-based sentiment analysis for negative content"""
        negative_text = "terrible bearish weak decline poor performance crash"
        sentiment, confidence = sentiment_agent._lexicon_sentiment_analysis(negative_text)
        
        assert sentiment < -0.3
        assert confidence > 0.1

    def test_lexicon_sentiment_analysis_neutral(self, sentiment_agent):
        """Test lexicon-based sentiment analysis for neutral content"""
        neutral_text = "the company maintains stable operations with steady performance"
        sentiment, confidence = sentiment_agent._lexicon_sentiment_analysis(neutral_text)
        
        assert abs(sentiment) < 0.3
        assert confidence >= 0.1

    def test_analyze_emotions(self, sentiment_agent):
        """Test emotion analysis"""
        # Fear-inducing text
        fear_text = "worried anxious panic terrified scared market crash"
        emotions = sentiment_agent._analyze_emotions(fear_text)
        
        assert "fear" in emotions
        assert emotions["fear"] > 0.3
        
        # Confidence-inducing text
        confidence_text = "confident optimistic bullish assured positive outlook"
        emotions = sentiment_agent._analyze_emotions(confidence_text)
        
        assert "confidence" in emotions
        assert emotions["confidence"] > 0.3

    def test_calculate_market_relevance(self, sentiment_agent):
        """Test market relevance calculation"""
        # High relevance text
        high_relevance = "stock market trading investment earnings revenue financial"
        relevance = sentiment_agent._calculate_market_relevance(high_relevance)
        assert relevance > 0.3
        
        # Low relevance text
        low_relevance = "weather forecast sunny tomorrow rain"
        relevance = sentiment_agent._calculate_market_relevance(low_relevance)
        assert relevance < 0.2

    def test_extract_symbols(self, sentiment_agent):
        """Test stock symbol extraction"""
        text = "Looking at AAPL and MSFT for the quarterly earnings. GOOGL also interesting."
        symbols = sentiment_agent._extract_symbols(text)
        
        assert "AAPL" in symbols
        assert "MSFT" in symbols
        assert "GOOGL" in symbols
        assert len(symbols) == 3

    def test_extract_symbols_filters_common_words(self, sentiment_agent):
        """Test that common words are filtered from symbol extraction"""
        text = "THE market AND stocks ARE good BUT not ALL stocks CAN perform well"
        symbols = sentiment_agent._extract_symbols(text)
        
        # Common words should be filtered out
        common_words = {"THE", "AND", "ARE", "BUT", "ALL", "CAN"}
        for word in common_words:
            assert word not in symbols

    def test_analyze_symbol_sentiments(self, sentiment_agent):
        """Test symbol-specific sentiment analysis"""
        text = "aapl shows excellent growth while msft reports poor performance"
        symbols = ["AAPL", "MSFT"]
        
        symbol_sentiments = sentiment_agent._analyze_symbol_sentiments(text, symbols)
        
        # AAPL should have positive sentiment, MSFT negative
        if "AAPL" in symbol_sentiments and "MSFT" in symbol_sentiments:
            assert symbol_sentiments["AAPL"] > symbol_sentiments["MSFT"]

    def test_calculate_urgency(self, sentiment_agent):
        """Test urgency calculation"""
        # High urgency text
        urgent_text = "urgent breaking alert immediate emergency critical"
        urgency = sentiment_agent._calculate_urgency(urgent_text)
        assert urgency > 0.5
        
        # Low urgency text
        normal_text = "regular business operations continue as planned"
        urgency = sentiment_agent._calculate_urgency(normal_text)
        assert urgency < 0.3

    def test_calculate_volatility_indicator(self, sentiment_agent):
        """Test volatility indicator calculation"""
        # High volatility text
        volatile_text = "volatile swing fluctuate unstable erratic unpredictable"
        volatility = sentiment_agent._calculate_volatility_indicator(volatile_text)
        assert volatility > 0.5
        
        # Low volatility text
        stable_text = "steady consistent reliable stable predictable"
        volatility = sentiment_agent._calculate_volatility_indicator(stable_text)
        assert volatility < 0.3

    def test_get_source_credibility(self, sentiment_agent):
        """Test source credibility scoring"""
        # High credibility sources
        assert sentiment_agent._get_source_credibility("Reuters") >= 0.9
        assert sentiment_agent._get_source_credibility("Bloomberg") >= 0.9
        
        # Medium credibility sources
        assert sentiment_agent._get_source_credibility("CNN") >= 0.7
        assert sentiment_agent._get_source_credibility("BBC") >= 0.7
        
        # Low credibility sources
        assert sentiment_agent._get_source_credibility("Twitter") <= 0.4
        assert sentiment_agent._get_source_credibility("Reddit") <= 0.5
        
        # Unknown source
        assert sentiment_agent._get_source_credibility("UnknownSource") == 0.50

    def test_score_to_label(self, sentiment_agent):
        """Test sentiment score to label conversion"""
        assert sentiment_agent._score_to_label(0.8) == SentimentLabel.VERY_POSITIVE
        assert sentiment_agent._score_to_label(0.4) == SentimentLabel.POSITIVE
        assert sentiment_agent._score_to_label(0.0) == SentimentLabel.NEUTRAL
        assert sentiment_agent._score_to_label(-0.4) == SentimentLabel.NEGATIVE
        assert sentiment_agent._score_to_label(-0.8) == SentimentLabel.VERY_NEGATIVE

    @pytest.mark.asyncio
    async def test_analyze_sentiment_complete(self, sentiment_agent, sample_news_content):
        """Test complete sentiment analysis"""
        text = sentiment_agent._extract_text_content(sample_news_content, "news")
        content_id = sentiment_agent._generate_content_id(sample_news_content, "news")
        
        analysis = await sentiment_agent._analyze_sentiment(
            text, sample_news_content, "news", content_id
        )
        
        assert analysis is not None
        assert analysis.content_id == content_id
        assert analysis.content_type == "news"
        assert analysis.source == "MarketWatch"
        assert -1 <= analysis.sentiment_score <= 1
        assert 0 <= analysis.confidence <= 1
        assert analysis.sentiment_label in [label for label in SentimentLabel]
        assert analysis.primary_emotion in [emotion for emotion in EmotionLabel]
        assert 0 <= analysis.market_relevance <= 1
        assert 0 <= analysis.urgency_score <= 1
        assert 0 <= analysis.volatility_indicator <= 1
        assert 0 <= analysis.source_credibility <= 1

    def test_update_sentiment_windows(self, sentiment_agent):
        """Test sentiment window updates"""
        analysis = SentimentAnalysis(
            content_id="test123",
            content_type="news",
            source="TestSource",
            timestamp=datetime.now(),
            sentiment_score=0.6,
            sentiment_label=SentimentLabel.POSITIVE,
            confidence=0.8,
            primary_emotion=EmotionLabel.CONFIDENCE,
            emotion_scores={"confidence": 0.7, "neutral": 0.3},
            market_relevance=0.8,
            urgency_score=0.3,
            volatility_indicator=0.2,
            mentioned_symbols=["AAPL"],
            symbol_sentiments={"AAPL": 0.6},
            source_credibility=0.9,
            analysis_method="lexicon"
        )
        
        sentiment_agent._update_sentiment_windows(analysis)
        
        # Check that windows were updated
        assert len(sentiment_agent.sentiment_windows["market"]) == 1
        assert len(sentiment_agent.sentiment_windows["AAPL"]) == 1
        
        market_data = sentiment_agent.sentiment_windows["market"][0]
        assert market_data["sentiment"] == 0.6
        assert market_data["confidence"] == 0.8

    def test_calculate_trend(self, sentiment_agent):
        """Test trend calculation"""
        # Create a window with trending data
        window = deque()
        base_time = datetime.now()
        
        # Add data points showing upward trend
        for i in range(10):
            window.append({
                "timestamp": base_time + timedelta(minutes=i),
                "sentiment": 0.1 + (i * 0.1),  # Increasing sentiment
                "confidence": 0.8
            })
        
        trend = sentiment_agent._calculate_trend("AAPL", window)
        
        assert trend is not None
        assert trend.symbol == "AAPL"
        assert trend.sentiment_change > 0  # Should show positive change
        assert trend.trend_direction == "bullish"
        assert trend.sample_count == 10

    @pytest.mark.asyncio
    async def test_handle_content(self, sentiment_agent, sample_news_content):
        """Test handling content message"""
        # Mock the publish method
        sentiment_agent.publish_to_topic = AsyncMock()
        
        # Create message data
        message_data = json.dumps(sample_news_content).encode()
        
        # Handle the message
        await sentiment_agent._handle_content(message_data, "news")
        
        # Verify processing
        assert sentiment_agent.content_processed == 1
        assert len(sentiment_agent.sentiment_history) == 1

    @pytest.mark.asyncio
    async def test_get_agent_status(self, sentiment_agent):
        """Test agent status reporting"""
        status = await sentiment_agent.get_agent_status()
        
        assert "content_processed" in status
        assert "insights_published" in status
        assert "trends_detected" in status
        assert "sentiment_history_size" in status
        assert "tracked_symbols" in status
        assert "config" in status
        assert "confidence_threshold" in status["config"]
        assert "use_ml_models" in status["config"]


@pytest.mark.skipif(not DEPENDENCIES_AVAILABLE, reason="Agent dependencies not available")
class TestSentimentAnalysis:
    """Test cases for SentimentAnalysis data class"""

    def test_sentiment_analysis_creation(self):
        """Test creating SentimentAnalysis instance"""
        analysis = SentimentAnalysis(
            content_id="test123",
            content_type="news",
            source="TestSource",
            timestamp=datetime.now(),
            sentiment_score=0.6,
            sentiment_label=SentimentLabel.POSITIVE,
            confidence=0.8,
            primary_emotion=EmotionLabel.CONFIDENCE,
            emotion_scores={"confidence": 0.7},
            market_relevance=0.8,
            urgency_score=0.3,
            volatility_indicator=0.2,
            mentioned_symbols=["AAPL"],
            symbol_sentiments={"AAPL": 0.6},
            source_credibility=0.9,
            analysis_method="lexicon"
        )
        
        assert analysis.content_id == "test123"
        assert analysis.sentiment_score == 0.6
        assert analysis.sentiment_label == SentimentLabel.POSITIVE
        assert analysis.primary_emotion == EmotionLabel.CONFIDENCE

    def test_sentiment_analysis_to_dict(self):
        """Test converting SentimentAnalysis to dictionary"""
        analysis = SentimentAnalysis(
            content_id="test123",
            content_type="news",
            source="TestSource",
            timestamp=datetime(2023, 1, 1, 12, 0, 0),
            sentiment_score=0.6,
            sentiment_label=SentimentLabel.POSITIVE,
            confidence=0.8,
            primary_emotion=EmotionLabel.CONFIDENCE,
            emotion_scores={"confidence": 0.7},
            market_relevance=0.8,
            urgency_score=0.3,
            volatility_indicator=0.2,
            mentioned_symbols=["AAPL"],
            symbol_sentiments={"AAPL": 0.6},
            source_credibility=0.9,
            analysis_method="lexicon"
        )
        
        data = analysis.to_dict()
        
        assert data["content_id"] == "test123"
        assert data["sentiment_score"] == 0.6
        assert data["sentiment_label"] == "positive"
        assert data["primary_emotion"] == "confidence"
        assert isinstance(data["timestamp"], str)


@pytest.mark.skipif(not DEPENDENCIES_AVAILABLE, reason="Agent dependencies not available")
class TestSentimentTrend:
    """Test cases for SentimentTrend data class"""

    def test_sentiment_trend_creation(self):
        """Test creating SentimentTrend instance"""
        start_time = datetime.now()
        end_time = start_time + timedelta(hours=1)
        
        trend = SentimentTrend(
            symbol="AAPL",
            timeframe="1h",
            start_time=start_time,
            end_time=end_time,
            initial_sentiment=0.2,
            final_sentiment=0.8,
            sentiment_change=0.6,
            trend_direction="bullish",
            mean_sentiment=0.5,
            sentiment_volatility=0.1,
            sample_count=20,
            confidence=0.9
        )
        
        assert trend.symbol == "AAPL"
        assert trend.sentiment_change == 0.6
        assert trend.trend_direction == "bullish"

    def test_sentiment_trend_to_dict(self):
        """Test converting SentimentTrend to dictionary"""
        start_time = datetime(2023, 1, 1, 12, 0, 0)
        end_time = datetime(2023, 1, 1, 13, 0, 0)
        
        trend = SentimentTrend(
            symbol="AAPL",
            timeframe="1h",
            start_time=start_time,
            end_time=end_time,
            initial_sentiment=0.2,
            final_sentiment=0.8,
            sentiment_change=0.6,
            trend_direction="bullish",
            mean_sentiment=0.5,
            sentiment_volatility=0.1,
            sample_count=20,
            confidence=0.9
        )
        
        data = trend.to_dict()
        
        assert data["symbol"] == "AAPL"
        assert data["sentiment_change"] == 0.6
        assert data["trend_direction"] == "bullish"
        assert isinstance(data["start_time"], str)
        assert isinstance(data["end_time"], str)