"""
Base Agent Class

Provides common functionality for all Python-based analysis agents following
the CLAUDE.md architecture specifications.
"""

import asyncio
import json
import signal
import sys
import uuid
from abc import ABC, abstractmethod
from typing import Any, Callable, Dict, Optional, List
from datetime import datetime

import nats
import structlog
from nats.aio.client import Client as NATS
from pydantic import BaseModel, Field


class AgentConfig(BaseModel):
    """Base configuration for all agents"""
    
    agent_name: str = Field(..., description="Unique name for this agent instance")
    nats_url: str = Field(default="nats://localhost:4222", description="NATS server URL")
    log_level: str = Field(default="INFO", description="Logging level")
    shutdown_timeout: int = Field(default=30, description="Graceful shutdown timeout in seconds")
    capabilities: Optional[List[str]] = Field(default=None, description="Agent capabilities for router discovery")


class BaseAgent(ABC):
    """
    Base class for all analysis agents in the trading system.
    
    Implements the common NATS message bus integration, logging, and lifecycle
    management. Each concrete agent implements specific analysis logic.
    
    Now integrated with Central Router for intelligent A2A communication.
    """
    
    def __init__(self, config: AgentConfig) -> None:
        self.config = config
        self.logger = self._setup_logging()
        self.nats_client: Optional[NATS] = None
        self.subscriptions: Dict[str, Any] = {}
        self.running = False
        
        # Central Router integration
        self.agent_id = f"{config.agent_name}_{uuid.uuid4().hex[:8]}"
        self.agent_type = config.agent_name
        self.capabilities = getattr(config, 'capabilities', [self.agent_type])
        self.load_factor = 0.0
        self.message_count = 0
        
        # Setup signal handlers for graceful shutdown
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
        
    def _setup_logging(self) -> structlog.BoundLogger:
        """Setup structured logging for the agent"""
        structlog.configure(
            processors=[
                structlog.processors.TimeStamper(fmt="iso"),
                structlog.processors.add_log_level,
                structlog.processors.JSONRenderer()
            ],
            wrapper_class=structlog.make_filtering_bound_logger(
                getattr(structlog, self.config.log_level.upper())
            ),
            logger_factory=structlog.PrintLoggerFactory(),
            cache_logger_on_first_use=True,
        )
        
        return structlog.get_logger().bind(
            agent=self.config.agent_name,
            component="analysis_agent"
        )
    
    async def start(self) -> None:
        """Start the agent and connect to NATS"""
        try:
            self.logger.info("Starting agent", agent=self.config.agent_name)
            
            # Connect to NATS
            self.nats_client = await nats.connect(self.config.nats_url)
            self.logger.info("Connected to NATS", url=self.config.nats_url)
            
            # Register with Central Router
            await self._register_with_router()
            
            # Setup subscriptions (including agent-specific subscription)
            await self._setup_subscriptions()
            await self._setup_router_subscriptions()
            
            # Agent-specific initialization
            await self._initialize()
            
            # Start heartbeat task
            asyncio.create_task(self._heartbeat_task())
            
            self.running = True
            self.logger.info("Agent started successfully", agent_id=self.agent_id)
            
        except Exception as e:
            self.logger.error("Failed to start agent", error=str(e))
            raise
    
    async def stop(self) -> None:
        """Stop the agent gracefully"""
        if not self.running:
            return
            
        self.logger.info("Stopping agent", agent=self.config.agent_name)
        self.running = False
        
        try:
            # Unregister from Central Router
            await self._unregister_from_router()
            
            # Agent-specific cleanup
            await self._cleanup()
            
            # Close NATS subscriptions
            for subscription in self.subscriptions.values():
                await subscription.unsubscribe()
            self.subscriptions.clear()
            
            # Close NATS connection
            if self.nats_client:
                await self.nats_client.close()
                
            self.logger.info("Agent stopped successfully")
            
        except Exception as e:
            self.logger.error("Error during agent shutdown", error=str(e))
    
    async def run(self) -> None:
        """Main run loop for the agent"""
        await self.start()
        
        try:
            while self.running:
                await asyncio.sleep(1)
                
        except KeyboardInterrupt:
            self.logger.info("Received keyboard interrupt")
        finally:
            await self.stop()
    
    def _signal_handler(self, signum: int, frame: Any) -> None:
        """Handle shutdown signals"""
        self.logger.info("Received shutdown signal", signal=signum)
        asyncio.create_task(self.stop())
    
    async def publish(self, topic: str, message: Dict[str, Any]) -> None:
        """Publish a message to the NATS message bus"""
        if not self.nats_client:
            raise RuntimeError("NATS client not connected")
            
        try:
            data = json.dumps(message).encode()
            await self.nats_client.publish(topic, data)
            
            self.logger.debug(
                "Published message", 
                topic=topic, 
                message_size=len(data)
            )
            
        except Exception as e:
            self.logger.error(
                "Failed to publish message", 
                topic=topic, 
                error=str(e)
            )
            raise
    
    async def _message_handler(self, topic: str, handler: Callable) -> Callable:
        """Create a message handler wrapper with error handling and logging"""
        async def wrapper(msg) -> None:
            try:
                # Decode message
                data = json.loads(msg.data.decode())
                
                self.logger.debug(
                    "Received message",
                    topic=topic,
                    message_size=len(msg.data)
                )
                
                # Process message
                await handler(data)
                
            except json.JSONDecodeError as e:
                self.logger.error(
                    "Failed to decode message",
                    topic=topic,
                    error=str(e)
                )
            except Exception as e:
                self.logger.error(
                    "Error processing message",
                    topic=topic,
                    error=str(e)
                )
        
        return wrapper
    
    @abstractmethod
    async def _setup_subscriptions(self) -> None:
        """Setup NATS subscriptions for the agent"""
        pass
    
    # Central Router Integration Methods
    
    async def _register_with_router(self) -> None:
        """Register this agent with the Central Router"""
        try:
            registration_message = {
                "agent_id": self.agent_id,
                "agent_type": self.agent_type,
                "capabilities": self.capabilities,
                "endpoint": f"agent.{self.agent_id}.messages",
                "status": "healthy",
                "last_heartbeat": datetime.now().isoformat(),
                "load_factor": self.load_factor,
                "message_count": self.message_count,
                "version": "1.0.0",
                "metadata": {
                    "config": self.config.agent_name,
                    "nats_url": self.config.nats_url
                }
            }
            
            await self.nats_client.publish(
                "router.agent.register", 
                json.dumps(registration_message).encode()
            )
            
            self.logger.info(
                "Registered with Central Router", 
                agent_id=self.agent_id,
                capabilities=self.capabilities
            )
            
        except Exception as e:
            self.logger.error("Failed to register with router", error=str(e))
    
    async def _unregister_from_router(self) -> None:
        """Unregister this agent from the Central Router"""
        try:
            unregistration_message = {
                "agent_id": self.agent_id,
                "timestamp": datetime.now().isoformat()
            }
            
            await self.nats_client.publish(
                "router.agent.unregister", 
                json.dumps(unregistration_message).encode()
            )
            
            self.logger.info("Unregistered from Central Router", agent_id=self.agent_id)
            
        except Exception as e:
            self.logger.error("Failed to unregister from router", error=str(e))
    
    async def _setup_router_subscriptions(self) -> None:
        """Setup subscriptions for Central Router integration"""
        try:
            # Subscribe to agent-specific messages from router
            agent_topic = f"agent.{self.agent_id}.messages"
            subscription = await self.nats_client.subscribe(
                agent_topic,
                cb=await self._message_handler(agent_topic, self._handle_routed_message)
            )
            self.subscriptions[agent_topic] = subscription
            
            # Subscribe to router updates
            router_updates_subscription = await self.nats_client.subscribe(
                "router.agent.updates",
                cb=await self._message_handler("router.agent.updates", self._handle_router_update)
            )
            self.subscriptions["router.agent.updates"] = router_updates_subscription
            
            self.logger.info("Setup router subscriptions", agent_id=self.agent_id)
            
        except Exception as e:
            self.logger.error("Failed to setup router subscriptions", error=str(e))
    
    async def _heartbeat_task(self) -> None:
        """Send periodic heartbeats to the Central Router"""
        while self.running:
            try:
                await asyncio.sleep(15)  # Send heartbeat every 15 seconds
                
                if not self.running:
                    break
                
                heartbeat_message = {
                    "agent_id": self.agent_id,
                    "timestamp": datetime.now().isoformat(),
                    "load_factor": self.load_factor,
                    "message_count": self.message_count,
                    "status": "healthy"
                }
                
                await self.nats_client.publish(
                    "router.agent.heartbeat", 
                    json.dumps(heartbeat_message).encode()
                )
                
                self.logger.debug("Sent heartbeat", agent_id=self.agent_id)
                
            except Exception as e:
                self.logger.error("Heartbeat failed", error=str(e))
                await asyncio.sleep(5)  # Retry after 5 seconds on error
    
    async def _handle_routed_message(self, message: Dict[str, Any]) -> None:
        """Handle messages routed through the Central Router"""
        try:
            # Update load metrics
            self.message_count += 1
            self.load_factor = self.message_count / 100.0
            
            # Check if this is an A2A communication message
            if message.get("routed_by") == "central_router":
                self.logger.debug(
                    "Received routed message", 
                    message_id=message.get("message_id"),
                    sender=message.get("sender_id")
                )
                
                # Handle A2A message
                await self._handle_a2a_message(message)
            else:
                # Handle regular message
                await self._handle_regular_message(message)
            
        except Exception as e:
            self.logger.error("Failed to handle routed message", error=str(e))
    
    async def _handle_a2a_message(self, message: Dict[str, Any]) -> None:
        """Handle Agent-to-Agent communication messages"""
        # Default implementation - can be overridden by concrete agents
        self.logger.info(
            "Received A2A message", 
            message_type=message.get("message_type"),
            sender=message.get("sender_id")
        )
    
    async def _handle_regular_message(self, message: Dict[str, Any]) -> None:
        """Handle regular business logic messages"""
        # Default implementation - can be overridden by concrete agents
        self.logger.debug("Received regular message", message_id=message.get("message_id"))
    
    async def _handle_router_update(self, message: Dict[str, Any]) -> None:
        """Handle updates from the Central Router"""
        try:
            event_type = message.get("event_type")
            agent_id = message.get("agent_id")
            
            if event_type == "agent_registered":
                self.logger.info("New agent registered", new_agent_id=agent_id)
            elif event_type == "agent_unregistered":
                self.logger.info("Agent unregistered", removed_agent_id=agent_id)
            
        except Exception as e:
            self.logger.error("Failed to handle router update", error=str(e))
    
    async def route_message_to_agent(self, 
                                   message: Dict[str, Any], 
                                   destination_type: Optional[str] = None,
                                   destination_id: Optional[str] = None,
                                   strategy: str = "round_robin") -> bool:
        """Route message to another agent through the Central Router"""
        try:
            # Add sender information
            message["sender_id"] = self.agent_id
            message["message_id"] = message.get("message_id", str(uuid.uuid4()))
            message["timestamp"] = datetime.now().isoformat()
            
            routing_request = {
                "message": message,
                "destination_type": destination_type,
                "destination_id": destination_id,
                "strategy": strategy
            }
            
            await self.nats_client.publish(
                "router.message.route",
                json.dumps(routing_request).encode()
            )
            
            self.logger.debug(
                "Routed message", 
                message_id=message["message_id"],
                destination_type=destination_type,
                destination_id=destination_id
            )
            
            return True
            
        except Exception as e:
            self.logger.error("Failed to route message", error=str(e))
            return False
    
    async def broadcast_message(self, 
                              message: Dict[str, Any], 
                              agent_types: Optional[List[str]] = None) -> bool:
        """Broadcast message to multiple agents through the Central Router"""
        try:
            # Add sender information
            message["sender_id"] = self.agent_id
            message["message_id"] = message.get("message_id", str(uuid.uuid4()))
            message["timestamp"] = datetime.now().isoformat()
            
            broadcast_request = {
                "message": message,
                "agent_types": agent_types
            }
            
            await self.nats_client.publish(
                "router.message.broadcast",
                json.dumps(broadcast_request).encode()
            )
            
            self.logger.info(
                "Broadcasted message", 
                message_id=message["message_id"],
                agent_types=agent_types
            )
            
            return True
            
        except Exception as e:
            self.logger.error("Failed to broadcast message", error=str(e))
            return False
    
    @abstractmethod
    async def _initialize(self) -> None:
        """Agent-specific initialization logic"""
        pass
    
    @abstractmethod
    async def _cleanup(self) -> None:
        """Agent-specific cleanup logic"""
        pass