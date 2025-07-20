"""
Tests for News Analysis Agent

Tests the news article processing and insight generation capabilities
of the News Analysis Agent.
"""

import asyncio
import json
import pytest
from datetime import datetime, timedelta
from unittest.mock import Mock, AsyncMock

# Import test dependencies
try:
    from agents.news_analysis.agent import NewsAnalysisAgent, NewsConfig, NewsArticle, NewsInsight
    from shared.base_agent import AgentConfig
    DEPENDENCIES_AVAILABLE = True
except ImportError:
    DEPENDENCIES_AVAILABLE = False


@pytest.mark.skipif(not DEPENDENCIES_AVAILABLE, reason="Agent dependencies not available")
class TestNewsAnalysisAgent:
    """Test cases for News Analysis Agent"""

    @pytest.fixture
    def config(self):
        """Create test configuration"""
        return NewsConfig(
            agent_name="test-news-agent",
            nats_url="nats://localhost:4222",
            log_level="DEBUG",
            relevance_threshold=0.3,
            impact_threshold=0.5,
            min_article_length=50,
            max_article_age_hours=24,
            tracked_symbols=["AAPL", "MSFT", "GOOGL"],
            tracked_sectors=["technology", "finance"]
        )

    @pytest.fixture
    def news_agent(self, config):
        """Create News Analysis Agent instance"""
        agent = NewsAnalysisAgent(config)
        agent.nats_client = Mock()
        agent.logger = Mock()
        return agent

    @pytest.fixture
    def sample_news_message(self):
        """Create sample news message"""
        return {
            "title": "Apple Reports Strong Q4 Earnings Beat",
            "content": "Apple Inc. reported quarterly earnings that beat analyst expectations, driven by strong iPhone sales and services revenue growth. The technology giant saw revenue increase 8% year-over-year.",
            "source": "Reuters",
            "url": "https://example.com/apple-earnings",
            "published_at": datetime.now().isoformat(),
            "symbol": "AAPL"
        }

    def test_config_initialization(self, config):
        """Test configuration initialization"""
        assert config.agent_name == "test-news-agent"
        assert config.relevance_threshold == 0.3
        assert config.impact_threshold == 0.5
        assert "AAPL" in config.tracked_symbols
        assert "technology" in config.tracked_sectors

    def test_agent_initialization(self, news_agent):
        """Test agent initialization"""
        assert news_agent.config.agent_name == "test-news-agent"
        assert len(news_agent.market_keywords) > 0
        assert "bullish" in news_agent.market_keywords
        assert "bearish" in news_agent.market_keywords
        assert len(news_agent.source_credibility) > 0

    def test_parse_news_article_valid(self, news_agent, sample_news_message):
        """Test parsing valid news article"""
        article = news_agent._parse_news_article(sample_news_message)
        
        assert article is not None
        assert article.title == "Apple Reports Strong Q4 Earnings Beat"
        assert article.source == "Reuters"
        assert article.symbol == "AAPL"
        assert len(article.content) >= news_agent.config.min_article_length

    def test_parse_news_article_invalid(self, news_agent):
        """Test parsing invalid news article"""
        # Missing required fields
        invalid_message = {
            "title": "Test Title"
            # Missing content, source, published_at
        }
        
        article = news_agent._parse_news_article(invalid_message)
        assert article is None

    def test_parse_news_article_too_old(self, news_agent):
        """Test parsing article that's too old"""
        old_message = {
            "title": "Old News",
            "content": "This is old news content that should be filtered out.",
            "source": "TestSource",
            "published_at": (datetime.now() - timedelta(hours=48)).isoformat()
        }
        
        article = news_agent._parse_news_article(old_message)
        assert article is None

    def test_parse_news_article_too_short(self, news_agent):
        """Test parsing article that's too short"""
        short_message = {
            "title": "Short",
            "content": "Short",  # Too short
            "source": "TestSource",
            "published_at": datetime.now().isoformat()
        }
        
        article = news_agent._parse_news_article(short_message)
        assert article is None

    def test_generate_article_id(self, news_agent, sample_news_message):
        """Test article ID generation"""
        article = news_agent._parse_news_article(sample_news_message)
        article_id = news_agent._generate_article_id(article)
        
        assert article_id is not None
        assert len(article_id) == 16  # MD5 hash truncated to 16 chars
        
        # Same article should generate same ID
        article_id2 = news_agent._generate_article_id(article)
        assert article_id == article_id2

    def test_calculate_relevance_score(self, news_agent):
        """Test relevance score calculation"""
        # High relevance text with tracked symbols and market keywords
        high_relevance_text = "apple earnings beat expectations revenue growth technology stock"
        score = news_agent._calculate_relevance_score(high_relevance_text)
        assert score > 0.5
        
        # Low relevance text
        low_relevance_text = "weather forecast sunny tomorrow no clouds"
        score = news_agent._calculate_relevance_score(low_relevance_text)
        assert score < 0.3

    def test_calculate_impact_score(self, news_agent):
        """Test impact score calculation"""
        # High impact text
        high_impact_text = "fed interest rate earnings guidance merger breaking"
        score = news_agent._calculate_impact_score(high_impact_text, "Reuters")
        assert score > 0.3
        
        # Low impact text
        low_impact_text = "regular business operations continue normally"
        score = news_agent._calculate_impact_score(low_impact_text, "Unknown")
        assert score < 0.5

    def test_calculate_sentiment_score(self, news_agent):
        """Test sentiment score calculation"""
        # Positive sentiment
        positive_text = "excellent growth strong performance good results"
        score = news_agent._calculate_sentiment_score(positive_text)
        assert score > 0
        
        # Negative sentiment
        negative_text = "poor performance weak results bad earnings decline"
        score = news_agent._calculate_sentiment_score(negative_text)
        assert score < 0
        
        # Neutral sentiment
        neutral_text = "company maintains stable operations"
        score = news_agent._calculate_sentiment_score(neutral_text)
        assert abs(score) < 0.3

    def test_extract_mentioned_symbols(self, news_agent):
        """Test symbol extraction"""
        text = "aapl reported earnings while msft announced partnership"
        symbols = news_agent._extract_mentioned_symbols(text)
        
        assert "AAPL" in symbols
        assert "MSFT" in symbols

    def test_extract_mentioned_sectors(self, news_agent):
        """Test sector extraction"""
        text = "technology sector shows growth while finance remains stable"
        sectors = news_agent._extract_mentioned_sectors(text)
        
        assert "technology" in sectors
        assert "finance" in sectors

    def test_extract_key_topics(self, news_agent):
        """Test key topic extraction"""
        text = "company reports earnings beat with revenue growth and merger talks"
        topics = news_agent._extract_key_topics(text)
        
        assert "earnings" in topics
        assert "mergers" in topics

    def test_get_source_credibility(self, news_agent):
        """Test source credibility scoring"""
        # High credibility source
        reuters_score = news_agent._get_source_credibility("Reuters")
        assert reuters_score >= 0.9
        
        # Unknown source
        unknown_score = news_agent._get_source_credibility("UnknownSource")
        assert unknown_score == 0.50

    def test_calculate_urgency_level(self, news_agent):
        """Test urgency level calculation"""
        # Critical urgency
        critical_level = news_agent._calculate_urgency_level(0.9, 0.9, 0.8)
        assert critical_level == "critical"
        
        # Low urgency
        low_level = news_agent._calculate_urgency_level(0.2, 0.2, 0.1)
        assert low_level == "low"

    @pytest.mark.asyncio
    async def test_analyze_article(self, news_agent, sample_news_message):
        """Test complete article analysis"""
        article = news_agent._parse_news_article(sample_news_message)
        article_id = news_agent._generate_article_id(article)
        
        insight = await news_agent._analyze_article(article, article_id)
        
        assert insight is not None
        assert insight.article_id == article_id
        assert insight.title == article.title
        assert insight.source == article.source
        assert 0 <= insight.relevance_score <= 1
        assert -1 <= insight.sentiment_score <= 1
        assert 0 <= insight.source_credibility <= 1
        assert insight.urgency_level in ["low", "medium", "high", "critical"]

    @pytest.mark.asyncio
    async def test_handle_news_article(self, news_agent, sample_news_message):
        """Test handling news article message"""
        # Mock the publish method
        news_agent.publish_to_topic = AsyncMock()
        
        # Create message data
        message_data = json.dumps(sample_news_message).encode()
        
        # Handle the message
        await news_agent._handle_news_article("test.topic", message_data)
        
        # Verify processing
        assert news_agent.articles_processed == 1
        assert len(news_agent.news_history) == 1

    @pytest.mark.asyncio
    async def test_get_agent_status(self, news_agent):
        """Test agent status reporting"""
        status = await news_agent.get_agent_status()
        
        assert "articles_processed" in status
        assert "insights_published" in status
        assert "history_size" in status
        assert "config" in status
        assert "relevance_threshold" in status["config"]
        assert "tracked_symbols" in status["config"]

    def test_market_keywords_loaded(self, news_agent):
        """Test that market keywords are properly loaded"""
        keywords = news_agent.market_keywords
        
        assert "bullish" in keywords
        assert "bearish" in keywords
        assert "volatility" in keywords
        assert "company_events" in keywords
        assert "market_events" in keywords
        
        # Check that each category has keywords
        for category, words in keywords.items():
            assert len(words) > 0
            assert all(isinstance(word, str) for word in words)

    def test_source_credibility_loaded(self, news_agent):
        """Test that source credibility scores are loaded"""
        credibility = news_agent.source_credibility
        
        assert "reuters" in credibility
        assert "bloomberg" in credibility
        assert "unknown" in credibility
        
        # Check that all scores are between 0 and 1
        for source, score in credibility.items():
            assert 0 <= score <= 1


@pytest.mark.skipif(not DEPENDENCIES_AVAILABLE, reason="Agent dependencies not available") 
class TestNewsInsight:
    """Test cases for NewsInsight data class"""

    def test_news_insight_creation(self):
        """Test creating NewsInsight instance"""
        insight = NewsInsight(
            article_id="test123",
            title="Test Article",
            source="TestSource",
            published_at=datetime.now(),
            relevance_score=0.8,
            impact_score=0.6,
            sentiment_score=0.4,
            urgency_level="high",
            mentioned_symbols=["AAPL"],
            mentioned_sectors=["technology"],
            key_topics=["earnings"],
            extracted_entities={"companies": ["Apple Inc."]},
            source_credibility=0.9,
            article_quality=0.8
        )
        
        assert insight.article_id == "test123"
        assert insight.relevance_score == 0.8
        assert "AAPL" in insight.mentioned_symbols

    def test_news_insight_to_dict(self):
        """Test converting NewsInsight to dictionary"""
        insight = NewsInsight(
            article_id="test123",
            title="Test Article", 
            source="TestSource",
            published_at=datetime(2023, 1, 1, 12, 0, 0),
            relevance_score=0.8,
            impact_score=0.6,
            sentiment_score=0.4,
            urgency_level="high",
            mentioned_symbols=["AAPL"],
            mentioned_sectors=["technology"],
            key_topics=["earnings"],
            extracted_entities={"companies": ["Apple Inc."]},
            source_credibility=0.9,
            article_quality=0.8
        )
        
        data = insight.to_dict()
        
        assert data["article_id"] == "test123"
        assert data["relevance_score"] == 0.8
        assert isinstance(data["published_at"], str)  # Should be ISO format
        assert "AAPL" in data["mentioned_symbols"]