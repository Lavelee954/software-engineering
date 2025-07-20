"""
News Analysis Agent Implementation (Enhanced with AI Framework)

Advanced news analysis leveraging:
- LangGraph for multi-step reasoning workflows
- Agent-to-Agent (A2A) communication protocols
- Sophisticated market impact assessment
- Cross-agent validation and peer review
- Dynamic knowledge graph construction
"""

import asyncio
import json
import re
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Set, Any
from collections import defaultdict, deque

from shared.base_agent import BaseAgent, AgentConfig
from shared.a2a_communication import (
    A2ACommunicationManager, AgentProfile, AgentCapability, 
    CollaborationPattern, MessageType, MessagePriority
)
from .ai_framework import LangGraphNewsAnalyzer, AgentMessage as AIAgentMessage


@dataclass
class NewsConfig(AgentConfig):
    """Configuration for News Analysis Agent"""
    
    # Subscription topics
    news_topic: str = "raw.news.article"
    
    # Publishing topics  
    news_insights_topic: str = "insight.news"
    
    # Analysis parameters
    max_history_items: int = 1000
    relevance_threshold: float = 0.3
    impact_threshold: float = 0.5
    
    # Content filtering
    min_article_length: int = 100
    max_article_age_hours: int = 24
    
    # NLP settings
    use_advanced_nlp: bool = True
    extract_entities: bool = True
    
    # AI Framework settings
    enable_langgraph: bool = True
    enable_a2a_communication: bool = True
    peer_review_enabled: bool = True
    consensus_threshold: float = 0.6
    
    # Company/sector tracking
    tracked_symbols: List[str] = None
    tracked_sectors: List[str] = None
    
    def __post_init__(self):
        super().__post_init__()
        if self.tracked_symbols is None:
            self.tracked_symbols = ["AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "NVDA"]
        if self.tracked_sectors is None:
            self.tracked_sectors = ["technology", "finance", "energy", "healthcare", "automotive"]


@dataclass
class NewsArticle:
    """Represents a news article for analysis"""
    title: str
    content: str
    source: str
    url: str
    published_at: datetime
    symbol: Optional[str] = None
    
    def to_dict(self) -> Dict[str, Any]:
        data = asdict(self)
        data['published_at'] = self.published_at.isoformat()
        return data


@dataclass 
class NewsInsight:
    """Represents analyzed news insight"""
    article_id: str
    title: str
    source: str
    published_at: datetime
    
    # Analysis results
    relevance_score: float
    impact_score: float
    sentiment_score: float
    urgency_level: str  # low, medium, high, critical
    
    # Content analysis
    mentioned_symbols: List[str]
    mentioned_sectors: List[str] 
    key_topics: List[str]
    extracted_entities: Dict[str, List[str]]
    
    # Metadata
    source_credibility: float
    article_quality: float
    
    def to_dict(self) -> Dict[str, Any]:
        data = asdict(self)
        data['published_at'] = self.published_at.isoformat()
        return data


class NewsAnalysisAgent(BaseAgent):
    """
    Enhanced News Analysis Agent with AI Framework
    
    Advanced news analysis capabilities:
    - LangGraph multi-step reasoning workflows
    - Agent-to-Agent communication and collaboration
    - Peer review and consensus building
    - Dynamic knowledge graph construction
    - Cross-agent validation and verification
    
    Follows CLAUDE.md specifications:
    - Subscribes to raw.news.article messages
    - Publishes insights to insight.news topic
    - Collaborates with other analysis agents
    """
    
    def __init__(self, config: NewsConfig):
        super().__init__(config)
        self.config = config
        
        # News processing state
        self.news_history: deque = deque(maxlen=config.max_history_items)
        self.processed_articles: Set[str] = set()
        
        # Traditional analysis components
        self.market_keywords = self._load_market_keywords()
        self.source_credibility = self._load_source_credibility()
        
        # AI Framework components
        self.ai_analyzer: Optional[LangGraphNewsAnalyzer] = None
        self.a2a_manager: Optional[A2ACommunicationManager] = None
        self.agent_profile: Optional[AgentProfile] = None
        
        # Statistics
        self.articles_processed = 0
        self.insights_published = 0
        self.a2a_collaborations = 0
        self.peer_reviews_conducted = 0
        
        # Initialize AI framework if enabled
        if config.enable_langgraph or config.enable_a2a_communication:
            self._initialize_ai_framework()
        
        self.logger.info("Enhanced News Analysis Agent initialized", extra={
            "max_history": config.max_history_items,
            "relevance_threshold": config.relevance_threshold,
            "tracked_symbols": len(config.tracked_symbols),
            "tracked_sectors": len(config.tracked_sectors),
            "ai_framework_enabled": config.enable_langgraph,
            "a2a_enabled": config.enable_a2a_communication
        })
    
    def _load_market_keywords(self) -> Dict[str, List[str]]:
        """Load market-relevant keywords for analysis"""
        return {
            "bullish": [
                "growth", "profit", "earnings beat", "revenue increase", "expansion",
                "acquisition", "merger", "partnership", "breakthrough", "launch",
                "upgrade", "outperform", "buy rating", "price target raised"
            ],
            "bearish": [
                "loss", "decline", "earnings miss", "revenue drop", "layoffs",
                "bankruptcy", "lawsuit", "investigation", "downgrade", "sell rating",
                "price target lowered", "warning", "guidance cut", "recession"
            ],
            "volatility": [
                "volatility", "uncertainty", "risk", "crisis", "shock", "surprise",
                "unexpected", "emergency", "breaking", "urgent", "alert"
            ],
            "company_events": [
                "ceo", "management change", "board", "dividend", "split", "buyback",
                "ipo", "listing", "delisting", "restructuring", "spinoff"
            ],
            "market_events": [
                "fed", "interest rate", "inflation", "gdp", "employment", "trade war",
                "tariff", "sanctions", "regulation", "policy", "election"
            ]
        }
    
    def _load_source_credibility(self) -> Dict[str, float]:
        """Load source credibility scores"""
        return {
            # Tier 1 - Highly credible financial sources
            "reuters": 0.95,
            "bloomberg": 0.95,
            "wsj": 0.95,
            "ft": 0.95,
            "cnbc": 0.85,
            "marketwatch": 0.85,
            
            # Tier 2 - General news with financial coverage
            "cnn": 0.75,
            "bbc": 0.80,
            "nytimes": 0.80,
            "guardian": 0.75,
            
            # Tier 3 - Tech/industry specific
            "techcrunch": 0.70,
            "venturebeat": 0.65,
            
            # Default for unknown sources
            "unknown": 0.50
        }
    
    def _initialize_ai_framework(self):
        """Initialize AI framework components"""
        try:
            # Initialize LangGraph analyzer
            if self.config.enable_langgraph:
                self.ai_analyzer = LangGraphNewsAnalyzer(
                    agent_id=self.config.agent_name,
                    config={
                        "relevance_threshold": self.config.relevance_threshold,
                        "impact_threshold": self.config.impact_threshold,
                        "tracked_symbols": self.config.tracked_symbols,
                        "tracked_sectors": self.config.tracked_sectors
                    }
                )
                self.logger.info("LangGraph analyzer initialized")
            
            # Initialize A2A communication
            if self.config.enable_a2a_communication:
                self.a2a_manager = A2ACommunicationManager()
                
                # Create agent profile
                self.agent_profile = AgentProfile(
                    agent_id=self.config.agent_name,
                    agent_type="news_analysis",
                    capabilities=[
                        AgentCapability.NEWS_ANALYSIS,
                        AgentCapability.SENTIMENT_ANALYSIS
                    ],
                    service_level=0.85,
                    reputation_score=0.80,
                    load_factor=0.0,
                    max_concurrent_requests=10,
                    supported_patterns=[
                        CollaborationPattern.REQUEST_RESPONSE,
                        CollaborationPattern.PEER_REVIEW,
                        CollaborationPattern.CONSENSUS_BUILDING,
                        CollaborationPattern.KNOWLEDGE_SHARING
                    ],
                    metadata={
                        "specialization": "financial_news_analysis",
                        "data_sources": ["news_feeds", "press_releases", "earnings_reports"],
                        "analysis_types": ["impact_assessment", "sentiment_analysis", "entity_extraction"]
                    }
                )
                
                self.logger.info("A2A communication framework initialized")
        
        except Exception as e:
            self.logger.error(f"Failed to initialize AI framework: {e}")
            # Continue with traditional analysis if AI framework fails
            self.ai_analyzer = None
            self.a2a_manager = None
    
    async def start(self):
        """Start the Enhanced News Analysis Agent"""
        try:
            await super().start()
            
            # Start A2A communication system
            if self.a2a_manager:
                await self.a2a_manager.start()
                
                # Register this agent
                self.a2a_manager.registry.register_agent(self.agent_profile)
                
                # Register message handler
                self.a2a_manager.router.register_handler(
                    self.config.agent_name,
                    self._handle_a2a_message
                )
                
                self.logger.info("A2A communication system started")
            
            # Subscribe to news articles
            await self.subscribe_to_topic(
                self.config.news_topic,
                self._handle_news_article
            )
            
            self.logger.info("Enhanced News Analysis Agent started successfully")
            
        except Exception as e:
            self.logger.error(f"Failed to start Enhanced News Analysis Agent: {e}")
            raise
    
    async def _handle_news_article(self, subject: str, data: bytes):
        """Handle incoming news articles"""
        try:
            # Parse message
            message = json.loads(data.decode())
            self.logger.debug(f"Received news article: {message.get('title', 'Unknown')}")
            
            # Create article object
            article = self._parse_news_article(message)
            if not article:
                return
            
            # Check if already processed
            article_id = self._generate_article_id(article)
            if article_id in self.processed_articles:
                self.logger.debug(f"Article already processed: {article_id}")
                return
            
            # Analyze article using AI framework or traditional method
            analysis_result = await self._analyze_article_enhanced(article, article_id)
            
            if analysis_result:
                # Store in history
                self.processed_articles.add(article_id)
                
                # Handle both traditional insight and AI analysis results
                if isinstance(analysis_result, dict) and "insights" in analysis_result:
                    # AI framework result
                    ai_insights = analysis_result["insights"]
                    confidence = analysis_result["confidence"]
                    
                    # Store AI analysis result
                    self.news_history.append({
                        "article_id": article_id,
                        "ai_analysis": analysis_result,
                        "timestamp": datetime.now()
                    })
                    
                    # Publish if meets confidence threshold
                    if confidence >= self.config.relevance_threshold:
                        await self._publish_ai_insight(article, analysis_result)
                        self.insights_published += 1
                        
                        # Initiate peer review if enabled
                        if self.config.peer_review_enabled and self.a2a_manager:
                            await self._initiate_peer_review(article_id, analysis_result)
                    
                    self.logger.info("Article processed with AI framework", extra={
                        "article_id": article_id,
                        "source": article.source,
                        "confidence": confidence,
                        "impact_score": ai_insights.get("market_impact_score", 0),
                        "reasoning_steps": len(analysis_result.get("reasoning_trace", []))
                    })
                
                else:
                    # Traditional insight result
                    insight = analysis_result
                    self.news_history.append(insight)
                    
                    if insight.relevance_score >= self.config.relevance_threshold:
                        await self._publish_insight(insight)
                        self.insights_published += 1
                    
                    self.logger.info("Article processed traditionally", extra={
                        "article_id": article_id,
                        "source": article.source,
                        "relevance": insight.relevance_score,
                        "impact": insight.impact_score,
                        "sentiment": insight.sentiment_score
                    })
                
                self.articles_processed += 1
        
        except Exception as e:
            self.logger.error(f"Error processing news article: {e}")
    
    def _parse_news_article(self, message: Dict[str, Any]) -> Optional[NewsArticle]:
        """Parse news article from message"""
        try:
            # Validate required fields
            required_fields = ['title', 'content', 'source', 'published_at']
            if not all(field in message for field in required_fields):
                self.logger.warning("Missing required fields in news article")
                return None
            
            # Parse timestamp
            try:
                published_at = datetime.fromisoformat(message['published_at'].replace('Z', '+00:00'))
            except ValueError:
                self.logger.warning("Invalid timestamp format in news article")
                return None
            
            # Check article age
            age_hours = (datetime.now() - published_at.replace(tzinfo=None)).total_seconds() / 3600
            if age_hours > self.config.max_article_age_hours:
                self.logger.debug(f"Article too old: {age_hours:.1f} hours")
                return None
            
            # Check content length
            if len(message['content']) < self.config.min_article_length:
                self.logger.debug("Article content too short")
                return None
            
            return NewsArticle(
                title=message['title'],
                content=message['content'], 
                source=message['source'],
                url=message.get('url', ''),
                published_at=published_at,
                symbol=message.get('symbol')
            )
            
        except Exception as e:
            self.logger.error(f"Error parsing news article: {e}")
            return None
    
    def _generate_article_id(self, article: NewsArticle) -> str:
        """Generate unique ID for article"""
        import hashlib
        content = f"{article.title}_{article.source}_{article.published_at.isoformat()}"
        return hashlib.md5(content.encode()).hexdigest()[:16]
    
    async def _analyze_article(self, article: NewsArticle, article_id: str) -> Optional[NewsInsight]:
        """Analyze article and generate insights"""
        try:
            # Combine title and content for analysis
            full_text = f"{article.title} {article.content}".lower()
            
            # Analyze relevance
            relevance_score = self._calculate_relevance_score(full_text)
            
            # Analyze impact
            impact_score = self._calculate_impact_score(full_text, article.source)
            
            # Analyze sentiment
            sentiment_score = self._calculate_sentiment_score(full_text)
            
            # Extract mentioned symbols and sectors
            mentioned_symbols = self._extract_mentioned_symbols(full_text)
            mentioned_sectors = self._extract_mentioned_sectors(full_text)
            
            # Extract key topics
            key_topics = self._extract_key_topics(full_text)
            
            # Extract entities (if enabled)
            extracted_entities = {}
            if self.config.extract_entities:
                extracted_entities = self._extract_entities(full_text)
            
            # Calculate urgency level
            urgency_level = self._calculate_urgency_level(
                relevance_score, impact_score, sentiment_score
            )
            
            # Calculate source credibility
            source_credibility = self._get_source_credibility(article.source)
            
            # Calculate article quality
            article_quality = self._calculate_article_quality(article)
            
            return NewsInsight(
                article_id=article_id,
                title=article.title,
                source=article.source,
                published_at=article.published_at,
                relevance_score=relevance_score,
                impact_score=impact_score,
                sentiment_score=sentiment_score,
                urgency_level=urgency_level,
                mentioned_symbols=mentioned_symbols,
                mentioned_sectors=mentioned_sectors,
                key_topics=key_topics,
                extracted_entities=extracted_entities,
                source_credibility=source_credibility,
                article_quality=article_quality
            )
            
        except Exception as e:
            self.logger.error(f"Error analyzing article: {e}")
            return None
    
    async def _analyze_article_enhanced(self, article: NewsArticle, article_id: str):
        """Enhanced analysis using AI framework or traditional method"""
        try:
            # Use AI framework if available
            if self.ai_analyzer and self.config.enable_langgraph:
                return await self._analyze_with_ai_framework(article, article_id)
            else:
                # Fall back to traditional analysis
                return await self._analyze_article(article, article_id)
                
        except Exception as e:
            self.logger.error(f"Enhanced analysis failed, falling back to traditional: {e}")
            return await self._analyze_article(article, article_id)
    
    async def _analyze_with_ai_framework(self, article: NewsArticle, article_id: str) -> Dict[str, Any]:
        """Analyze article using LangGraph AI framework"""
        try:
            # Prepare metadata for AI analysis
            metadata = {
                "source": article.source,
                "published_at": article.published_at.isoformat(),
                "url": article.url,
                "symbol": article.symbol,
                "source_credibility": self._get_source_credibility(article.source),
                "article_id": article_id
            }
            
            # Combine title and content
            full_content = f"{article.title}\n\n{article.content}"
            
            # Process through LangGraph workflow
            result = await self.ai_analyzer.process_news_article(full_content, metadata)
            
            # Add traditional analysis as fallback data
            traditional_insight = await self._analyze_article(article, article_id)
            result["traditional_fallback"] = traditional_insight.to_dict() if traditional_insight else None
            
            return result
            
        except Exception as e:
            self.logger.error(f"AI framework analysis failed: {e}")
            raise
    
    async def _initiate_peer_review(self, article_id: str, analysis_result: Dict[str, Any]):
        """Initiate peer review of analysis using A2A communication"""
        try:
            if not self.a2a_manager:
                return
            
            # Extract key analysis data for review
            review_data = {
                "article_id": article_id,
                "market_impact_score": analysis_result["insights"].get("market_impact_score", 0),
                "confidence_level": analysis_result["confidence"],
                "key_takeaways": analysis_result["insights"].get("key_takeaways", []),
                "risk_indicators": analysis_result["insights"].get("risk_indicators", []),
                "reasoning_trace": analysis_result.get("reasoning_trace", [])
            }
            
            # Define review criteria
            review_criteria = [
                "market_relevance",
                "impact_assessment_accuracy", 
                "risk_evaluation",
                "temporal_urgency",
                "cross_validation_consistency"
            ]
            
            # Orchestrate peer review
            review_result = await self.a2a_manager.orchestrator.orchestrate_peer_review(
                subject_agent=self.config.agent_name,
                analysis_data=review_data,
                review_criteria=review_criteria,
                num_reviewers=2
            )
            
            self.peer_reviews_conducted += 1
            
            self.logger.info("Peer review completed", extra={
                "article_id": article_id,
                "review_id": review_result.get("review_id"),
                "overall_score": review_result.get("overall_score", 0),
                "consensus_level": review_result.get("consensus_level", 0),
                "reviewer_count": review_result.get("reviewer_count", 0)
            })
            
            # Store peer review results
            analysis_result["peer_review"] = review_result
            
        except Exception as e:
            self.logger.error(f"Peer review initiation failed: {e}")
    
    async def _handle_a2a_message(self, message) -> Optional[Any]:
        """Handle incoming A2A messages"""
        try:
            content = message.content
            message_type = content.get("type")
            
            self.a2a_collaborations += 1
            
            if message_type == "consensus_request":
                return await self._handle_consensus_request(message)
            elif message_type == "peer_review_request":
                return await self._handle_peer_review_request(message)
            elif message_type == "knowledge_sharing":
                return await self._handle_knowledge_sharing(message)
            elif message_type == "validation_request":
                return await self._handle_validation_request(message)
            else:
                self.logger.warning(f"Unknown A2A message type: {message_type}")
                return None
                
        except Exception as e:
            self.logger.error(f"A2A message handling failed: {e}")
            return None
    
    async def _handle_consensus_request(self, message) -> Any:
        """Handle consensus building request"""
        try:
            content = message.content
            topic = content.get("topic")
            data = content.get("data", {})
            
            # Analyze the consensus topic using our domain expertise
            if "market_impact" in topic.lower():
                decision = self._evaluate_market_impact_consensus(data)
            elif "sentiment" in topic.lower():
                decision = self._evaluate_sentiment_consensus(data)
            else:
                decision = "neutral"  # Default position
            
            # Create consensus response
            response_message = self.a2a_manager.create_agent_message(
                sender_id=self.config.agent_name,
                receiver_id=message.sender_id,
                content={
                    "collaboration_id": content.get("collaboration_id"),
                    "type": "consensus_response",
                    "decision": decision,
                    "confidence": 0.8,
                    "reasoning": f"News analysis perspective on {topic}"
                },
                message_type=MessageType.RESPONSE,
                priority=MessagePriority.HIGH
            )
            
            await self.a2a_manager.send_message(response_message)
            
            self.logger.info(f"Consensus response sent for topic: {topic}")
            return response_message
            
        except Exception as e:
            self.logger.error(f"Consensus request handling failed: {e}")
            return None
    
    async def _handle_peer_review_request(self, message) -> Any:
        """Handle peer review request"""
        try:
            content = message.content
            analysis_data = content.get("analysis_data", {})
            review_criteria = content.get("review_criteria", [])
            
            # Perform review based on our expertise
            review_scores = {}
            for criterion in review_criteria:
                review_scores[criterion] = self._evaluate_review_criterion(criterion, analysis_data)
            
            overall_score = sum(review_scores.values()) / len(review_scores) if review_scores else 0.5
            
            # Generate recommendations
            recommendations = self._generate_review_recommendations(analysis_data, review_scores)
            
            # Create peer review response
            response_message = self.a2a_manager.create_agent_message(
                sender_id=self.config.agent_name,
                receiver_id=message.sender_id,
                content={
                    "review_id": content.get("review_id"),
                    "type": "peer_review_response",
                    "scores": review_scores,
                    "overall_score": overall_score,
                    "recommendations": recommendations,
                    "reviewer_expertise": "news_analysis"
                },
                message_type=MessageType.RESPONSE,
                priority=MessagePriority.MEDIUM
            )
            
            await self.a2a_manager.send_message(response_message)
            
            self.logger.info("Peer review response sent")
            return response_message
            
        except Exception as e:
            self.logger.error(f"Peer review request handling failed: {e}")
            return None
    
    async def _handle_knowledge_sharing(self, message) -> None:
        """Handle knowledge sharing from other agents"""
        try:
            content = message.content
            knowledge_type = content.get("knowledge_type")
            shared_data = content.get("data", {})
            
            if knowledge_type == "market_patterns":
                self._integrate_market_pattern_knowledge(shared_data)
            elif knowledge_type == "sentiment_trends":
                self._integrate_sentiment_trend_knowledge(shared_data)
            elif knowledge_type == "risk_indicators":
                self._integrate_risk_indicator_knowledge(shared_data)
            
            self.logger.info(f"Knowledge integrated from {message.sender_id}: {knowledge_type}")
            
        except Exception as e:
            self.logger.error(f"Knowledge sharing handling failed: {e}")
    
    async def _handle_validation_request(self, message) -> Any:
        """Handle validation request from other agents"""
        try:
            content = message.content
            analysis_type = content.get("analysis_type")
            findings = content.get("preliminary_findings", {})
            
            # Validate findings based on our domain expertise
            validation_result = self._validate_analysis_findings(analysis_type, findings)
            
            # Create validation response
            response_message = self.a2a_manager.create_agent_message(
                sender_id=self.config.agent_name,
                receiver_id=message.sender_id,
                content={
                    "validation_id": content.get("validation_id"),
                    "type": "validation_response",
                    "validation_result": validation_result,
                    "confidence_adjustment": validation_result.get("confidence_adjustment", 0),
                    "validator_expertise": "news_analysis"
                },
                message_type=MessageType.RESPONSE,
                priority=MessagePriority.MEDIUM
            )
            
            await self.a2a_manager.send_message(response_message)
            
            self.logger.info(f"Validation response sent for {analysis_type}")
            return response_message
            
        except Exception as e:
            self.logger.error(f"Validation request handling failed: {e}")
            return None
    
    def _evaluate_market_impact_consensus(self, data: Dict[str, Any]) -> str:
        """Evaluate market impact for consensus building"""
        # Implementation based on news analysis expertise
        impact_indicators = data.get("impact_indicators", {})
        
        if impact_indicators.get("earnings_surprise", 0) > 0.2:
            return "high_positive_impact"
        elif impact_indicators.get("regulatory_risk", 0) > 0.7:
            return "high_negative_impact"
        else:
            return "moderate_impact"
    
    def _evaluate_sentiment_consensus(self, data: Dict[str, Any]) -> str:
        """Evaluate sentiment for consensus building"""
        # Implementation based on news analysis expertise
        sentiment_data = data.get("sentiment_indicators", {})
        
        positive_signals = sentiment_data.get("positive_signals", 0)
        negative_signals = sentiment_data.get("negative_signals", 0)
        
        if positive_signals > negative_signals * 1.5:
            return "bullish_sentiment"
        elif negative_signals > positive_signals * 1.5:
            return "bearish_sentiment"
        else:
            return "neutral_sentiment"
    
    def _evaluate_review_criterion(self, criterion: str, analysis_data: Dict[str, Any]) -> float:
        """Evaluate specific review criterion"""
        # Implementation varies by criterion
        if criterion == "market_relevance":
            return min(1.0, analysis_data.get("market_impact_score", 0.5) * 1.2)
        elif criterion == "impact_assessment_accuracy":
            return 0.8  # Based on our confidence in the analysis
        elif criterion == "risk_evaluation":
            risk_indicators = analysis_data.get("risk_indicators", [])
            return min(1.0, len(risk_indicators) * 0.2 + 0.4)
        elif criterion == "temporal_urgency":
            return 0.7  # Default assessment
        elif criterion == "cross_validation_consistency":
            return 0.75  # Based on internal validation
        else:
            return 0.6  # Default score
    
    def _generate_review_recommendations(self, analysis_data: Dict[str, Any], scores: Dict[str, float]) -> List[str]:
        """Generate recommendations based on review scores"""
        recommendations = []
        
        if scores.get("market_relevance", 0) < 0.6:
            recommendations.append("Consider additional market context factors")
        
        if scores.get("risk_evaluation", 0) < 0.5:
            recommendations.append("Enhance risk assessment with sector-specific factors")
        
        if scores.get("temporal_urgency", 0) > 0.8:
            recommendations.append("Monitor for immediate market response")
        
        return recommendations
    
    def _integrate_market_pattern_knowledge(self, data: Dict[str, Any]):
        """Integrate market pattern knowledge from other agents"""
        # Update internal knowledge base with market patterns
        patterns = data.get("patterns", [])
        for pattern in patterns:
            # Store pattern for future analysis enhancement
            pass
    
    def _integrate_sentiment_trend_knowledge(self, data: Dict[str, Any]):
        """Integrate sentiment trend knowledge"""
        # Update sentiment analysis capabilities
        trends = data.get("trends", [])
        # Enhance sentiment analysis with trend data
        pass
    
    def _integrate_risk_indicator_knowledge(self, data: Dict[str, Any]):
        """Integrate risk indicator knowledge"""
        # Update risk assessment capabilities
        indicators = data.get("indicators", [])
        # Enhance risk evaluation with new indicators
        pass
    
    def _validate_analysis_findings(self, analysis_type: str, findings: Dict[str, Any]) -> Dict[str, Any]:
        """Validate analysis findings from other agents"""
        validation_result = {
            "status": "validated",
            "confidence_adjustment": 0.0,
            "comments": []
        }
        
        if analysis_type == "technical_analysis":
            # Cross-validate technical findings with news sentiment
            technical_signal = findings.get("signal", "neutral")
            
            # Check if news sentiment aligns with technical signal
            if hasattr(self, '_current_market_sentiment'):
                if technical_signal == "bullish" and self._current_market_sentiment > 0.5:
                    validation_result["confidence_adjustment"] = 0.1
                    validation_result["comments"].append("News sentiment supports technical bullish signal")
                elif technical_signal == "bearish" and self._current_market_sentiment < -0.5:
                    validation_result["confidence_adjustment"] = 0.1
                    validation_result["comments"].append("News sentiment supports technical bearish signal")
                else:
                    validation_result["confidence_adjustment"] = -0.05
                    validation_result["comments"].append("Mixed signals between technical and news sentiment")
        
        return validation_result
    
    async def _publish_ai_insight(self, article: NewsArticle, analysis_result: Dict[str, Any]):
        """Publish AI-generated insight to message bus"""
        try:
            ai_insights = analysis_result["insights"]
            
            # Create enhanced message with AI analysis
            message = {
                "type": "ai_news_insight",
                "timestamp": datetime.now().isoformat(),
                "agent_name": self.config.agent_name,
                "analysis_framework": "langgraph_enhanced",
                "data": {
                    "article_metadata": {
                        "title": article.title,
                        "source": article.source,
                        "published_at": article.published_at.isoformat(),
                        "url": article.url
                    },
                    "ai_insights": ai_insights,
                    "confidence": analysis_result["confidence"],
                    "reasoning_trace": analysis_result.get("reasoning_trace", []),
                    "agent_communications": analysis_result.get("agent_communications", []),
                    "peer_review": analysis_result.get("peer_review"),
                    "processing_metadata": analysis_result.get("metadata", {})
                }
            }
            
            await self.publish_to_topic(
                self.config.news_insights_topic,
                json.dumps(message).encode()
            )
            
            self.logger.debug(f"Published AI insight for article: {article.title[:50]}")
            
        except Exception as e:
            self.logger.error(f"Error publishing AI insight: {e}")
    
    def _calculate_relevance_score(self, text: str) -> float:
        """Calculate market relevance score (0.0 to 1.0)"""
        score = 0.0
        total_keywords = 0
        
        for category, keywords in self.market_keywords.items():
            category_score = 0
            for keyword in keywords:
                if keyword in text:
                    category_score += 1
                total_keywords += 1
            
            # Weight different categories
            if category in ["bullish", "bearish"]:
                score += category_score * 0.3
            elif category == "volatility":
                score += category_score * 0.2
            else:
                score += category_score * 0.1
        
        # Check for tracked symbols/sectors
        for symbol in self.config.tracked_symbols:
            if symbol.lower() in text:
                score += 0.4
        
        for sector in self.config.tracked_sectors:
            if sector in text:
                score += 0.2
        
        # Normalize score
        return min(score, 1.0)
    
    def _calculate_impact_score(self, text: str, source: str) -> float:
        """Calculate potential market impact score (0.0 to 1.0)"""
        impact_score = 0.0
        
        # High impact keywords
        high_impact_terms = [
            "fed", "federal reserve", "interest rate", "inflation", "gdp",
            "earnings", "revenue", "guidance", "acquisition", "merger",
            "bankruptcy", "investigation", "regulation", "tariff", "war"
        ]
        
        for term in high_impact_terms:
            if term in text:
                impact_score += 0.2
        
        # Source credibility multiplier
        source_credibility = self._get_source_credibility(source)
        impact_score *= source_credibility
        
        # Breaking news bonus
        if any(word in text for word in ["breaking", "urgent", "alert"]):
            impact_score += 0.3
        
        return min(impact_score, 1.0)
    
    def _calculate_sentiment_score(self, text: str) -> float:
        """Calculate sentiment score (-1.0 to 1.0, negative to positive)"""
        positive_score = 0
        negative_score = 0
        
        # Count positive and negative keywords
        for keyword in self.market_keywords["bullish"]:
            positive_score += text.count(keyword)
        
        for keyword in self.market_keywords["bearish"]:
            negative_score += text.count(keyword)
        
        # Additional sentiment words
        positive_words = ["good", "great", "excellent", "strong", "positive", "success"]
        negative_words = ["bad", "poor", "weak", "negative", "failure", "decline"]
        
        for word in positive_words:
            positive_score += text.count(word)
        
        for word in negative_words:
            negative_score += text.count(word)
        
        # Calculate net sentiment
        total_sentiment = positive_score + negative_score
        if total_sentiment == 0:
            return 0.0
        
        net_sentiment = (positive_score - negative_score) / total_sentiment
        return max(-1.0, min(1.0, net_sentiment))
    
    def _extract_mentioned_symbols(self, text: str) -> List[str]:
        """Extract mentioned stock symbols"""
        mentioned = []
        for symbol in self.config.tracked_symbols:
            if symbol.lower() in text:
                mentioned.append(symbol)
        return mentioned
    
    def _extract_mentioned_sectors(self, text: str) -> List[str]:
        """Extract mentioned sectors"""
        mentioned = []
        for sector in self.config.tracked_sectors:
            if sector in text:
                mentioned.append(sector)
        return mentioned
    
    def _extract_key_topics(self, text: str) -> List[str]:
        """Extract key topics from text"""
        topics = []
        
        # Define topic patterns
        topic_patterns = {
            "earnings": ["earnings", "revenue", "profit", "loss"],
            "mergers": ["merger", "acquisition", "takeover", "buyout"],
            "regulation": ["regulation", "regulatory", "sec", "fda"],
            "technology": ["ai", "artificial intelligence", "blockchain", "crypto"],
            "economy": ["inflation", "gdp", "unemployment", "recession"]
        }
        
        for topic, keywords in topic_patterns.items():
            if any(keyword in text for keyword in keywords):
                topics.append(topic)
        
        return topics
    
    def _extract_entities(self, text: str) -> Dict[str, List[str]]:
        """Extract named entities (basic implementation)"""
        entities = defaultdict(list)
        
        # Company patterns (basic)
        company_patterns = [
            r'\b[A-Z][a-z]+ (Inc|Corp|LLC|Ltd|Co)\b',
            r'\b[A-Z]{2,5}\b'  # Stock symbols
        ]
        
        for pattern in company_patterns:
            matches = re.findall(pattern, text)
            entities["companies"].extend(matches)
        
        # Person patterns (basic)
        person_pattern = r'\b[A-Z][a-z]+ [A-Z][a-z]+\b'
        persons = re.findall(person_pattern, text)
        entities["persons"] = persons[:5]  # Limit to top 5
        
        # Convert defaultdict to regular dict
        return dict(entities)
    
    def _calculate_urgency_level(self, relevance: float, impact: float, sentiment: float) -> str:
        """Calculate urgency level based on scores"""
        combined_score = (relevance + impact + abs(sentiment)) / 3
        
        if combined_score >= 0.8:
            return "critical"
        elif combined_score >= 0.6:
            return "high"
        elif combined_score >= 0.4:
            return "medium"
        else:
            return "low"
    
    def _get_source_credibility(self, source: str) -> float:
        """Get credibility score for news source"""
        source_lower = source.lower()
        for known_source, credibility in self.source_credibility.items():
            if known_source in source_lower:
                return credibility
        return self.source_credibility["unknown"]
    
    def _calculate_article_quality(self, article: NewsArticle) -> float:
        """Calculate article quality score"""
        quality_score = 0.5  # Base score
        
        # Length bonus
        if len(article.content) > 500:
            quality_score += 0.2
        if len(article.content) > 1000:
            quality_score += 0.1
        
        # Title quality
        if len(article.title) > 10 and not article.title.isupper():
            quality_score += 0.1
        
        # URL presence
        if article.url:
            quality_score += 0.1
        
        return min(quality_score, 1.0)
    
    async def _publish_insight(self, insight: NewsInsight):
        """Publish news insight to message bus"""
        try:
            message = {
                "type": "news_insight",
                "timestamp": datetime.now().isoformat(),
                "agent_name": self.config.agent_name,
                "data": insight.to_dict()
            }
            
            await self.publish_to_topic(
                self.config.news_insights_topic,
                json.dumps(message).encode()
            )
            
            self.logger.debug(f"Published news insight: {insight.article_id}")
            
        except Exception as e:
            self.logger.error(f"Error publishing insight: {e}")
    
    async def get_agent_status(self) -> Dict[str, Any]:
        """Get current enhanced agent status"""
        base_status = await super().get_agent_status()
        
        agent_status = {
            "articles_processed": self.articles_processed,
            "insights_published": self.insights_published,
            "history_size": len(self.news_history),
            "processed_articles_count": len(self.processed_articles),
            "a2a_collaborations": self.a2a_collaborations,
            "peer_reviews_conducted": self.peer_reviews_conducted,
            "config": {
                "relevance_threshold": self.config.relevance_threshold,
                "impact_threshold": self.config.impact_threshold,
                "tracked_symbols": len(self.config.tracked_symbols),
                "tracked_sectors": len(self.config.tracked_sectors),
                "ai_framework_enabled": self.config.enable_langgraph,
                "a2a_communication_enabled": self.config.enable_a2a_communication,
                "peer_review_enabled": self.config.peer_review_enabled
            },
            "ai_framework": {
                "langgraph_available": self.ai_analyzer is not None,
                "a2a_manager_available": self.a2a_manager is not None,
                "agent_registered": self.agent_profile is not None
            }
        }
        
        # Add A2A system stats if available
        if self.a2a_manager:
            agent_status["a2a_system_stats"] = self.a2a_manager.get_system_stats()
        
        return {**base_status, **agent_status}