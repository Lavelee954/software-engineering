"""
Sentiment Analysis Agent Implementation

Performs sophisticated sentiment analysis using multiple techniques:
- Lexicon-based sentiment scoring
- Machine learning models (when available)
- Emotion detection
- Sentiment trend analysis
- Market-specific sentiment indicators
- Advanced AI framework integration with LangGraph and A2A communication
"""

import asyncio
import json
import math
import statistics
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple, Any
from collections import defaultdict, deque
from enum import Enum

from shared.base_agent import BaseAgent, AgentConfig

# AI Framework Integration
try:
    from .ai_framework import LangGraphSentimentAnalyzer
    from shared.a2a_communication import A2ACommunicationManager
    AI_FRAMEWORK_AVAILABLE = True
except ImportError:
    AI_FRAMEWORK_AVAILABLE = False


class SentimentLabel(Enum):
    """Sentiment classification labels"""
    VERY_NEGATIVE = "very_negative"
    NEGATIVE = "negative"
    NEUTRAL = "neutral"
    POSITIVE = "positive"
    VERY_POSITIVE = "very_positive"


class EmotionLabel(Enum):
    """Emotion classification labels"""
    FEAR = "fear"
    GREED = "greed"
    ANXIETY = "anxiety"
    CONFIDENCE = "confidence"
    EUPHORIA = "euphoria"
    PANIC = "panic"
    NEUTRAL = "neutral"


@dataclass
class SentimentConfig(AgentConfig):
    """Configuration for Sentiment Analysis Agent"""
    
    # Subscription topics
    news_topic: str = "raw.news.article"
    social_topic: str = "raw.social.post"
    
    # Publishing topics
    sentiment_insights_topic: str = "insight.sentiment"
    
    # Analysis parameters
    sentiment_window_minutes: int = 15
    trend_analysis_hours: int = 24
    max_history_items: int = 2000
    
    # Model settings
    use_lexicon_analysis: bool = True
    use_ml_models: bool = False  # Set to True when models are available
    confidence_threshold: float = 0.6
    
    # Content filtering
    min_content_length: int = 10
    max_content_age_hours: int = 48
    
    # Market sentiment specific
    track_market_emotions: bool = True
    weight_source_credibility: bool = True
    aggregate_by_symbol: bool = True
    
    # Trend analysis
    detect_sentiment_shifts: bool = True
    shift_threshold: float = 0.3
    min_samples_for_trend: int = 10
    
    # AI Framework Configuration
    enable_langgraph: bool = True
    enable_a2a_communication: bool = True
    langgraph_reasoning_depth: int = 8
    a2a_peer_validation: bool = True
    emotional_consensus_threshold: float = 0.75
    sentiment_cross_validation: bool = True


@dataclass
class SentimentAnalysis:
    """Represents sentiment analysis results"""
    content_id: str
    content_type: str  # "news" or "social"
    source: str
    timestamp: datetime
    
    # Core sentiment
    sentiment_score: float  # -1.0 to 1.0
    sentiment_label: SentimentLabel
    confidence: float  # 0.0 to 1.0
    
    # Emotion analysis
    primary_emotion: EmotionLabel
    emotion_scores: Dict[str, float]
    
    # Market specific
    market_relevance: float
    urgency_score: float
    volatility_indicator: float
    
    # Symbol/sector specific
    mentioned_symbols: List[str]
    symbol_sentiments: Dict[str, float]
    
    # Metadata
    source_credibility: float
    analysis_method: str  # "lexicon", "ml", "hybrid"
    
    def to_dict(self) -> Dict[str, Any]:
        data = asdict(self)
        data['timestamp'] = self.timestamp.isoformat()
        data['sentiment_label'] = self.sentiment_label.value
        data['primary_emotion'] = self.primary_emotion.value
        return data


@dataclass
class SentimentTrend:
    """Represents sentiment trend over time"""
    symbol: Optional[str]
    timeframe: str  # "15m", "1h", "4h", "1d"
    start_time: datetime
    end_time: datetime
    
    # Trend metrics
    initial_sentiment: float
    final_sentiment: float
    sentiment_change: float
    trend_direction: str  # "bullish", "bearish", "sideways"
    
    # Statistics
    mean_sentiment: float
    sentiment_volatility: float
    sample_count: int
    confidence: float
    
    def to_dict(self) -> Dict[str, Any]:
        data = asdict(self)
        data['start_time'] = self.start_time.isoformat()
        data['end_time'] = self.end_time.isoformat()
        return data


class SentimentAnalysisAgent(BaseAgent):
    """
    Sentiment Analysis Agent
    
    Performs advanced sentiment analysis on textual content following CLAUDE.md:
    - Subscribes to raw.news.article and raw.social.post messages  
    - Analyzes sentiment using multiple techniques
    - Tracks sentiment trends and shifts
    - Publishes insights to insight.sentiment topic
    """
    
    def __init__(self, config: SentimentConfig):
        super().__init__(config)
        self.config = config
        
        # Analysis state
        self.sentiment_history: deque = deque(maxlen=config.max_history_items)
        self.processed_content: set = set()
        
        # Trend analysis
        self.sentiment_windows: Dict[str, deque] = defaultdict(lambda: deque(maxlen=100))
        
        # Lexicon components
        self.sentiment_lexicon = self._load_sentiment_lexicon()
        self.emotion_lexicon = self._load_emotion_lexicon()
        self.market_lexicon = self._load_market_lexicon()
        
        # Statistics
        self.content_processed = 0
        self.insights_published = 0
        self.trends_detected = 0
        
        # AI Framework Integration
        self.ai_analyzer = None
        self.a2a_manager = None
        self._initialize_ai_framework()
        
        self.logger.info("Sentiment Analysis Agent initialized", extra={
            "sentiment_window": config.sentiment_window_minutes,
            "trend_hours": config.trend_analysis_hours,
            "use_ml": config.use_ml_models,
            "track_emotions": config.track_market_emotions,
            "ai_framework_enabled": AI_FRAMEWORK_AVAILABLE,
            "langgraph_enabled": config.enable_langgraph,
            "a2a_enabled": config.enable_a2a_communication
        })
    
    def _initialize_ai_framework(self):
        """Initialize AI framework components"""
        if not AI_FRAMEWORK_AVAILABLE:
            self.logger.warning("AI framework not available - using traditional analysis only")
            return
        
        try:
            # Initialize LangGraph analyzer
            if self.config.enable_langgraph:
                self.ai_analyzer = LangGraphSentimentAnalyzer(
                    agent_id=self.config.agent_name,
                    config={
                        "reasoning_depth": self.config.langgraph_reasoning_depth,
                        "emotional_consensus_threshold": self.config.emotional_consensus_threshold,
                        "cross_validation": self.config.sentiment_cross_validation
                    }
                )
                self.logger.info("LangGraph sentiment analyzer initialized")
            
            # Initialize A2A communication
            if self.config.enable_a2a_communication:
                self.a2a_manager = A2ACommunicationManager()
                # Register this agent in the A2A system
                from shared.a2a_communication import AgentProfile, AgentCapability, CollaborationPattern
                
                profile = AgentProfile(
                    agent_id=self.config.agent_name,
                    agent_type="sentiment_analysis",
                    capabilities=[AgentCapability.SENTIMENT_ANALYSIS],
                    service_level=0.9,
                    reputation_score=0.85,
                    load_factor=0.0,
                    max_concurrent_requests=10,
                    supported_patterns=[
                        CollaborationPattern.PEER_REVIEW,
                        CollaborationPattern.CONSENSUS_BUILDING,
                        CollaborationPattern.KNOWLEDGE_SHARING
                    ],
                    metadata={
                        "emotional_analysis": True,
                        "sentiment_scoring": True,
                        "volatility_assessment": True
                    }
                )
                
                self.a2a_manager.registry.register_agent(profile)
                self.a2a_manager.router.register_handler(self.config.agent_name, self._handle_a2a_message)
                self.logger.info("A2A communication manager initialized")
                
        except Exception as e:
            self.logger.error(f"Failed to initialize AI framework: {e}")
            self.ai_analyzer = None
            self.a2a_manager = None
    
    def _load_sentiment_lexicon(self) -> Dict[str, float]:
        """Load sentiment lexicon with financial terms"""
        lexicon = {
            # Very positive (0.8 to 1.0)
            "excellent": 0.9, "outstanding": 0.9, "exceptional": 0.9,
            "bullish": 0.8, "rally": 0.8, "surge": 0.8, "soar": 0.8,
            "breakthrough": 0.8, "triumph": 0.8, "boom": 0.8,
            
            # Positive (0.3 to 0.7)
            "good": 0.5, "positive": 0.5, "strong": 0.6, "solid": 0.5,
            "growth": 0.6, "gain": 0.5, "rise": 0.4, "increase": 0.4,
            "profit": 0.6, "revenue": 0.3, "beat": 0.6, "outperform": 0.7,
            
            # Neutral (-0.2 to 0.2)
            "stable": 0.1, "steady": 0.1, "flat": 0.0, "unchanged": 0.0,
            "maintain": 0.0, "hold": 0.0, "neutral": 0.0,
            
            # Negative (-0.7 to -0.3)
            "weak": -0.5, "poor": -0.6, "bad": -0.5, "negative": -0.5,
            "decline": -0.4, "fall": -0.4, "drop": -0.5, "loss": -0.6,
            "miss": -0.6, "underperform": -0.7, "concern": -0.4,
            
            # Very negative (-1.0 to -0.8)
            "terrible": -0.9, "disaster": -0.9, "crash": -0.9,
            "bearish": -0.8, "plunge": -0.8, "collapse": -0.9,
            "crisis": -0.8, "panic": -0.9, "catastrophe": -0.9
        }
        return lexicon
    
    def _load_emotion_lexicon(self) -> Dict[str, Dict[str, float]]:
        """Load emotion lexicon for financial markets"""
        return {
            "fear": {
                "afraid": 0.8, "scared": 0.7, "worried": 0.6, "anxious": 0.7,
                "panic": 0.9, "terrified": 0.9, "concerned": 0.5, "nervous": 0.6
            },
            "greed": {
                "greedy": 0.8, "aggressive": 0.6, "ambitious": 0.5,
                "opportunistic": 0.6, "acquisitive": 0.7
            },
            "confidence": {
                "confident": 0.8, "optimistic": 0.7, "assured": 0.6,
                "certain": 0.6, "bullish": 0.8, "positive": 0.5
            },
            "anxiety": {
                "uncertain": 0.6, "volatile": 0.7, "unstable": 0.6,
                "risky": 0.7, "unpredictable": 0.8
            },
            "euphoria": {
                "euphoric": 0.9, "ecstatic": 0.8, "thrilled": 0.7,
                "excited": 0.6, "exuberant": 0.8
            }
        }
    
    def _load_market_lexicon(self) -> Dict[str, float]:
        """Load market-specific sentiment modifiers"""
        return {
            # Market sentiment amplifiers
            "breakout": 0.3, "breakdown": -0.3, "momentum": 0.2,
            "volatility": -0.1, "uncertainty": -0.3, "stability": 0.2,
            
            # Financial performance
            "earnings": 0.1, "revenue": 0.1, "guidance": 0.2,
            "outlook": 0.1, "forecast": 0.1, "estimate": 0.0,
            
            # Market events
            "ipo": 0.2, "merger": 0.3, "acquisition": 0.3,
            "dividend": 0.2, "buyback": 0.3, "split": 0.1,
            
            # Risk indicators  
            "risk": -0.2, "exposure": -0.1, "hedge": -0.1,
            "protection": 0.1, "safety": 0.2, "secure": 0.2
        }
    
    async def start(self):
        """Start the Sentiment Analysis Agent"""
        try:
            await super().start()
            
            # Start A2A communication system
            if self.a2a_manager:
                await self.a2a_manager.start()
                self.logger.info("A2A communication system started")
            
            # Subscribe to content sources
            await self.subscribe_to_topic(
                self.config.news_topic,
                self._handle_news_content
            )
            
            await self.subscribe_to_topic(
                self.config.social_topic,
                self._handle_social_content
            )
            
            # Start trend analysis task
            if self.config.detect_sentiment_shifts:
                asyncio.create_task(self._trend_analysis_loop())
            
            # Start peer collaboration tasks
            if self.a2a_manager and self.config.a2a_peer_validation:
                asyncio.create_task(self._peer_collaboration_loop())
            
            self.logger.info("Sentiment Analysis Agent started successfully")
            
        except Exception as e:
            self.logger.error(f"Failed to start Sentiment Analysis Agent: {e}")
            raise
    
    async def _handle_news_content(self, subject: str, data: bytes):
        """Handle news content for sentiment analysis"""
        await self._handle_content(data, "news")
    
    async def _handle_social_content(self, subject: str, data: bytes):
        """Handle social media content for sentiment analysis"""
        await self._handle_content(data, "social")
    
    async def _handle_content(self, data: bytes, content_type: str):
        """Handle content for sentiment analysis"""
        try:
            # Parse message
            message = json.loads(data.decode())
            self.logger.debug(f"Received {content_type} content for sentiment analysis")
            
            # Extract text content
            text_content = self._extract_text_content(message, content_type)
            if not text_content:
                return
            
            # Generate content ID
            content_id = self._generate_content_id(message, content_type)
            if content_id in self.processed_content:
                return
            
            # Perform sentiment analysis (with AI framework if available)
            analysis = await self._analyze_sentiment_enhanced(
                text_content, message, content_type, content_id
            )
            
            if analysis:
                # Store in history
                self.sentiment_history.append(analysis)
                self.processed_content.add(content_id)
                
                # Update sentiment windows for trend analysis
                self._update_sentiment_windows(analysis)
                
                # Publish high-confidence insights
                if analysis.confidence >= self.config.confidence_threshold:
                    await self._publish_sentiment_insight(analysis)
                    self.insights_published += 1
                
                self.content_processed += 1
                
                self.logger.info("Content analyzed", extra={
                    "content_id": content_id,
                    "type": content_type,
                    "sentiment": analysis.sentiment_score,
                    "confidence": analysis.confidence,
                    "emotion": analysis.primary_emotion.value
                })
        
        except Exception as e:
            self.logger.error(f"Error analyzing {content_type} content: {e}")
    
    def _extract_text_content(self, message: Dict[str, Any], content_type: str) -> Optional[str]:
        """Extract text content from message"""
        try:
            if content_type == "news":
                title = message.get('title', '')
                content = message.get('content', '')
                return f"{title} {content}".strip()
            
            elif content_type == "social":
                return message.get('text', '').strip()
            
            return None
            
        except Exception:
            return None
    
    def _generate_content_id(self, message: Dict[str, Any], content_type: str) -> str:
        """Generate unique content ID"""
        import hashlib
        
        if content_type == "news":
            content = f"{message.get('title', '')}_{message.get('source', '')}_{message.get('published_at', '')}"
        else:
            content = f"{message.get('text', '')}_{message.get('source', '')}_{message.get('timestamp', '')}"
        
        return hashlib.md5(content.encode()).hexdigest()[:16]
    
    async def _analyze_sentiment_enhanced(
        self, 
        text: str, 
        message: Dict[str, Any], 
        content_type: str, 
        content_id: str
    ) -> Optional[SentimentAnalysis]:
        """Enhanced sentiment analysis with AI framework integration"""
        try:
            # Use AI framework for enhanced analysis if available
            if self.ai_analyzer and self.config.enable_langgraph:
                return await self._ai_framework_analysis(text, message, content_type, content_id)
            else:
                # Fallback to traditional analysis
                return await self._analyze_sentiment(text, message, content_type, content_id)
                
        except Exception as e:
            self.logger.error(f"Error in enhanced sentiment analysis: {e}")
            # Fallback to traditional analysis
            return await self._analyze_sentiment(text, message, content_type, content_id)
    
    async def _ai_framework_analysis(
        self, 
        text: str, 
        message: Dict[str, Any], 
        content_type: str, 
        content_id: str
    ) -> Optional[SentimentAnalysis]:
        """Perform AI framework-enhanced sentiment analysis"""
        try:
            # Prepare metadata for AI analysis
            metadata = {
                "source": message.get('source', 'unknown'),
                "content_type": content_type,
                "published_at": message.get('published_at', message.get('timestamp')),
                "content_length": len(text),
                "content_id": content_id
            }
            
            # Process through LangGraph reasoning pipeline
            ai_result = await self.ai_analyzer.process_content_sentiment(text, metadata)
            
            # Convert AI framework result to SentimentAnalysis object
            if ai_result and "sentiment_analysis" in ai_result:
                sentiment_data = ai_result["sentiment_analysis"]
                
                # Extract key metrics from AI analysis
                overall_sentiment = sentiment_data.get("overall_sentiment", 0.0)
                confidence_level = sentiment_data.get("confidence_level", 0.5)
                emotional_summary = sentiment_data.get("emotional_summary", {})
                
                # Create enhanced SentimentAnalysis object
                analysis = SentimentAnalysis(
                    content_id=content_id,
                    content_type=content_type,
                    source=message.get('source', 'unknown'),
                    timestamp=datetime.now(),
                    sentiment_score=overall_sentiment,
                    sentiment_label=self._score_to_label(overall_sentiment),
                    confidence=confidence_level,
                    primary_emotion=EmotionLabel(emotional_summary.get("dominant_emotion", "neutral")),
                    emotion_scores=emotional_summary.get("emotion_scores", {"neutral": 1.0}),
                    market_relevance=sentiment_data.get("market_implications", {}).get("impact_score", 0.5),
                    urgency_score=self._map_urgency_to_score(sentiment_data.get("temporal_urgency", "medium")),
                    volatility_indicator=sentiment_data.get("volatility_forecast", {}).get("volatility_score", 0.5),
                    mentioned_symbols=self._extract_symbols(text),
                    symbol_sentiments={},  # Will be filled by traditional analysis
                    source_credibility=self._get_source_credibility(message.get('source', '')),
                    analysis_method="ai_framework"
                )
                
                # Enhance with traditional symbol analysis
                analysis.symbol_sentiments = self._analyze_symbol_sentiments(
                    text.lower(), analysis.mentioned_symbols
                )
                
                # Initiate A2A peer review if enabled
                if self.a2a_manager and self.config.a2a_peer_validation:
                    await self._initiate_peer_review(analysis, ai_result)
                
                self.logger.info("AI framework sentiment analysis completed", extra={
                    "content_id": content_id,
                    "sentiment_score": overall_sentiment,
                    "confidence": confidence_level,
                    "ai_reasoning_steps": len(ai_result.get("reasoning_trace", [])),
                    "a2a_communications": len(ai_result.get("agent_communications", []))
                })
                
                return analysis
            
            else:
                # AI framework failed, use traditional analysis
                self.logger.warning("AI framework analysis failed, falling back to traditional analysis")
                return await self._analyze_sentiment(text, message, content_type, content_id)
                
        except Exception as e:
            self.logger.error(f"Error in AI framework analysis: {e}")
            # Fallback to traditional analysis
            return await self._analyze_sentiment(text, message, content_type, content_id)
    
    def _map_urgency_to_score(self, urgency: str) -> float:
        """Map urgency level to numeric score"""
        urgency_map = {
            "immediate": 1.0,
            "high": 0.8,
            "medium": 0.5,
            "low": 0.2
        }
        return urgency_map.get(urgency, 0.5)
    
    async def _initiate_peer_review(self, analysis: SentimentAnalysis, ai_result: Dict[str, Any]):
        """Initiate peer review process through A2A communication"""
        try:
            if not self.a2a_manager:
                return
            
            # Create peer review request
            review_content = {
                "analysis_type": "sentiment_analysis",
                "content_id": analysis.content_id,
                "sentiment_score": analysis.sentiment_score,
                "confidence": analysis.confidence,
                "primary_emotion": analysis.primary_emotion.value,
                "market_relevance": analysis.market_relevance,
                "ai_reasoning_trace": ai_result.get("reasoning_trace", []),
                "request_validation": True
            }
            
            # Send to news analysis agent for cross-validation
            peer_agents = ["news_analysis_agent", "technical_analysis_agent"]
            review_criteria = ["sentiment_accuracy", "emotional_consistency", "market_relevance"]
            
            review_result = await self.a2a_manager.orchestrator.orchestrate_peer_review(
                subject_agent=self.config.agent_name,
                analysis_data=review_content,
                review_criteria=review_criteria,
                num_reviewers=min(2, len(peer_agents))
            )
            
            # Adjust confidence based on peer review
            if review_result.get("overall_score", 0) > 0.8:
                analysis.confidence = min(1.0, analysis.confidence * 1.1)
                self.logger.info("Peer review boosted confidence", extra={
                    "content_id": analysis.content_id,
                    "review_score": review_result.get("overall_score", 0),
                    "new_confidence": analysis.confidence
                })
            elif review_result.get("overall_score", 0) < 0.5:
                analysis.confidence = max(0.1, analysis.confidence * 0.9)
                self.logger.warning("Peer review reduced confidence", extra={
                    "content_id": analysis.content_id,
                    "review_score": review_result.get("overall_score", 0),
                    "new_confidence": analysis.confidence
                })
                
        except Exception as e:
            self.logger.error(f"Error in peer review initiation: {e}")
    
    async def _analyze_sentiment(
        self, 
        text: str, 
        message: Dict[str, Any], 
        content_type: str, 
        content_id: str
    ) -> Optional[SentimentAnalysis]:
        """Perform comprehensive sentiment analysis"""
        try:
            # Basic validation
            if len(text) < self.config.min_content_length:
                return None
            
            text_lower = text.lower()
            
            # Lexicon-based analysis
            sentiment_score, confidence = self._lexicon_sentiment_analysis(text_lower)
            
            # Emotion analysis
            emotion_scores = self._analyze_emotions(text_lower)
            primary_emotion = max(emotion_scores.items(), key=lambda x: x[1])
            
            # Market relevance
            market_relevance = self._calculate_market_relevance(text_lower)
            
            # Symbol extraction and sentiment
            mentioned_symbols = self._extract_symbols(text)
            symbol_sentiments = self._analyze_symbol_sentiments(text_lower, mentioned_symbols)
            
            # Calculate additional metrics
            urgency_score = self._calculate_urgency(text_lower)
            volatility_indicator = self._calculate_volatility_indicator(text_lower)
            
            # Source credibility
            source_credibility = self._get_source_credibility(message.get('source', ''))
            
            # Adjust confidence based on credibility
            adjusted_confidence = confidence * (0.5 + 0.5 * source_credibility)
            
            # Determine sentiment label
            sentiment_label = self._score_to_label(sentiment_score)
            
            return SentimentAnalysis(
                content_id=content_id,
                content_type=content_type,
                source=message.get('source', 'unknown'),
                timestamp=datetime.now(),
                sentiment_score=sentiment_score,
                sentiment_label=sentiment_label,
                confidence=adjusted_confidence,
                primary_emotion=EmotionLabel(primary_emotion[0]),
                emotion_scores=emotion_scores,
                market_relevance=market_relevance,
                urgency_score=urgency_score,
                volatility_indicator=volatility_indicator,
                mentioned_symbols=mentioned_symbols,
                symbol_sentiments=symbol_sentiments,
                source_credibility=source_credibility,
                analysis_method="lexicon"
            )
            
        except Exception as e:
            self.logger.error(f"Error in sentiment analysis: {e}")
            return None
    
    def _lexicon_sentiment_analysis(self, text: str) -> Tuple[float, float]:
        """Perform lexicon-based sentiment analysis"""
        sentiment_scores = []
        total_words = len(text.split())
        
        # Score individual words
        for word, score in self.sentiment_lexicon.items():
            if word in text:
                sentiment_scores.append(score)
        
        # Apply market-specific modifiers
        market_modifier = 0.0
        for term, modifier in self.market_lexicon.items():
            if term in text:
                market_modifier += modifier
        
        # Calculate overall sentiment
        if sentiment_scores:
            base_sentiment = statistics.mean(sentiment_scores)
            final_sentiment = base_sentiment + (market_modifier * 0.1)
            
            # Clamp to [-1, 1]
            final_sentiment = max(-1.0, min(1.0, final_sentiment))
            
            # Calculate confidence based on coverage
            coverage = len(sentiment_scores) / max(total_words, 1)
            confidence = min(0.9, 0.3 + coverage * 0.6)
            
            return final_sentiment, confidence
        
        return 0.0, 0.1  # Neutral with low confidence
    
    def _analyze_emotions(self, text: str) -> Dict[str, float]:
        """Analyze emotional content"""
        emotion_scores = {emotion.value: 0.0 for emotion in EmotionLabel}
        
        for emotion, words in self.emotion_lexicon.items():
            score = 0.0
            for word, weight in words.items():
                if word in text:
                    score += weight
            
            # Normalize score
            emotion_scores[emotion] = min(1.0, score / 3.0)
        
        # Set neutral if no emotions detected
        if all(score == 0.0 for score in emotion_scores.values()):
            emotion_scores["neutral"] = 1.0
        
        return emotion_scores
    
    def _calculate_market_relevance(self, text: str) -> float:
        """Calculate market relevance score"""
        market_terms = [
            "stock", "market", "trading", "investment", "portfolio",
            "earnings", "revenue", "profit", "loss", "price", "shares",
            "financial", "economic", "analyst", "rating", "target"
        ]
        
        relevance_score = 0.0
        for term in market_terms:
            if term in text:
                relevance_score += 0.1
        
        return min(1.0, relevance_score)
    
    def _extract_symbols(self, text: str) -> List[str]:
        """Extract stock symbols from text"""
        import re
        
        # Pattern for stock symbols (3-5 capital letters)
        symbol_pattern = r'\b[A-Z]{3,5}\b'
        potential_symbols = re.findall(symbol_pattern, text)
        
        # Filter out common words that match the pattern
        common_words = {"THE", "AND", "FOR", "ARE", "BUT", "NOT", "YOU", "ALL", "CAN", "HER", "WAS", "ONE", "OUR", "HAD", "HAS"}
        symbols = [s for s in potential_symbols if s not in common_words]
        
        return list(set(symbols))  # Remove duplicates
    
    def _analyze_symbol_sentiments(self, text: str, symbols: List[str]) -> Dict[str, float]:
        """Analyze sentiment for specific symbols"""
        symbol_sentiments = {}
        
        for symbol in symbols:
            # Find context around symbol mentions
            symbol_contexts = self._extract_symbol_context(text, symbol.lower())
            
            if symbol_contexts:
                context_sentiments = []
                for context in symbol_contexts:
                    sentiment, _ = self._lexicon_sentiment_analysis(context)
                    context_sentiments.append(sentiment)
                
                # Average sentiment for this symbol
                if context_sentiments:
                    symbol_sentiments[symbol] = statistics.mean(context_sentiments)
        
        return symbol_sentiments
    
    def _extract_symbol_context(self, text: str, symbol: str) -> List[str]:
        """Extract context around symbol mentions"""
        words = text.split()
        contexts = []
        
        for i, word in enumerate(words):
            if symbol in word:
                # Extract context window (5 words before and after)
                start = max(0, i - 5)
                end = min(len(words), i + 6)
                context = ' '.join(words[start:end])
                contexts.append(context)
        
        return contexts
    
    def _calculate_urgency(self, text: str) -> float:
        """Calculate urgency score"""
        urgency_terms = ["urgent", "breaking", "alert", "immediate", "emergency", "critical"]
        
        urgency_score = 0.0
        for term in urgency_terms:
            if term in text:
                urgency_score += 0.2
        
        return min(1.0, urgency_score)
    
    def _calculate_volatility_indicator(self, text: str) -> float:
        """Calculate volatility indicator"""
        volatility_terms = ["volatile", "swing", "fluctuate", "unstable", "erratic", "unpredictable"]
        
        volatility_score = 0.0
        for term in volatility_terms:
            if term in text:
                volatility_score += 0.15
        
        return min(1.0, volatility_score)
    
    def _get_source_credibility(self, source: str) -> float:
        """Get source credibility score"""
        credibility_map = {
            "reuters": 0.95, "bloomberg": 0.95, "wsj": 0.95,
            "cnbc": 0.85, "marketwatch": 0.85, "yahoo": 0.75,
            "twitter": 0.30, "reddit": 0.40, "facebook": 0.35
        }
        
        source_lower = source.lower()
        for known_source, credibility in credibility_map.items():
            if known_source in source_lower:
                return credibility
        
        return 0.50  # Default
    
    def _score_to_label(self, score: float) -> SentimentLabel:
        """Convert sentiment score to label"""
        if score >= 0.6:
            return SentimentLabel.VERY_POSITIVE
        elif score >= 0.2:
            return SentimentLabel.POSITIVE
        elif score >= -0.2:
            return SentimentLabel.NEUTRAL
        elif score >= -0.6:
            return SentimentLabel.NEGATIVE
        else:
            return SentimentLabel.VERY_NEGATIVE
    
    def _update_sentiment_windows(self, analysis: SentimentAnalysis):
        """Update sentiment windows for trend analysis"""
        # Overall market sentiment
        self.sentiment_windows["market"].append({
            "timestamp": analysis.timestamp,
            "sentiment": analysis.sentiment_score,
            "confidence": analysis.confidence
        })
        
        # Symbol-specific sentiment
        for symbol in analysis.mentioned_symbols:
            if symbol in analysis.symbol_sentiments:
                self.sentiment_windows[symbol].append({
                    "timestamp": analysis.timestamp,
                    "sentiment": analysis.symbol_sentiments[symbol],
                    "confidence": analysis.confidence
                })
    
    async def _trend_analysis_loop(self):
        """Background task for trend analysis"""
        while True:
            try:
                await asyncio.sleep(self.config.sentiment_window_minutes * 60)
                await self._analyze_sentiment_trends()
            except Exception as e:
                self.logger.error(f"Error in trend analysis: {e}")
    
    async def _analyze_sentiment_trends(self):
        """Analyze sentiment trends and detect shifts"""
        try:
            for symbol, window in self.sentiment_windows.items():
                if len(window) >= self.config.min_samples_for_trend:
                    trend = self._calculate_trend(symbol, window)
                    if trend and abs(trend.sentiment_change) >= self.config.shift_threshold:
                        await self._publish_trend_insight(trend)
                        self.trends_detected += 1
        
        except Exception as e:
            self.logger.error(f"Error analyzing trends: {e}")
    
    def _calculate_trend(self, symbol: str, window: deque) -> Optional[SentimentTrend]:
        """Calculate sentiment trend for a window"""
        try:
            if len(window) < 2:
                return None
            
            # Convert to lists for analysis
            timestamps = [item["timestamp"] for item in window]
            sentiments = [item["sentiment"] for item in window]
            confidences = [item["confidence"] for item in window]
            
            # Calculate trend metrics
            initial_sentiment = sentiments[0]
            final_sentiment = sentiments[-1]
            sentiment_change = final_sentiment - initial_sentiment
            
            # Determine trend direction
            if sentiment_change > 0.1:
                direction = "bullish"
            elif sentiment_change < -0.1:
                direction = "bearish"
            else:
                direction = "sideways"
            
            # Calculate statistics
            mean_sentiment = statistics.mean(sentiments)
            sentiment_volatility = statistics.stdev(sentiments) if len(sentiments) > 1 else 0.0
            mean_confidence = statistics.mean(confidences)
            
            return SentimentTrend(
                symbol=symbol if symbol != "market" else None,
                timeframe=f"{self.config.sentiment_window_minutes}m",
                start_time=timestamps[0],
                end_time=timestamps[-1],
                initial_sentiment=initial_sentiment,
                final_sentiment=final_sentiment,
                sentiment_change=sentiment_change,
                trend_direction=direction,
                mean_sentiment=mean_sentiment,
                sentiment_volatility=sentiment_volatility,
                sample_count=len(sentiments),
                confidence=mean_confidence
            )
            
        except Exception as e:
            self.logger.error(f"Error calculating trend for {symbol}: {e}")
            return None
    
    async def _publish_sentiment_insight(self, analysis: SentimentAnalysis):
        """Publish sentiment insight to message bus"""
        try:
            message = {
                "type": "sentiment_insight",
                "timestamp": datetime.now().isoformat(),
                "agent_name": self.config.agent_name,
                "data": analysis.to_dict()
            }
            
            await self.publish_to_topic(
                self.config.sentiment_insights_topic,
                json.dumps(message).encode()
            )
            
            self.logger.debug(f"Published sentiment insight: {analysis.content_id}")
            
        except Exception as e:
            self.logger.error(f"Error publishing sentiment insight: {e}")
    
    async def _publish_trend_insight(self, trend: SentimentTrend):
        """Publish sentiment trend insight"""
        try:
            message = {
                "type": "sentiment_trend",
                "timestamp": datetime.now().isoformat(),
                "agent_name": self.config.agent_name,
                "data": trend.to_dict()
            }
            
            await self.publish_to_topic(
                self.config.sentiment_insights_topic,
                json.dumps(message).encode()
            )
            
            self.logger.info("Published sentiment trend", extra={
                "symbol": trend.symbol or "market",
                "direction": trend.direction,
                "change": trend.sentiment_change
            })
            
        except Exception as e:
            self.logger.error(f"Error publishing trend insight: {e}")
    
    async def _peer_collaboration_loop(self):
        """Background task for peer collaboration and consensus building"""
        while True:
            try:
                await asyncio.sleep(300)  # Every 5 minutes
                
                if self.a2a_manager and len(self.sentiment_history) > 0:
                    # Get recent sentiment trends for collaboration
                    recent_sentiments = list(self.sentiment_history)[-10:]  # Last 10 analyses
                    
                    # Calculate aggregate sentiment metrics
                    avg_sentiment = statistics.mean([s.sentiment_score for s in recent_sentiments])
                    avg_confidence = statistics.mean([s.confidence for s in recent_sentiments])
                    
                    # Initiate consensus building for significant sentiment shifts
                    if abs(avg_sentiment) > 0.5 and avg_confidence > 0.7:
                        await self._initiate_sentiment_consensus(avg_sentiment, avg_confidence)
                        
            except Exception as e:
                self.logger.error(f"Error in peer collaboration loop: {e}")
    
    async def _initiate_sentiment_consensus(self, sentiment_score: float, confidence: float):
        """Initiate sentiment consensus building with peer agents"""
        try:
            consensus_data = {
                "sentiment_direction": "positive" if sentiment_score > 0 else "negative",
                "sentiment_strength": abs(sentiment_score),
                "confidence_level": confidence,
                "analysis_window": "5_minutes",
                "agent_assessment": {
                    "dominant_emotion": self._get_recent_dominant_emotion(),
                    "volatility_indication": self._get_recent_volatility(),
                    "market_mood": sentiment_score
                }
            }
            
            participants = ["news_analysis_agent", "technical_analysis_agent"]
            
            consensus_result = await self.a2a_manager.orchestrator.initiate_consensus_building(
                topic="market_sentiment_assessment",
                participants=participants,
                decision_data=consensus_data,
                timeout=120  # 2 minutes
            )
            
            if consensus_result["consensus_achieved"]:
                self.logger.info("Sentiment consensus achieved", extra={
                    "agreement_score": consensus_result["agreement_score"],
                    "final_decision": consensus_result["final_decision"],
                    "participants": len(participants)
                })
            else:
                self.logger.warning("Sentiment consensus not achieved", extra={
                    "agreement_score": consensus_result["agreement_score"],
                    "dissenting_agents": consensus_result["dissenting_agents"]
                })
                
        except Exception as e:
            self.logger.error(f"Error in sentiment consensus initiation: {e}")
    
    def _get_recent_dominant_emotion(self) -> str:
        """Get dominant emotion from recent analyses"""
        if not self.sentiment_history:
            return "neutral"
        
        recent_emotions = [s.primary_emotion.value for s in list(self.sentiment_history)[-5:]]
        emotion_counts = {}
        for emotion in recent_emotions:
            emotion_counts[emotion] = emotion_counts.get(emotion, 0) + 1
        
        return max(emotion_counts.items(), key=lambda x: x[1])[0] if emotion_counts else "neutral"
    
    def _get_recent_volatility(self) -> float:
        """Get recent volatility indicator"""
        if not self.sentiment_history:
            return 0.5
        
        recent_volatilities = [s.volatility_indicator for s in list(self.sentiment_history)[-5:]]
        return statistics.mean(recent_volatilities) if recent_volatilities else 0.5
    
    async def _handle_a2a_message(self, message) -> Optional[Dict[str, Any]]:
        """Handle incoming A2A messages"""
        try:
            if hasattr(self.ai_analyzer, 'handle_agent_message'):
                response = await self.ai_analyzer.handle_agent_message(message)
                if response:
                    return response.__dict__
            
            # Handle direct sentiment-related messages
            if message.content.get("type") == "sentiment_validation_request":
                return await self._handle_sentiment_validation_request(message)
            elif message.content.get("type") == "emotional_consensus_request":
                return await self._handle_emotional_consensus_request(message)
                
        except Exception as e:
            self.logger.error(f"Error handling A2A message: {e}")
            
        return None
    
    async def _handle_sentiment_validation_request(self, message) -> Dict[str, Any]:
        """Handle sentiment validation request from peer agent"""
        try:
            request_data = message.content
            content_id = request_data.get("content_id", "unknown")
            
            # Find our analysis for this content
            our_analysis = None
            for analysis in self.sentiment_history:
                if analysis.content_id == content_id:
                    our_analysis = analysis
                    break
            
            if our_analysis:
                # Compare with requesting agent's analysis
                requested_sentiment = request_data.get("sentiment_score", 0)
                our_sentiment = our_analysis.sentiment_score
                
                agreement_score = 1.0 - abs(requested_sentiment - our_sentiment)
                
                response = {
                    "validation_result": "agreement" if agreement_score > 0.7 else "disagreement",
                    "agreement_score": agreement_score,
                    "our_sentiment": our_sentiment,
                    "our_confidence": our_analysis.confidence,
                    "emotional_alignment": our_analysis.primary_emotion.value
                }
            else:
                response = {
                    "validation_result": "no_data",
                    "message": "No analysis found for requested content"
                }
            
            return response
            
        except Exception as e:
            self.logger.error(f"Error handling sentiment validation request: {e}")
            return {"validation_result": "error", "message": str(e)}
    
    async def _handle_emotional_consensus_request(self, message) -> Dict[str, Any]:
        """Handle emotional consensus request from peer agent"""
        try:
            request_data = message.content
            
            # Get our current emotional assessment
            recent_emotions = self._get_recent_emotional_profile()
            market_mood = self._calculate_current_market_mood()
            
            response = {
                "consensus_participation": "active",
                "emotional_profile": recent_emotions,
                "market_mood_assessment": market_mood,
                "confidence_in_assessment": self._calculate_emotional_confidence(),
                "recommended_action": self._generate_emotional_recommendation(recent_emotions, market_mood)
            }
            
            return response
            
        except Exception as e:
            self.logger.error(f"Error handling emotional consensus request: {e}")
            return {"consensus_participation": "error", "message": str(e)}
    
    def _get_recent_emotional_profile(self) -> Dict[str, float]:
        """Get recent emotional profile aggregation"""
        if not self.sentiment_history:
            return {"neutral": 1.0}
        
        recent_analyses = list(self.sentiment_history)[-10:]
        emotion_aggregation = {}
        
        for analysis in recent_analyses:
            for emotion, score in analysis.emotion_scores.items():
                emotion_aggregation[emotion] = emotion_aggregation.get(emotion, 0) + score
        
        # Normalize scores
        total_score = sum(emotion_aggregation.values())
        if total_score > 0:
            emotion_aggregation = {k: v/total_score for k, v in emotion_aggregation.items()}
        
        return emotion_aggregation
    
    def _calculate_current_market_mood(self) -> Dict[str, Any]:
        """Calculate current market mood assessment"""
        if not self.sentiment_history:
            return {"mood": "neutral", "strength": 0.5, "trend": "stable"}
        
        recent_sentiments = [s.sentiment_score for s in list(self.sentiment_history)[-10:]]
        
        avg_sentiment = statistics.mean(recent_sentiments)
        sentiment_trend = "rising" if len(recent_sentiments) > 1 and recent_sentiments[-1] > recent_sentiments[-2] else "falling"
        
        mood = "positive" if avg_sentiment > 0.2 else "negative" if avg_sentiment < -0.2 else "neutral"
        strength = abs(avg_sentiment)
        
        return {
            "mood": mood,
            "strength": strength,
            "trend": sentiment_trend,
            "volatility": statistics.stdev(recent_sentiments) if len(recent_sentiments) > 1 else 0.0
        }
    
    def _calculate_emotional_confidence(self) -> float:
        """Calculate confidence in emotional assessment"""
        if not self.sentiment_history:
            return 0.5
        
        recent_confidences = [s.confidence for s in list(self.sentiment_history)[-10:]]
        return statistics.mean(recent_confidences)
    
    def _generate_emotional_recommendation(self, emotions: Dict[str, float], mood: Dict[str, Any]) -> str:
        """Generate recommendation based on emotional state"""
        dominant_emotion = max(emotions.items(), key=lambda x: x[1])[0] if emotions else "neutral"
        
        if dominant_emotion == "fear" and mood["mood"] == "negative":
            return "monitor_for_panic_selling"
        elif dominant_emotion == "greed" and mood["mood"] == "positive":
            return "watch_for_euphoric_bubbles"
        elif dominant_emotion == "confidence" and mood["trend"] == "rising":
            return "positive_momentum_confirmed"
        else:
            return "maintain_current_assessment"
    
    async def get_agent_status(self) -> Dict[str, Any]:
        """Get current agent status"""
        base_status = await super().get_agent_status()
        
        agent_status = {
            "content_processed": self.content_processed,
            "insights_published": self.insights_published,
            "trends_detected": self.trends_detected,
            "sentiment_history_size": len(self.sentiment_history),
            "tracked_symbols": len(self.sentiment_windows),
            "config": {
                "confidence_threshold": self.config.confidence_threshold,
                "sentiment_window_minutes": self.config.sentiment_window_minutes,
                "use_ml_models": self.config.use_ml_models,
                "track_emotions": self.config.track_market_emotions
            },
            # AI Framework Status
            "ai_framework": {
                "available": AI_FRAMEWORK_AVAILABLE,
                "langgraph_enabled": self.config.enable_langgraph and self.ai_analyzer is not None,
                "a2a_enabled": self.config.enable_a2a_communication and self.a2a_manager is not None,
                "peer_validation": self.config.a2a_peer_validation,
                "emotional_consensus_threshold": self.config.emotional_consensus_threshold
            }
        }
        
        # Add A2A system stats if available
        if self.a2a_manager:
            agent_status["a2a_stats"] = self.a2a_manager.get_system_stats()
        
        # Add recent emotional state
        if self.sentiment_history:
            agent_status["current_state"] = {
                "recent_dominant_emotion": self._get_recent_dominant_emotion(),
                "recent_volatility": self._get_recent_volatility(),
                "market_mood": self._calculate_current_market_mood(),
                "emotional_confidence": self._calculate_emotional_confidence()
            }
        
        return {**base_status, **agent_status}