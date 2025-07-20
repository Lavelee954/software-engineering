"""
Agent-to-Agent (A2A) Communication Framework

Advanced multi-agent communication system enabling:
- Structured agent protocols and message formats
- Asynchronous agent collaboration
- Knowledge sharing and consensus building
- Distributed decision making
- Agent reputation and trust management
"""

import asyncio
import json
import uuid
from typing import Dict, List, Optional, Any, Callable, Set
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass, field
from abc import ABC, abstractmethod
import logging


class MessagePriority(Enum):
    """Message priority levels"""
    CRITICAL = 1
    HIGH = 2  
    MEDIUM = 3
    LOW = 4
    BACKGROUND = 5


class MessageType(Enum):
    """Types of inter-agent messages"""
    REQUEST = "request"
    RESPONSE = "response"
    NOTIFICATION = "notification"
    BROADCAST = "broadcast"
    QUERY = "query"
    COMMAND = "command"
    EVENT = "event"
    HEARTBEAT = "heartbeat"


class AgentCapability(Enum):
    """Agent capabilities for discovery"""
    TECHNICAL_ANALYSIS = "technical_analysis"
    NEWS_ANALYSIS = "news_analysis"
    SENTIMENT_ANALYSIS = "sentiment_analysis"
    RISK_MANAGEMENT = "risk_management"
    PORTFOLIO_MANAGEMENT = "portfolio_management"
    STRATEGY_EXECUTION = "strategy_execution"
    MARKET_DATA = "market_data"
    EXECUTION = "execution"
    BACKTESTING = "backtesting"
    MACRO_ANALYSIS = "macro_analysis"


class CollaborationPattern(Enum):
    """Agent collaboration patterns"""
    REQUEST_RESPONSE = "request_response"
    PUBLISH_SUBSCRIBE = "publish_subscribe"
    CONSENSUS_BUILDING = "consensus_building"
    WORKFLOW_ORCHESTRATION = "workflow_orchestration"
    PEER_REVIEW = "peer_review"
    KNOWLEDGE_SHARING = "knowledge_sharing"
    COMPETITIVE_ANALYSIS = "competitive_analysis"
    ENSEMBLE_DECISION = "ensemble_decision"


@dataclass
class AgentMessage:
    """Structured inter-agent message"""
    message_id: str
    sender_id: str
    receiver_id: str
    message_type: MessageType
    priority: MessagePriority
    content: Dict[str, Any]
    timestamp: datetime
    ttl: Optional[datetime] = None
    correlation_id: Optional[str] = None
    requires_response: bool = False
    response_timeout: Optional[int] = None  # seconds
    routing_path: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentProfile:
    """Agent profile for registration and discovery"""
    agent_id: str
    agent_type: str
    capabilities: List[AgentCapability]
    service_level: float  # 0.0 to 1.0
    reputation_score: float  # 0.0 to 1.0
    load_factor: float  # 0.0 to 1.0 (current load)
    max_concurrent_requests: int
    supported_patterns: List[CollaborationPattern]
    metadata: Dict[str, Any] = field(default_factory=dict)
    last_heartbeat: Optional[datetime] = None
    status: str = "active"  # active, busy, offline, maintenance


class AgentRegistry:
    """Central registry for agent discovery and management"""
    
    def __init__(self):
        self.agents: Dict[str, AgentProfile] = {}
        self.capabilities_index: Dict[AgentCapability, Set[str]] = {}
        self.message_routes: Dict[str, List[str]] = {}
        self.logger = logging.getLogger(__name__)
    
    def register_agent(self, profile: AgentProfile) -> bool:
        """Register an agent in the system"""
        try:
            self.agents[profile.agent_id] = profile
            
            # Update capabilities index
            for capability in profile.capabilities:
                if capability not in self.capabilities_index:
                    self.capabilities_index[capability] = set()
                self.capabilities_index[capability].add(profile.agent_id)
            
            self.logger.info(f"Registered agent: {profile.agent_id} with capabilities: {profile.capabilities}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to register agent {profile.agent_id}: {e}")
            return False
    
    def unregister_agent(self, agent_id: str) -> bool:
        """Unregister an agent from the system"""
        try:
            if agent_id in self.agents:
                profile = self.agents[agent_id]
                
                # Remove from capabilities index
                for capability in profile.capabilities:
                    if capability in self.capabilities_index:
                        self.capabilities_index[capability].discard(agent_id)
                        if not self.capabilities_index[capability]:
                            del self.capabilities_index[capability]
                
                del self.agents[agent_id]
                self.logger.info(f"Unregistered agent: {agent_id}")
                return True
                
        except Exception as e:
            self.logger.error(f"Failed to unregister agent {agent_id}: {e}")
            
        return False
    
    def find_agents_by_capability(self, capability: AgentCapability) -> List[AgentProfile]:
        """Find agents with specific capability"""
        agent_ids = self.capabilities_index.get(capability, set())
        return [self.agents[agent_id] for agent_id in agent_ids if agent_id in self.agents]
    
    def get_best_agent_for_capability(self, capability: AgentCapability) -> Optional[AgentProfile]:
        """Get the best available agent for a capability"""
        candidates = self.find_agents_by_capability(capability)
        
        if not candidates:
            return None
        
        # Score agents based on availability, reputation, and load
        def score_agent(agent: AgentProfile) -> float:
            if agent.status != "active":
                return 0.0
            
            availability_score = max(0.0, 1.0 - agent.load_factor)
            reputation_weight = 0.4
            service_weight = 0.3
            availability_weight = 0.3
            
            return (agent.reputation_score * reputation_weight + 
                   agent.service_level * service_weight +
                   availability_score * availability_weight)
        
        return max(candidates, key=score_agent)
    
    def update_agent_status(self, agent_id: str, status: str, load_factor: Optional[float] = None):
        """Update agent status and load"""
        if agent_id in self.agents:
            self.agents[agent_id].status = status
            self.agents[agent_id].last_heartbeat = datetime.now()
            
            if load_factor is not None:
                self.agents[agent_id].load_factor = load_factor
    
    def cleanup_stale_agents(self, timeout_minutes: int = 5):
        """Remove agents that haven't sent heartbeat"""
        cutoff_time = datetime.now() - timedelta(minutes=timeout_minutes)
        stale_agents = []
        
        for agent_id, profile in self.agents.items():
            if profile.last_heartbeat and profile.last_heartbeat < cutoff_time:
                stale_agents.append(agent_id)
        
        for agent_id in stale_agents:
            self.unregister_agent(agent_id)
            self.logger.warning(f"Removed stale agent: {agent_id}")


class MessageRouter:
    """Routes messages between agents with intelligent routing"""
    
    def __init__(self, registry: AgentRegistry):
        self.registry = registry
        self.routing_table: Dict[str, str] = {}  # receiver_id -> handler_endpoint
        self.message_handlers: Dict[str, Callable] = {}
        self.pending_responses: Dict[str, asyncio.Future] = {}
        self.logger = logging.getLogger(__name__)
    
    def register_handler(self, agent_id: str, handler: Callable):
        """Register message handler for an agent"""
        self.message_handlers[agent_id] = handler
    
    async def route_message(self, message: AgentMessage) -> bool:
        """Route message to appropriate handler"""
        try:
            # Add routing information
            message.routing_path.append(message.sender_id)
            
            # Check if receiver exists
            if message.receiver_id not in self.registry.agents:
                self.logger.error(f"Unknown receiver: {message.receiver_id}")
                return False
            
            # Check TTL
            if message.ttl and datetime.now() > message.ttl:
                self.logger.warning(f"Message expired: {message.message_id}")
                return False
            
            # Route to handler
            if message.receiver_id in self.message_handlers:
                handler = self.message_handlers[message.receiver_id]
                
                # Handle response tracking
                if message.requires_response:
                    self._track_pending_response(message)
                
                # Async message delivery
                asyncio.create_task(self._deliver_message(handler, message))
                return True
            else:
                self.logger.error(f"No handler registered for agent: {message.receiver_id}")
                return False
                
        except Exception as e:
            self.logger.error(f"Message routing failed: {e}")
            return False
    
    async def _deliver_message(self, handler: Callable, message: AgentMessage):
        """Deliver message to handler"""
        try:
            if asyncio.iscoroutinefunction(handler):
                response = await handler(message)
            else:
                response = handler(message)
            
            # Handle response message
            if response and isinstance(response, AgentMessage):
                await self.route_message(response)
                
        except Exception as e:
            self.logger.error(f"Message delivery failed: {e}")
    
    def _track_pending_response(self, message: AgentMessage):
        """Track pending response for timeout handling"""
        if message.response_timeout:
            future = asyncio.Future()
            self.pending_responses[message.message_id] = future
            
            # Set timeout
            asyncio.create_task(self._handle_response_timeout(message.message_id, message.response_timeout))
    
    async def _handle_response_timeout(self, message_id: str, timeout: int):
        """Handle response timeout"""
        await asyncio.sleep(timeout)
        
        if message_id in self.pending_responses:
            future = self.pending_responses.pop(message_id)
            if not future.done():
                future.set_exception(TimeoutError(f"Response timeout for message: {message_id}"))
    
    async def wait_for_response(self, message_id: str) -> AgentMessage:
        """Wait for response to a specific message"""
        if message_id in self.pending_responses:
            return await self.pending_responses[message_id]
        else:
            raise ValueError(f"No pending response tracked for message: {message_id}")


class CollaborationOrchestrator:
    """Orchestrates complex multi-agent collaborations"""
    
    def __init__(self, registry: AgentRegistry, router: MessageRouter):
        self.registry = registry
        self.router = router
        self.active_collaborations: Dict[str, Dict[str, Any]] = {}
        self.logger = logging.getLogger(__name__)
    
    async def initiate_consensus_building(
        self, 
        topic: str, 
        participants: List[str], 
        decision_data: Dict[str, Any],
        timeout: int = 60
    ) -> Dict[str, Any]:
        """Initiate consensus building among agents"""
        
        collaboration_id = str(uuid.uuid4())
        
        self.active_collaborations[collaboration_id] = {
            "type": CollaborationPattern.CONSENSUS_BUILDING,
            "topic": topic,
            "participants": participants,
            "responses": {},
            "started_at": datetime.now(),
            "timeout": timeout
        }
        
        # Send consensus request to all participants
        consensus_messages = []
        for participant in participants:
            message = AgentMessage(
                message_id=str(uuid.uuid4()),
                sender_id="orchestrator",
                receiver_id=participant,
                message_type=MessageType.REQUEST,
                priority=MessagePriority.HIGH,
                content={
                    "collaboration_id": collaboration_id,
                    "type": "consensus_request",
                    "topic": topic,
                    "data": decision_data,
                    "participants": participants
                },
                timestamp=datetime.now(),
                ttl=datetime.now() + timedelta(seconds=timeout),
                requires_response=True,
                response_timeout=timeout
            )
            consensus_messages.append(message)
            await self.router.route_message(message)
        
        # Wait for responses
        return await self._collect_consensus_responses(collaboration_id, timeout)
    
    async def _collect_consensus_responses(self, collaboration_id: str, timeout: int) -> Dict[str, Any]:
        """Collect and analyze consensus responses"""
        collaboration = self.active_collaborations[collaboration_id]
        
        start_time = datetime.now()
        while (datetime.now() - start_time).total_seconds() < timeout:
            if len(collaboration["responses"]) >= len(collaboration["participants"]):
                break
            await asyncio.sleep(0.1)
        
        # Analyze consensus
        responses = collaboration["responses"]
        consensus_result = self._analyze_consensus(responses)
        
        # Cleanup
        del self.active_collaborations[collaboration_id]
        
        return {
            "collaboration_id": collaboration_id,
            "consensus_achieved": consensus_result["consensus_achieved"],
            "agreement_score": consensus_result["agreement_score"],
            "responses": responses,
            "final_decision": consensus_result["final_decision"],
            "dissenting_agents": consensus_result["dissenting_agents"]
        }
    
    def _analyze_consensus(self, responses: Dict[str, Any]) -> Dict[str, Any]:
        """Analyze consensus from agent responses"""
        if not responses:
            return {
                "consensus_achieved": False,
                "agreement_score": 0.0,
                "final_decision": None,
                "dissenting_agents": []
            }
        
        # Simple majority consensus (can be enhanced with weighted voting)
        decisions = [r.get("decision") for r in responses.values() if "decision" in r]
        
        if not decisions:
            return {
                "consensus_achieved": False,
                "agreement_score": 0.0,
                "final_decision": None,
                "dissenting_agents": list(responses.keys())
            }
        
        # Count votes
        vote_counts = {}
        for decision in decisions:
            vote_counts[decision] = vote_counts.get(decision, 0) + 1
        
        # Find majority decision
        majority_decision = max(vote_counts.items(), key=lambda x: x[1])
        majority_count = majority_decision[1]
        total_responses = len(decisions)
        
        agreement_score = majority_count / total_responses
        consensus_achieved = agreement_score >= 0.6  # 60% threshold
        
        # Find dissenting agents
        dissenting_agents = [
            agent_id for agent_id, response in responses.items()
            if response.get("decision") != majority_decision[0]
        ]
        
        return {
            "consensus_achieved": consensus_achieved,
            "agreement_score": agreement_score,
            "final_decision": majority_decision[0],
            "dissenting_agents": dissenting_agents
        }
    
    async def orchestrate_peer_review(
        self, 
        subject_agent: str, 
        analysis_data: Dict[str, Any],
        review_criteria: List[str],
        num_reviewers: int = 3
    ) -> Dict[str, Any]:
        """Orchestrate peer review process"""
        
        # Find eligible reviewers (exclude subject agent)
        all_agents = list(self.registry.agents.keys())
        eligible_reviewers = [a for a in all_agents if a != subject_agent]
        
        if len(eligible_reviewers) < num_reviewers:
            num_reviewers = len(eligible_reviewers)
        
        # Select best reviewers based on reputation and availability
        selected_reviewers = sorted(
            eligible_reviewers,
            key=lambda x: (
                self.registry.agents[x].reputation_score,
                -self.registry.agents[x].load_factor
            ),
            reverse=True
        )[:num_reviewers]
        
        # Initiate peer review
        review_id = str(uuid.uuid4())
        review_messages = []
        
        for reviewer in selected_reviewers:
            message = AgentMessage(
                message_id=str(uuid.uuid4()),
                sender_id="orchestrator",
                receiver_id=reviewer,
                message_type=MessageType.REQUEST,
                priority=MessagePriority.MEDIUM,
                content={
                    "review_id": review_id,
                    "type": "peer_review_request",
                    "subject_agent": subject_agent,
                    "analysis_data": analysis_data,
                    "review_criteria": review_criteria
                },
                timestamp=datetime.now(),
                requires_response=True,
                response_timeout=120  # 2 minutes
            )
            review_messages.append(message)
            await self.router.route_message(message)
        
        # Collect review responses
        reviews = {}
        for message in review_messages:
            try:
                response = await self.router.wait_for_response(message.message_id)
                reviews[response.sender_id] = response.content
            except TimeoutError:
                self.logger.warning(f"Peer review timeout for reviewer: {message.receiver_id}")
        
        # Aggregate review results
        return self._aggregate_peer_reviews(review_id, reviews, review_criteria)
    
    def _aggregate_peer_reviews(
        self, 
        review_id: str, 
        reviews: Dict[str, Any], 
        criteria: List[str]
    ) -> Dict[str, Any]:
        """Aggregate peer review results"""
        
        if not reviews:
            return {
                "review_id": review_id,
                "overall_score": 0.0,
                "criteria_scores": {},
                "recommendations": [],
                "consensus_level": 0.0
            }
        
        # Aggregate scores by criteria
        criteria_scores = {}
        for criterion in criteria:
            scores = [
                review.get("scores", {}).get(criterion, 0.0) 
                for review in reviews.values()
            ]
            criteria_scores[criterion] = sum(scores) / len(scores) if scores else 0.0
        
        # Calculate overall score
        overall_score = sum(criteria_scores.values()) / len(criteria_scores) if criteria_scores else 0.0
        
        # Aggregate recommendations
        all_recommendations = []
        for review in reviews.values():
            all_recommendations.extend(review.get("recommendations", []))
        
        # Calculate consensus level
        score_variance = sum(
            (score - overall_score) ** 2 
            for score in criteria_scores.values()
        ) / len(criteria_scores) if criteria_scores else 0.0
        
        consensus_level = max(0.0, 1.0 - score_variance)
        
        return {
            "review_id": review_id,
            "overall_score": overall_score,
            "criteria_scores": criteria_scores,
            "recommendations": list(set(all_recommendations)),  # Remove duplicates
            "consensus_level": consensus_level,
            "reviewer_count": len(reviews)
        }
    
    def handle_consensus_response(self, message: AgentMessage):
        """Handle consensus response from an agent"""
        content = message.content
        collaboration_id = content.get("collaboration_id")
        
        if collaboration_id in self.active_collaborations:
            collaboration = self.active_collaborations[collaboration_id]
            collaboration["responses"][message.sender_id] = content


class A2ACommunicationManager:
    """Main manager for Agent-to-Agent communication"""
    
    def __init__(self):
        self.registry = AgentRegistry()
        self.router = MessageRouter(self.registry)
        self.orchestrator = CollaborationOrchestrator(self.registry, self.router)
        self.logger = logging.getLogger(__name__)
        
        # Background tasks
        self._cleanup_task = None
        self._heartbeat_task = None
    
    async def start(self):
        """Start the A2A communication system"""
        self._cleanup_task = asyncio.create_task(self._periodic_cleanup())
        self.logger.info("A2A Communication Manager started")
    
    async def stop(self):
        """Stop the A2A communication system"""
        if self._cleanup_task:
            self._cleanup_task.cancel()
        
        if self._heartbeat_task:
            self._heartbeat_task.cancel()
        
        self.logger.info("A2A Communication Manager stopped")
    
    async def _periodic_cleanup(self):
        """Periodic cleanup of stale agents and expired messages"""
        while True:
            try:
                await asyncio.sleep(60)  # Cleanup every minute
                self.registry.cleanup_stale_agents()
            except asyncio.CancelledError:
                break
            except Exception as e:
                self.logger.error(f"Cleanup error: {e}")
    
    def create_agent_message(
        self,
        sender_id: str,
        receiver_id: str,
        content: Dict[str, Any],
        message_type: MessageType = MessageType.REQUEST,
        priority: MessagePriority = MessagePriority.MEDIUM,
        requires_response: bool = False,
        response_timeout: Optional[int] = None
    ) -> AgentMessage:
        """Create a standardized agent message"""
        
        return AgentMessage(
            message_id=str(uuid.uuid4()),
            sender_id=sender_id,
            receiver_id=receiver_id,
            message_type=message_type,
            priority=priority,
            content=content,
            timestamp=datetime.now(),
            requires_response=requires_response,
            response_timeout=response_timeout
        )
    
    async def send_message(self, message: AgentMessage) -> bool:
        """Send message through the communication system"""
        return await self.router.route_message(message)
    
    async def broadcast_message(
        self,
        sender_id: str,
        content: Dict[str, Any],
        capability_filter: Optional[AgentCapability] = None
    ) -> int:
        """Broadcast message to multiple agents"""
        
        if capability_filter:
            targets = self.registry.find_agents_by_capability(capability_filter)
            target_ids = [agent.agent_id for agent in targets]
        else:
            target_ids = list(self.registry.agents.keys())
        
        # Remove sender from targets
        target_ids = [tid for tid in target_ids if tid != sender_id]
        
        success_count = 0
        for target_id in target_ids:
            message = self.create_agent_message(
                sender_id=sender_id,
                receiver_id=target_id,
                content=content,
                message_type=MessageType.BROADCAST,
                priority=MessagePriority.LOW
            )
            
            if await self.send_message(message):
                success_count += 1
        
        return success_count
    
    def get_system_stats(self) -> Dict[str, Any]:
        """Get A2A communication system statistics"""
        return {
            "registered_agents": len(self.registry.agents),
            "active_agents": len([a for a in self.registry.agents.values() if a.status == "active"]),
            "capabilities_coverage": len(self.registry.capabilities_index),
            "pending_responses": len(self.router.pending_responses),
            "active_collaborations": len(self.orchestrator.active_collaborations)
        }