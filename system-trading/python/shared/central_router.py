"""
Central Router for Agent-to-Agent Communication

Intelligent routing system that manages communication between all agents:
- Service discovery and agent registry
- Load balancing across agent instances
- Message routing with priority handling
- Circuit breaker patterns for fault tolerance
- Real-time agent health monitoring
"""

import asyncio
import json
import uuid
from typing import Dict, List, Optional, Any, Set
from datetime import datetime, timedelta
from enum import Enum
from dataclasses import dataclass, field
import logging
from collections import defaultdict, deque

import nats
from nats.aio.client import Client as NATS


class RoutingStrategy(Enum):
    """Message routing strategies"""
    ROUND_ROBIN = "round_robin"
    LEAST_LOADED = "least_loaded"
    RANDOM = "random"
    STICKY_SESSION = "sticky_session"
    PRIORITY_BASED = "priority_based"


class AgentStatus(Enum):
    """Agent health status"""
    HEALTHY = "healthy"
    DEGRADED = "degraded"
    UNHEALTHY = "unhealthy"
    OFFLINE = "offline"


@dataclass
class AgentInstance:
    """Agent instance metadata"""
    agent_id: str
    agent_type: str
    capabilities: List[str]
    endpoint: str
    status: AgentStatus
    last_heartbeat: datetime
    load_factor: float = 0.0
    message_count: int = 0
    error_count: int = 0
    version: str = "1.0.0"
    metadata: Dict[str, Any] = field(default_factory=dict)


@dataclass
class RoutingRule:
    """Message routing configuration"""
    source_pattern: str
    destination_pattern: str
    strategy: RoutingStrategy
    priority: int = 5
    timeout: int = 30
    retry_count: int = 3
    circuit_breaker_threshold: int = 5
    conditions: Dict[str, Any] = field(default_factory=dict)


class CircuitBreaker:
    """Circuit breaker for agent fault tolerance"""
    
    def __init__(self, threshold: int = 5, timeout: int = 60):
        self.threshold = threshold
        self.timeout = timeout
        self.failure_count = 0
        self.last_failure = None
        self.state = "closed"  # closed, open, half-open
    
    def record_success(self):
        self.failure_count = 0
        self.state = "closed"
    
    def record_failure(self):
        self.failure_count += 1
        self.last_failure = datetime.now()
        
        if self.failure_count >= self.threshold:
            self.state = "open"
    
    def can_execute(self) -> bool:
        if self.state == "closed":
            return True
        elif self.state == "open":
            if self.last_failure and \
               (datetime.now() - self.last_failure).seconds > self.timeout:
                self.state = "half-open"
                return True
            return False
        else:  # half-open
            return True


class CentralRouter:
    """
    Central router for agent-to-agent communication
    
    Provides intelligent routing, load balancing, and fault tolerance
    for the multi-agent trading system.
    """
    
    def __init__(self, nats_url: str = "nats://localhost:4222"):
        self.nats_url = nats_url
        self.nats_client: Optional[NATS] = None
        
        # Agent registry and service discovery
        self.agents: Dict[str, AgentInstance] = {}
        self.agent_types: Dict[str, List[str]] = defaultdict(list)
        self.capabilities_index: Dict[str, List[str]] = defaultdict(list)
        
        # Routing configuration
        self.routing_rules: List[RoutingRule] = []
        self.routing_counters: Dict[str, int] = defaultdict(int)
        
        # Circuit breakers for fault tolerance
        self.circuit_breakers: Dict[str, CircuitBreaker] = {}
        
        # Message handling
        self.message_queue: deque = deque(maxlen=10000)
        self.pending_messages: Dict[str, Dict] = {}
        
        # Statistics and monitoring
        self.stats = {
            "messages_routed": 0,
            "routing_errors": 0,
            "agents_registered": 0,
            "circuit_breaker_trips": 0
        }
        
        self.logger = logging.getLogger(__name__)
        self.running = False
    
    async def start(self):
        """Start the central router"""
        self.logger.info("Starting Central Router...")
        
        # Connect to NATS
        self.nats_client = await nats.connect(self.nats_url)
        
        # Setup core subscriptions
        await self._setup_subscriptions()
        
        # Start background tasks
        asyncio.create_task(self._health_monitor_task())
        asyncio.create_task(self._cleanup_task())
        asyncio.create_task(self._statistics_task())
        
        self.running = True
        self.logger.info("Central Router started successfully")
    
    async def stop(self):
        """Stop the central router"""
        self.logger.info("Stopping Central Router...")
        
        self.running = False
        
        if self.nats_client:
            await self.nats_client.close()
        
        self.logger.info("Central Router stopped")
    
    async def _setup_subscriptions(self):
        """Setup core message subscriptions"""
        # Agent registration and heartbeats
        await self.nats_client.subscribe("router.agent.register", 
                                        cb=self._handle_agent_registration)
        await self.nats_client.subscribe("router.agent.heartbeat", 
                                        cb=self._handle_agent_heartbeat)
        await self.nats_client.subscribe("router.agent.unregister", 
                                        cb=self._handle_agent_unregistration)
        
        # Message routing requests
        await self.nats_client.subscribe("router.message.route", 
                                        cb=self._handle_route_message)
        await self.nats_client.subscribe("router.message.broadcast", 
                                        cb=self._handle_broadcast_message)
        
        # Configuration updates
        await self.nats_client.subscribe("router.config.update", 
                                        cb=self._handle_config_update)
    
    async def register_agent(self, agent_instance: AgentInstance):
        """Register a new agent instance"""
        try:
            self.agents[agent_instance.agent_id] = agent_instance
            self.agent_types[agent_instance.agent_type].append(agent_instance.agent_id)
            
            # Index capabilities
            for capability in agent_instance.capabilities:
                self.capabilities_index[capability].append(agent_instance.agent_id)
            
            # Initialize circuit breaker
            self.circuit_breakers[agent_instance.agent_id] = CircuitBreaker()
            
            self.stats["agents_registered"] += 1
            
            self.logger.info(f"Registered agent: {agent_instance.agent_id}", extra={
                "agent_type": agent_instance.agent_type,
                "capabilities": agent_instance.capabilities,
                "endpoint": agent_instance.endpoint
            })
            
            # Notify other agents about new agent
            await self._broadcast_agent_update("agent_registered", agent_instance)
            
        except Exception as e:
            self.logger.error(f"Failed to register agent {agent_instance.agent_id}: {e}")
    
    async def route_message(self, 
                           message: Dict[str, Any], 
                           destination_type: Optional[str] = None,
                           destination_id: Optional[str] = None,
                           strategy: RoutingStrategy = RoutingStrategy.ROUND_ROBIN) -> bool:
        """Route message to appropriate agent(s)"""
        try:
            message_id = message.get("message_id", str(uuid.uuid4()))
            
            # Determine destination agents
            destination_agents = []
            
            if destination_id:
                # Specific agent
                if destination_id in self.agents:
                    destination_agents = [destination_id]
            elif destination_type:
                # Agent type
                destination_agents = self.agent_types.get(destination_type, [])
            else:
                # Capability-based routing
                required_capability = message.get("required_capability")
                if required_capability:
                    destination_agents = self.capabilities_index.get(required_capability, [])
            
            if not destination_agents:
                self.logger.warning(f"No destination agents found for message {message_id}")
                return False
            
            # Filter healthy agents
            healthy_agents = [
                agent_id for agent_id in destination_agents
                if self.agents[agent_id].status == AgentStatus.HEALTHY
                and self.circuit_breakers[agent_id].can_execute()
            ]
            
            if not healthy_agents:
                self.logger.warning(f"No healthy agents available for message {message_id}")
                return False
            
            # Select agent based on strategy
            selected_agent = self._select_agent(healthy_agents, strategy)
            
            # Route message
            success = await self._deliver_message(selected_agent, message)
            
            if success:
                self.stats["messages_routed"] += 1
                self.circuit_breakers[selected_agent].record_success()
            else:
                self.stats["routing_errors"] += 1
                self.circuit_breakers[selected_agent].record_failure()
                
                # Try alternative agent
                if len(healthy_agents) > 1:
                    alternative_agents = [a for a in healthy_agents if a != selected_agent]
                    alternative_agent = self._select_agent(alternative_agents, strategy)
                    success = await self._deliver_message(alternative_agent, message)
            
            return success
            
        except Exception as e:
            self.logger.error(f"Message routing failed: {e}")
            self.stats["routing_errors"] += 1
            return False
    
    def _select_agent(self, agents: List[str], strategy: RoutingStrategy) -> str:
        """Select agent based on routing strategy"""
        if strategy == RoutingStrategy.ROUND_ROBIN:
            key = f"rr_{hash(tuple(agents))}"
            self.routing_counters[key] = (self.routing_counters[key] + 1) % len(agents)
            return agents[self.routing_counters[key]]
        
        elif strategy == RoutingStrategy.LEAST_LOADED:
            return min(agents, key=lambda a: self.agents[a].load_factor)
        
        elif strategy == RoutingStrategy.RANDOM:
            import random
            return random.choice(agents)
        
        else:  # Default to round robin
            return self._select_agent(agents, RoutingStrategy.ROUND_ROBIN)
    
    async def _deliver_message(self, agent_id: str, message: Dict[str, Any]) -> bool:
        """Deliver message to specific agent"""
        try:
            agent = self.agents[agent_id]
            
            # Add routing metadata
            message["routed_by"] = "central_router"
            message["routed_at"] = datetime.now().isoformat()
            message["destination_agent"] = agent_id
            
            # Send via NATS to agent's specific topic
            agent_topic = f"agent.{agent_id}.messages"
            await self.nats_client.publish(agent_topic, json.dumps(message).encode())
            
            # Update agent load
            agent.message_count += 1
            agent.load_factor = agent.message_count / 100.0  # Simple load calculation
            
            return True
            
        except Exception as e:
            self.logger.error(f"Message delivery failed to {agent_id}: {e}")
            return False
    
    async def _handle_agent_registration(self, msg):
        """Handle agent registration messages"""
        try:
            data = json.loads(msg.data.decode())
            agent_instance = AgentInstance(**data)
            await self.register_agent(agent_instance)
            
        except Exception as e:
            self.logger.error(f"Agent registration handling failed: {e}")
    
    async def _handle_agent_heartbeat(self, msg):
        """Handle agent heartbeat messages"""
        try:
            data = json.loads(msg.data.decode())
            agent_id = data["agent_id"]
            
            if agent_id in self.agents:
                agent = self.agents[agent_id]
                agent.last_heartbeat = datetime.now()
                agent.status = AgentStatus.HEALTHY
                agent.load_factor = data.get("load_factor", 0.0)
                
        except Exception as e:
            self.logger.error(f"Heartbeat handling failed: {e}")
    
    async def _handle_route_message(self, msg):
        """Handle message routing requests"""
        try:
            data = json.loads(msg.data.decode())
            message = data["message"]
            destination_type = data.get("destination_type")
            destination_id = data.get("destination_id")
            strategy = RoutingStrategy(data.get("strategy", "round_robin"))
            
            await self.route_message(message, destination_type, destination_id, strategy)
            
        except Exception as e:
            self.logger.error(f"Route message handling failed: {e}")
    
    async def _handle_broadcast_message(self, msg):
        """Handle broadcast message requests"""
        try:
            data = json.loads(msg.data.decode())
            message = data["message"]
            agent_types = data.get("agent_types", [])
            
            # Broadcast to specified agent types or all agents
            target_agents = []
            if agent_types:
                for agent_type in agent_types:
                    target_agents.extend(self.agent_types.get(agent_type, []))
            else:
                target_agents = list(self.agents.keys())
            
            # Deliver to all target agents
            for agent_id in target_agents:
                await self._deliver_message(agent_id, message)
                
        except Exception as e:
            self.logger.error(f"Broadcast handling failed: {e}")
    
    async def _health_monitor_task(self):
        """Monitor agent health and update status"""
        while self.running:
            try:
                current_time = datetime.now()
                
                for agent_id, agent in self.agents.items():
                    # Check heartbeat timeout
                    if agent.last_heartbeat:
                        time_since_heartbeat = current_time - agent.last_heartbeat
                        
                        if time_since_heartbeat > timedelta(seconds=60):
                            agent.status = AgentStatus.OFFLINE
                        elif time_since_heartbeat > timedelta(seconds=30):
                            agent.status = AgentStatus.DEGRADED
                
                await asyncio.sleep(10)  # Check every 10 seconds
                
            except Exception as e:
                self.logger.error(f"Health monitor error: {e}")
    
    async def _cleanup_task(self):
        """Cleanup expired messages and offline agents"""
        while self.running:
            try:
                current_time = datetime.now()
                
                # Remove offline agents after 5 minutes
                offline_agents = [
                    agent_id for agent_id, agent in self.agents.items()
                    if agent.status == AgentStatus.OFFLINE and
                    agent.last_heartbeat and
                    (current_time - agent.last_heartbeat) > timedelta(minutes=5)
                ]
                
                for agent_id in offline_agents:
                    await self._unregister_agent(agent_id)
                
                await asyncio.sleep(60)  # Cleanup every minute
                
            except Exception as e:
                self.logger.error(f"Cleanup task error: {e}")
    
    async def _statistics_task(self):
        """Publish routing statistics"""
        while self.running:
            try:
                stats_message = {
                    "timestamp": datetime.now().isoformat(),
                    "router_stats": self.stats,
                    "agent_count": len(self.agents),
                    "healthy_agents": len([
                        a for a in self.agents.values() 
                        if a.status == AgentStatus.HEALTHY
                    ])
                }
                
                await self.nats_client.publish("router.stats", 
                                              json.dumps(stats_message).encode())
                
                await asyncio.sleep(30)  # Publish every 30 seconds
                
            except Exception as e:
                self.logger.error(f"Statistics task error: {e}")
    
    async def _unregister_agent(self, agent_id: str):
        """Unregister an agent"""
        if agent_id in self.agents:
            agent = self.agents[agent_id]
            
            # Remove from indices
            self.agent_types[agent.agent_type].remove(agent_id)
            for capability in agent.capabilities:
                if agent_id in self.capabilities_index[capability]:
                    self.capabilities_index[capability].remove(agent_id)
            
            # Remove circuit breaker
            if agent_id in self.circuit_breakers:
                del self.circuit_breakers[agent_id]
            
            # Remove agent
            del self.agents[agent_id]
            
            self.logger.info(f"Unregistered agent: {agent_id}")
    
    async def _broadcast_agent_update(self, event_type: str, agent: AgentInstance):
        """Broadcast agent status updates to other agents"""
        update_message = {
            "event_type": event_type,
            "agent_id": agent.agent_id,
            "agent_type": agent.agent_type,
            "capabilities": agent.capabilities,
            "status": agent.status.value,
            "timestamp": datetime.now().isoformat()
        }
        
        await self.nats_client.publish("router.agent.updates", 
                                      json.dumps(update_message).encode())
    
    def get_agent_info(self, agent_id: Optional[str] = None) -> Dict[str, Any]:
        """Get agent information"""
        if agent_id:
            return self.agents.get(agent_id, {})
        else:
            return {
                "total_agents": len(self.agents),
                "agent_types": dict(self.agent_types),
                "capabilities": dict(self.capabilities_index),
                "stats": self.stats
            } 