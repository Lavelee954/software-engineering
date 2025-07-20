"""
Process Manager for Multi-Agent System

Provides process lifecycle management for individual agents in the monorepo
multi-agent architecture.
"""

import asyncio
import os
import signal
import sys
from typing import Optional, Dict, Any
import psutil


class ProcessManager:
    """
    Manages the lifecycle of an individual agent process
    
    Provides standardized process management including:
    - Signal handling
    - Health monitoring  
    - Resource tracking
    - Graceful shutdown
    """
    
    def __init__(self, agent_name: str, agent_instance: Any):
        self.agent_name = agent_name
        self.agent = agent_instance
        self.process_id = os.getpid()
        self.running = False
        self.start_time = None
        
        # Process monitoring
        self.process = psutil.Process(self.process_id)
        
        # Setup signal handlers
        self.setup_signal_handlers()
    
    def setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown"""
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGUSR1, self._info_handler)
    
    def _signal_handler(self, signum: int, frame):
        """Handle shutdown signals"""
        print(f"📡 Process {self.process_id} received signal {signum}")
        
        if signum == signal.SIGTERM:
            print(f"🛑 Graceful shutdown requested for {self.agent_name}")
        elif signum == signal.SIGINT:
            print(f"⚠️  Interrupt received for {self.agent_name}")
        
        self.running = False
        
        # Trigger agent shutdown
        if self.agent and hasattr(self.agent, 'stop'):
            asyncio.create_task(self.agent.stop())
    
    def _info_handler(self, signum: int, frame):
        """Handle info signal (SIGUSR1) to dump process status"""
        status = self.get_process_status()
        print(f"📊 Process Status for {self.agent_name}:")
        for key, value in status.items():
            print(f"   {key}: {value}")
    
    def get_process_status(self) -> Dict[str, Any]:
        """Get current process status and resource usage"""
        try:
            memory_info = self.process.memory_info()
            cpu_percent = self.process.cpu_percent()
            
            status = {
                "agent_name": self.agent_name,
                "process_id": self.process_id,
                "running": self.running,
                "cpu_percent": f"{cpu_percent:.2f}%",
                "memory_rss": f"{memory_info.rss / 1024 / 1024:.2f} MB",
                "memory_vms": f"{memory_info.vms / 1024 / 1024:.2f} MB",
                "num_threads": self.process.num_threads(),
                "status": self.process.status(),
            }
            
            if self.start_time:
                uptime = asyncio.get_event_loop().time() - self.start_time
                status["uptime"] = f"{uptime:.2f} seconds"
            
            return status
            
        except psutil.NoSuchProcess:
            return {"error": "Process no longer exists"}
        except Exception as e:
            return {"error": f"Failed to get process status: {e}"}
    
    async def start_agent(self):
        """Start the managed agent"""
        try:
            print(f"🚀 Starting agent: {self.agent_name} (PID: {self.process_id})")
            
            # Record start time
            self.start_time = asyncio.get_event_loop().time()
            
            # Start the agent
            if hasattr(self.agent, 'start'):
                await self.agent.start()
            
            self.running = True
            print(f"✅ Agent {self.agent_name} started successfully")
            
        except Exception as e:
            print(f"❌ Failed to start agent {self.agent_name}: {e}")
            raise
    
    async def stop_agent(self):
        """Stop the managed agent"""
        try:
            if not self.running:
                return
            
            print(f"🛑 Stopping agent: {self.agent_name}")
            
            # Stop the agent
            if hasattr(self.agent, 'stop'):
                await self.agent.stop()
            
            self.running = False
            print(f"✅ Agent {self.agent_name} stopped successfully")
            
        except Exception as e:
            print(f"❌ Error stopping agent {self.agent_name}: {e}")
    
    async def run(self):
        """
        Main process loop
        
        Starts the agent and keeps the process running until shutdown.
        """
        try:
            # Start the agent
            await self.start_agent()
            
            # Print initial status
            status = self.get_process_status()
            print(f"📊 Initial Status:")
            for key, value in status.items():
                print(f"   {key}: {value}")
            
            # Main loop
            while self.running:
                await asyncio.sleep(1)
                
        except KeyboardInterrupt:
            print(f"\n⚠️  Keyboard interrupt received")
        except Exception as e:
            print(f"❌ Process error: {e}")
            import traceback
            traceback.print_exc()
            return 1
        finally:
            # Cleanup
            await self.stop_agent()
            
            # Final status
            final_status = self.get_process_status()
            print(f"📊 Final Status:")
            for key, value in final_status.items():
                print(f"   {key}: {value}")
        
        return 0


class MultiProcessManager:
    """
    Manages multiple agent processes
    
    Coordinates startup, shutdown, and monitoring of multiple agents
    running as separate processes.
    """
    
    def __init__(self):
        self.processes: Dict[str, ProcessManager] = {}
        self.running = False
    
    def add_agent(self, agent_name: str, agent_instance: Any) -> ProcessManager:
        """Add an agent to be managed"""
        manager = ProcessManager(agent_name, agent_instance)
        self.processes[agent_name] = manager
        return manager
    
    async def start_all(self):
        """Start all managed agents"""
        print(f"🚀 Starting {len(self.processes)} agents...")
        
        for name, manager in self.processes.items():
            try:
                await manager.start_agent()
            except Exception as e:
                print(f"❌ Failed to start {name}: {e}")
                # Continue with other agents
        
        self.running = True
        print(f"✅ All agents started")
    
    async def stop_all(self):
        """Stop all managed agents"""
        print(f"🛑 Stopping {len(self.processes)} agents...")
        
        # Stop all agents concurrently
        stop_tasks = []
        for manager in self.processes.values():
            stop_tasks.append(manager.stop_agent())
        
        await asyncio.gather(*stop_tasks, return_exceptions=True)
        
        self.running = False
        print(f"✅ All agents stopped")
    
    def get_all_status(self) -> Dict[str, Dict[str, Any]]:
        """Get status of all managed processes"""
        return {
            name: manager.get_process_status() 
            for name, manager in self.processes.items()
        }
    
    async def health_check(self) -> Dict[str, bool]:
        """Perform health check on all agents"""
        health_status = {}
        
        for name, manager in self.processes.items():
            try:
                # Check if process is running
                if manager.running and manager.process.is_running():
                    # Additional health checks can be added here
                    health_status[name] = True
                else:
                    health_status[name] = False
            except Exception:
                health_status[name] = False
        
        return health_status


def create_process_manager(agent_name: str, agent_instance: Any) -> ProcessManager:
    """
    Factory function to create a process manager for an agent
    
    Args:
        agent_name: Name of the agent
        agent_instance: Instance of the agent to manage
    
    Returns:
        ProcessManager instance
    """
    return ProcessManager(agent_name, agent_instance)