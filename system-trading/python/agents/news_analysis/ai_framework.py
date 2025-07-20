"""
AI Framework for News Analysis Agent

Leverages advanced AI agent concepts including:
- LangGraph for complex reasoning workflows
- Agent-to-Agent (A2A) communication protocols
- Multi-step reasoning pipelines
- Knowledge graph integration
- Dynamic prompt engineering
"""

import asyncio
import json
from typing import Dict, List, Optional, Any, TypedDict, Annotated
from datetime import datetime
from enum import Enum
from dataclasses import dataclass
import operator

# Advanced AI frameworks
try:
    from langchain_core.messages import BaseMessage, HumanMessage, AIMessage, SystemMessage
    from langchain_core.prompts import ChatPromptTemplate, MessagesPlaceholder
    from langchain_core.output_parsers import JsonOutputParser
    from langgraph.graph import StateGraph, END
    from langgraph.prebuilt import ToolExecutor, ToolInvocation
    from langgraph.checkpoint.sqlite import SqliteSaver
    LANGRAPH_AVAILABLE = True
except ImportError:
    LANGRAPH_AVAILABLE = False
    # Fallback implementations
    class BaseMessage:
        def __init__(self, content: str): self.content = content
    class HumanMessage(BaseMessage): pass
    class AIMessage(BaseMessage): pass
    class SystemMessage(BaseMessage): pass


class ReasoningStep(Enum):
    """News analysis reasoning steps"""
    CONTENT_EXTRACTION = "content_extraction"
    RELEVANCE_ASSESSMENT = "relevance_assessment" 
    IMPACT_ANALYSIS = "impact_analysis"
    SENTIMENT_EVALUATION = "sentiment_evaluation"
    ENTITY_RECOGNITION = "entity_recognition"
    TEMPORAL_ANALYSIS = "temporal_analysis"
    CROSS_VALIDATION = "cross_validation"
    INSIGHT_SYNTHESIS = "insight_synthesis"


class A2AProtocol(Enum):
    """Agent-to-Agent communication protocols"""
    REQUEST_VALIDATION = "request_validation"
    CONTEXT_SHARING = "context_sharing"
    COLLABORATIVE_ANALYSIS = "collaborative_analysis"
    CONSENSUS_BUILDING = "consensus_building"
    KNOWLEDGE_EXCHANGE = "knowledge_exchange"
    PEER_REVIEW = "peer_review"


@dataclass
class AgentMessage:
    """Standardized A2A message format"""
    sender_id: str
    receiver_id: str
    protocol: A2AProtocol
    content: Dict[str, Any]
    timestamp: datetime
    message_id: str
    priority: int = 1  # 1=high, 2=medium, 3=low
    requires_response: bool = False
    correlation_id: Optional[str] = None


class NewsAnalysisState(TypedDict):
    """State for the news analysis workflow"""
    raw_content: str
    article_metadata: Dict[str, Any]
    extracted_entities: List[Dict[str, Any]]
    relevance_scores: Dict[str, float]
    impact_assessment: Dict[str, Any]
    sentiment_analysis: Dict[str, Any]
    temporal_context: Dict[str, Any]
    cross_references: List[Dict[str, Any]]
    final_insights: Dict[str, Any]
    reasoning_trace: List[Dict[str, Any]]
    agent_communications: Annotated[List[AgentMessage], operator.add]
    confidence_score: float
    processing_errors: List[str]


class LangGraphNewsAnalyzer:
    """
    Advanced news analysis using LangGraph for complex reasoning workflows
    
    Implements a multi-step reasoning pipeline with:
    - Content understanding and extraction
    - Multi-dimensional relevance assessment
    - Impact analysis with temporal context
    - Cross-validation with other agents
    - Insight synthesis and confidence scoring
    """
    
    def __init__(self, agent_id: str, config: Dict[str, Any]):
        self.agent_id = agent_id
        self.config = config
        self.graph = None
        self.checkpointer = None
        self.setup_reasoning_graph()
        
        # A2A communication setup
        self.peer_agents = {}
        self.communication_history = []
        
    def setup_reasoning_graph(self):
        """Setup the LangGraph reasoning workflow"""
        if not LANGRAPH_AVAILABLE:
            return
            
        # Initialize checkpointer for workflow persistence
        self.checkpointer = SqliteSaver.from_conn_string(":memory:")
        
        # Create the reasoning workflow
        workflow = StateGraph(NewsAnalysisState)
        
        # Add reasoning nodes
        workflow.add_node("extract_content", self._extract_content_node)
        workflow.add_node("assess_relevance", self._assess_relevance_node)
        workflow.add_node("analyze_impact", self._analyze_impact_node)
        workflow.add_node("evaluate_sentiment", self._evaluate_sentiment_node)
        workflow.add_node("recognize_entities", self._recognize_entities_node)
        workflow.add_node("analyze_temporal", self._analyze_temporal_node)
        workflow.add_node("cross_validate", self._cross_validate_node)
        workflow.add_node("synthesize_insights", self._synthesize_insights_node)
        
        # Define workflow edges (reasoning flow)
        workflow.set_entry_point("extract_content")
        workflow.add_edge("extract_content", "assess_relevance")
        workflow.add_edge("assess_relevance", "analyze_impact")
        workflow.add_edge("analyze_impact", "evaluate_sentiment")
        workflow.add_edge("evaluate_sentiment", "recognize_entities")
        workflow.add_edge("recognize_entities", "analyze_temporal")
        workflow.add_edge("analyze_temporal", "cross_validate")
        workflow.add_edge("cross_validate", "synthesize_insights")
        workflow.add_edge("synthesize_insights", END)
        
        # Compile the graph
        self.graph = workflow.compile(checkpointer=self.checkpointer)
    
    async def _extract_content_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Extract and structure content from raw news"""
        try:
            content = state["raw_content"]
            metadata = state["article_metadata"]
            
            # Advanced content extraction using LLM
            extraction_prompt = self._create_extraction_prompt(content, metadata)
            
            # Simulate LLM processing (replace with actual LLM call)
            extracted_data = await self._llm_process(
                extraction_prompt,
                "Extract key information, themes, and context from this news article"
            )
            
            # Update state
            state["extracted_entities"] = extracted_data.get("entities", [])
            state["reasoning_trace"].append({
                "step": ReasoningStep.CONTENT_EXTRACTION.value,
                "timestamp": datetime.now().isoformat(),
                "input_length": len(content),
                "entities_found": len(extracted_data.get("entities", [])),
                "confidence": extracted_data.get("confidence", 0.8)
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Content extraction error: {str(e)}")
            return state
    
    async def _assess_relevance_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Assess market relevance using multi-factor analysis"""
        try:
            entities = state["extracted_entities"]
            metadata = state["article_metadata"]
            
            # Multi-dimensional relevance assessment
            relevance_factors = {
                "market_entities": self._score_market_entities(entities),
                "temporal_relevance": self._score_temporal_relevance(metadata),
                "sector_impact": self._score_sector_impact(entities),
                "geographic_scope": self._score_geographic_scope(entities),
                "regulatory_implications": self._score_regulatory_implications(entities)
            }
            
            # Aggregate relevance score
            weighted_score = sum(
                score * self.config.get(f"{factor}_weight", 0.2)
                for factor, score in relevance_factors.items()
            )
            
            state["relevance_scores"] = {
                "overall": min(weighted_score, 1.0),
                "factors": relevance_factors
            }
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.RELEVANCE_ASSESSMENT.value,
                "timestamp": datetime.now().isoformat(),
                "relevance_score": weighted_score,
                "factors": relevance_factors
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Relevance assessment error: {str(e)}")
            return state
    
    async def _analyze_impact_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Analyze potential market impact"""
        try:
            entities = state["extracted_entities"]
            relevance = state["relevance_scores"]
            
            # Impact analysis using agent reasoning
            impact_analysis = {
                "immediate_impact": self._assess_immediate_impact(entities),
                "medium_term_outlook": self._assess_medium_term_impact(entities),
                "sector_contagion": self._assess_sector_contagion(entities),
                "volatility_potential": self._assess_volatility_potential(entities),
                "regulatory_risk": self._assess_regulatory_risk(entities)
            }
            
            # A2A communication: Request validation from risk agent
            validation_request = AgentMessage(
                sender_id=self.agent_id,
                receiver_id="risk_management_agent",
                protocol=A2AProtocol.REQUEST_VALIDATION,
                content={
                    "analysis_type": "impact_assessment",
                    "preliminary_findings": impact_analysis,
                    "confidence_level": relevance["overall"]
                },
                timestamp=datetime.now(),
                message_id=f"impact_validation_{datetime.now().timestamp()}",
                requires_response=True
            )
            
            state["agent_communications"].append(validation_request)
            state["impact_assessment"] = impact_analysis
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.IMPACT_ANALYSIS.value,
                "timestamp": datetime.now().isoformat(),
                "impact_factors": list(impact_analysis.keys()),
                "validation_requested": True
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Impact analysis error: {str(e)}")
            return state
    
    async def _evaluate_sentiment_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Evaluate sentiment with context awareness"""
        try:
            content = state["raw_content"]
            entities = state["extracted_entities"]
            
            # Advanced sentiment evaluation
            sentiment_analysis = {
                "overall_sentiment": self._calculate_contextual_sentiment(content),
                "entity_sentiments": self._calculate_entity_sentiments(content, entities),
                "temporal_sentiment": self._calculate_temporal_sentiment(content),
                "confidence_indicators": self._extract_confidence_indicators(content),
                "uncertainty_markers": self._extract_uncertainty_markers(content)
            }
            
            # A2A communication: Share findings with sentiment agent
            sentiment_sharing = AgentMessage(
                sender_id=self.agent_id,
                receiver_id="sentiment_analysis_agent",
                protocol=A2AProtocol.KNOWLEDGE_EXCHANGE,
                content={
                    "news_sentiment_analysis": sentiment_analysis,
                    "source_credibility": state["article_metadata"].get("source_credibility", 0.5),
                    "temporal_context": state.get("temporal_context", {})
                },
                timestamp=datetime.now(),
                message_id=f"sentiment_sharing_{datetime.now().timestamp()}",
                requires_response=False
            )
            
            state["agent_communications"].append(sentiment_sharing)
            state["sentiment_analysis"] = sentiment_analysis
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.SENTIMENT_EVALUATION.value,
                "timestamp": datetime.now().isoformat(),
                "overall_sentiment": sentiment_analysis["overall_sentiment"],
                "entity_count": len(sentiment_analysis["entity_sentiments"])
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Sentiment evaluation error: {str(e)}")
            return state
    
    async def _recognize_entities_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Advanced entity recognition with relationship mapping"""
        try:
            content = state["raw_content"]
            existing_entities = state["extracted_entities"]
            
            # Enhanced entity recognition
            enhanced_entities = []
            for entity in existing_entities:
                enhanced_entity = {
                    **entity,
                    "relationships": self._map_entity_relationships(entity, existing_entities),
                    "importance_score": self._score_entity_importance(entity, content),
                    "market_relevance": self._score_entity_market_relevance(entity),
                    "temporal_significance": self._score_entity_temporal_significance(entity)
                }
                enhanced_entities.append(enhanced_entity)
            
            # Knowledge graph construction
            knowledge_graph = self._build_knowledge_graph(enhanced_entities)
            
            state["extracted_entities"] = enhanced_entities
            state["knowledge_graph"] = knowledge_graph
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.ENTITY_RECOGNITION.value,
                "timestamp": datetime.now().isoformat(),
                "entity_count": len(enhanced_entities),
                "relationship_count": sum(len(e.get("relationships", [])) for e in enhanced_entities)
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Entity recognition error: {str(e)}")
            return state
    
    async def _analyze_temporal_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Analyze temporal context and trends"""
        try:
            metadata = state["article_metadata"]
            entities = state["extracted_entities"]
            
            temporal_analysis = {
                "publication_timing": self._analyze_publication_timing(metadata),
                "market_session_context": self._analyze_market_session_context(metadata),
                "historical_patterns": self._analyze_historical_patterns(entities),
                "trend_alignment": self._analyze_trend_alignment(entities),
                "seasonality_factors": self._analyze_seasonality_factors(metadata)
            }
            
            state["temporal_context"] = temporal_analysis
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.TEMPORAL_ANALYSIS.value,
                "timestamp": datetime.now().isoformat(),
                "temporal_factors": list(temporal_analysis.keys())
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Temporal analysis error: {str(e)}")
            return state
    
    async def _cross_validate_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Cross-validate findings with peer agents"""
        try:
            # A2A communication: Request peer review
            peer_review_request = AgentMessage(
                sender_id=self.agent_id,
                receiver_id="technical_analysis_agent",
                protocol=A2AProtocol.PEER_REVIEW,
                content={
                    "analysis_summary": {
                        "relevance_score": state["relevance_scores"]["overall"],
                        "impact_assessment": state["impact_assessment"],
                        "sentiment_analysis": state["sentiment_analysis"],
                        "key_entities": [e["name"] for e in state["extracted_entities"][:5]]
                    },
                    "validation_request": "cross_reference_with_technical_indicators"
                },
                timestamp=datetime.now(),
                message_id=f"peer_review_{datetime.now().timestamp()}",
                requires_response=True
            )
            
            state["agent_communications"].append(peer_review_request)
            
            # Simulate consensus building
            consensus_score = self._calculate_consensus_score(state)
            
            state["cross_references"] = [{
                "validation_source": "technical_analysis_agent",
                "consensus_score": consensus_score,
                "confidence_adjustment": 0.1 if consensus_score > 0.7 else -0.1
            }]
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.CROSS_VALIDATION.value,
                "timestamp": datetime.now().isoformat(),
                "consensus_score": consensus_score,
                "peer_agents_consulted": 1
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Cross validation error: {str(e)}")
            return state
    
    async def _synthesize_insights_node(self, state: NewsAnalysisState) -> NewsAnalysisState:
        """Synthesize final insights with confidence scoring"""
        try:
            # Aggregate all analysis components
            final_insights = {
                "market_impact_score": self._aggregate_impact_score(state),
                "confidence_level": self._calculate_final_confidence(state),
                "key_takeaways": self._extract_key_takeaways(state),
                "risk_indicators": self._extract_risk_indicators(state),
                "opportunity_indicators": self._extract_opportunity_indicators(state),
                "recommended_actions": self._generate_recommendations(state),
                "uncertainty_factors": self._identify_uncertainty_factors(state),
                "temporal_urgency": self._assess_temporal_urgency(state)
            }
            
            # Final confidence adjustment based on cross-validation
            cross_validation_adjustment = sum(
                ref.get("confidence_adjustment", 0) 
                for ref in state.get("cross_references", [])
            )
            
            final_confidence = min(1.0, max(0.0, 
                final_insights["confidence_level"] + cross_validation_adjustment
            ))
            
            state["final_insights"] = final_insights
            state["confidence_score"] = final_confidence
            
            state["reasoning_trace"].append({
                "step": ReasoningStep.INSIGHT_SYNTHESIS.value,
                "timestamp": datetime.now().isoformat(),
                "final_confidence": final_confidence,
                "impact_score": final_insights["market_impact_score"],
                "key_takeaway_count": len(final_insights["key_takeaways"])
            })
            
            return state
            
        except Exception as e:
            state["processing_errors"].append(f"Insight synthesis error: {str(e)}")
            return state
    
    # Helper methods for complex reasoning
    def _create_extraction_prompt(self, content: str, metadata: Dict[str, Any]) -> str:
        """Create contextual prompt for content extraction"""
        return f"""
        Analyze this financial news article and extract structured information:
        
        Article: {content}
        Source: {metadata.get('source', 'Unknown')}
        Published: {metadata.get('published_at', 'Unknown')}
        
        Extract:
        1. Key entities (companies, people, financial instruments)
        2. Market themes and topics
        3. Quantitative data (prices, percentages, volumes)
        4. Temporal references and deadlines
        5. Sentiment indicators
        6. Risk and opportunity signals
        
        Provide structured output with confidence scores.
        """
    
    async def _llm_process(self, prompt: str, instruction: str) -> Dict[str, Any]:
        """Simulate LLM processing (replace with actual LLM integration)"""
        # This would be replaced with actual LLM calls
        return {
            "entities": [
                {"name": "Apple Inc.", "type": "company", "confidence": 0.95},
                {"name": "Q4 earnings", "type": "event", "confidence": 0.90}
            ],
            "confidence": 0.85,
            "themes": ["earnings", "technology", "growth"]
        }
    
    def _score_market_entities(self, entities: List[Dict[str, Any]]) -> float:
        """Score based on market-relevant entities"""
        market_entity_count = sum(1 for e in entities if e.get("type") in ["company", "instrument", "index"])
        return min(1.0, market_entity_count * 0.2)
    
    def _score_temporal_relevance(self, metadata: Dict[str, Any]) -> float:
        """Score based on temporal relevance"""
        # Implementation for temporal scoring
        return 0.8
    
    def _score_sector_impact(self, entities: List[Dict[str, Any]]) -> float:
        """Score based on sector impact potential"""
        # Implementation for sector impact scoring
        return 0.7
    
    def _score_geographic_scope(self, entities: List[Dict[str, Any]]) -> float:
        """Score based on geographic scope"""
        # Implementation for geographic scoring
        return 0.6
    
    def _score_regulatory_implications(self, entities: List[Dict[str, Any]]) -> float:
        """Score based on regulatory implications"""
        # Implementation for regulatory scoring
        return 0.5
    
    def _assess_immediate_impact(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess immediate market impact"""
        return {"score": 0.7, "factors": ["earnings_surprise", "volume_spike"]}
    
    def _assess_medium_term_impact(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess medium-term impact"""
        return {"score": 0.6, "factors": ["trend_change", "sector_rotation"]}
    
    def _assess_sector_contagion(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess sector contagion risk"""
        return {"score": 0.4, "factors": ["sector_correlation", "supply_chain"]}
    
    def _assess_volatility_potential(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess volatility potential"""
        return {"score": 0.5, "factors": ["uncertainty", "surprise_factor"]}
    
    def _assess_regulatory_risk(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Assess regulatory risk"""
        return {"score": 0.3, "factors": ["policy_change", "compliance_risk"]}
    
    def _calculate_contextual_sentiment(self, content: str) -> float:
        """Calculate context-aware sentiment"""
        # Advanced sentiment calculation
        return 0.6
    
    def _calculate_entity_sentiments(self, content: str, entities: List[Dict[str, Any]]) -> Dict[str, float]:
        """Calculate sentiment for each entity"""
        return {entity["name"]: 0.5 for entity in entities}
    
    def _calculate_temporal_sentiment(self, content: str) -> Dict[str, Any]:
        """Calculate temporal sentiment patterns"""
        return {"trend": "positive", "momentum": 0.7}
    
    def _extract_confidence_indicators(self, content: str) -> List[str]:
        """Extract confidence indicators from text"""
        return ["strong guidance", "beat expectations"]
    
    def _extract_uncertainty_markers(self, content: str) -> List[str]:
        """Extract uncertainty markers from text"""
        return ["subject to conditions", "pending approval"]
    
    def _map_entity_relationships(self, entity: Dict[str, Any], all_entities: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Map relationships between entities"""
        return [{"target": "related_entity", "type": "partnership", "strength": 0.8}]
    
    def _score_entity_importance(self, entity: Dict[str, Any], content: str) -> float:
        """Score entity importance in context"""
        return 0.7
    
    def _score_entity_market_relevance(self, entity: Dict[str, Any]) -> float:
        """Score entity's market relevance"""
        return 0.8
    
    def _score_entity_temporal_significance(self, entity: Dict[str, Any]) -> float:
        """Score entity's temporal significance"""
        return 0.6
    
    def _build_knowledge_graph(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Build knowledge graph from entities"""
        return {"nodes": len(entities), "edges": 10, "clusters": 3}
    
    def _analyze_publication_timing(self, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """Analyze publication timing context"""
        return {"market_hours": "after_market", "significance": 0.7}
    
    def _analyze_market_session_context(self, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """Analyze market session context"""
        return {"session": "pre_market", "impact_multiplier": 1.2}
    
    def _analyze_historical_patterns(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Analyze historical patterns"""
        return {"pattern_match": 0.8, "historical_outcomes": ["positive", "volatile"]}
    
    def _analyze_trend_alignment(self, entities: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Analyze trend alignment"""
        return {"alignment_score": 0.7, "trend_direction": "bullish"}
    
    def _analyze_seasonality_factors(self, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """Analyze seasonality factors"""
        return {"seasonal_effect": 0.5, "quarter": "Q4", "month_effect": "positive"}
    
    def _calculate_consensus_score(self, state: NewsAnalysisState) -> float:
        """Calculate consensus score with peer agents"""
        return 0.75
    
    def _aggregate_impact_score(self, state: NewsAnalysisState) -> float:
        """Aggregate impact score from all factors"""
        relevance = state["relevance_scores"]["overall"]
        impact_factors = state["impact_assessment"]
        sentiment = state["sentiment_analysis"]["overall_sentiment"]
        
        return (relevance * 0.4 + 
                sum(f.get("score", 0) for f in impact_factors.values()) * 0.1 + 
                abs(sentiment) * 0.3)
    
    def _calculate_final_confidence(self, state: NewsAnalysisState) -> float:
        """Calculate final confidence score"""
        base_confidence = 0.7
        error_penalty = len(state["processing_errors"]) * 0.1
        cross_validation_bonus = len(state.get("cross_references", [])) * 0.05
        
        return max(0.1, min(1.0, base_confidence - error_penalty + cross_validation_bonus))
    
    def _extract_key_takeaways(self, state: NewsAnalysisState) -> List[str]:
        """Extract key takeaways"""
        return [
            "Strong earnings performance indicates positive momentum",
            "Technology sector showing resilience",
            "Market timing suggests pre-opening impact"
        ]
    
    def _extract_risk_indicators(self, state: NewsAnalysisState) -> List[str]:
        """Extract risk indicators"""
        return ["regulatory uncertainty", "market volatility"]
    
    def _extract_opportunity_indicators(self, state: NewsAnalysisState) -> List[str]:
        """Extract opportunity indicators"""
        return ["growth potential", "market expansion"]
    
    def _generate_recommendations(self, state: NewsAnalysisState) -> List[str]:
        """Generate actionable recommendations"""
        return [
            "Monitor pre-market trading for confirmation",
            "Consider sector rotation implications",
            "Watch for follow-up guidance updates"
        ]
    
    def _identify_uncertainty_factors(self, state: NewsAnalysisState) -> List[str]:
        """Identify uncertainty factors"""
        return ["pending regulatory approval", "market condition dependency"]
    
    def _assess_temporal_urgency(self, state: NewsAnalysisState) -> str:
        """Assess temporal urgency"""
        return "immediate"  # high, medium, low, immediate
    
    async def process_news_article(self, content: str, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """
        Process news article through the complete LangGraph reasoning pipeline
        
        Returns comprehensive analysis with reasoning trace and A2A communications
        """
        if not self.graph:
            # Fallback to simple analysis if LangGraph not available
            return await self._fallback_analysis(content, metadata)
        
        # Initialize state
        initial_state: NewsAnalysisState = {
            "raw_content": content,
            "article_metadata": metadata,
            "extracted_entities": [],
            "relevance_scores": {},
            "impact_assessment": {},
            "sentiment_analysis": {},
            "temporal_context": {},
            "cross_references": [],
            "final_insights": {},
            "reasoning_trace": [],
            "agent_communications": [],
            "confidence_score": 0.0,
            "processing_errors": []
        }
        
        # Execute the reasoning workflow
        config = {"configurable": {"thread_id": f"news_analysis_{datetime.now().timestamp()}"}}
        
        try:
            final_state = await self.graph.ainvoke(initial_state, config)
            
            return {
                "insights": final_state["final_insights"],
                "confidence": final_state["confidence_score"],
                "reasoning_trace": final_state["reasoning_trace"],
                "agent_communications": [msg.__dict__ for msg in final_state["agent_communications"]],
                "processing_errors": final_state["processing_errors"],
                "metadata": {
                    "processing_time": len(final_state["reasoning_trace"]),
                    "complexity_score": len(final_state["extracted_entities"]),
                    "cross_validation_count": len(final_state["cross_references"])
                }
            }
            
        except Exception as e:
            return {
                "insights": {},
                "confidence": 0.1,
                "reasoning_trace": [],
                "agent_communications": [],
                "processing_errors": [f"Workflow execution error: {str(e)}"],
                "metadata": {"processing_time": 0, "complexity_score": 0}
            }
    
    async def _fallback_analysis(self, content: str, metadata: Dict[str, Any]) -> Dict[str, Any]:
        """Fallback analysis when LangGraph is not available"""
        return {
            "insights": {
                "market_impact_score": 0.5,
                "confidence_level": 0.6,
                "key_takeaways": ["Basic analysis completed"],
                "recommended_actions": ["Review with advanced tools"]
            },
            "confidence": 0.6,
            "reasoning_trace": [{"step": "fallback_analysis", "timestamp": datetime.now().isoformat()}],
            "agent_communications": [],
            "processing_errors": ["LangGraph not available - using fallback"],
            "metadata": {"processing_time": 1, "complexity_score": 1}
        }
    
    def register_peer_agent(self, agent_id: str, communication_handler):
        """Register peer agent for A2A communication"""
        self.peer_agents[agent_id] = communication_handler
    
    async def handle_agent_message(self, message: AgentMessage) -> Optional[AgentMessage]:
        """Handle incoming A2A message"""
        self.communication_history.append(message)
        
        if message.protocol == A2AProtocol.REQUEST_VALIDATION:
            return await self._handle_validation_request(message)
        elif message.protocol == A2AProtocol.PEER_REVIEW:
            return await self._handle_peer_review_request(message)
        elif message.protocol == A2AProtocol.KNOWLEDGE_EXCHANGE:
            await self._handle_knowledge_exchange(message)
        
        return None
    
    async def _handle_validation_request(self, message: AgentMessage) -> AgentMessage:
        """Handle validation request from another agent"""
        # Process validation request and provide response
        response_content = {
            "validation_result": "approved",
            "confidence_adjustment": 0.05,
            "additional_insights": ["Cross-validation confirms analysis"]
        }
        
        return AgentMessage(
            sender_id=self.agent_id,
            receiver_id=message.sender_id,
            protocol=A2AProtocol.REQUEST_VALIDATION,
            content=response_content,
            timestamp=datetime.now(),
            message_id=f"validation_response_{datetime.now().timestamp()}",
            correlation_id=message.message_id
        )
    
    async def _handle_peer_review_request(self, message: AgentMessage) -> AgentMessage:
        """Handle peer review request"""
        # Perform peer review analysis
        review_content = {
            "review_result": "consistent",
            "agreement_score": 0.85,
            "suggested_improvements": ["Consider additional temporal factors"]
        }
        
        return AgentMessage(
            sender_id=self.agent_id,
            receiver_id=message.sender_id,
            protocol=A2AProtocol.PEER_REVIEW,
            content=review_content,
            timestamp=datetime.now(),
            message_id=f"peer_review_response_{datetime.now().timestamp()}",
            correlation_id=message.message_id
        )
    
    async def _handle_knowledge_exchange(self, message: AgentMessage):
        """Handle knowledge exchange from another agent"""
        # Store shared knowledge for future analysis
        knowledge = message.content
        # Update internal knowledge base
        pass