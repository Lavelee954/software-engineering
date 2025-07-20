#!/usr/bin/env python3
"""
Central Router Service

Runs the central router for intelligent agent-to-agent communication.
This service provides:
- Service discovery and agent registry
- Intelligent message routing with load balancing
- Health monitoring and fault tolerance
- Circuit breaker patterns for resilience
"""

import asyncio
import logging
import signal
import sys
from pathlib import Path

# Add project root to path
sys.path.insert(0, str(Path(__file__).parent))

from shared.central_router import CentralRouter


class RouterService:
    """Central Router Service wrapper"""
    
    def __init__(self, nats_url: str = "nats://localhost:4222"):
        self.router = CentralRouter(nats_url)
        self.running = False
        
        # Setup logging
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger(__name__)
        
        # Setup signal handlers
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        self.logger.info(f"Received signal {signum}, shutting down...")
        asyncio.create_task(self.stop())
    
    async def start(self):
        """Start the router service"""
        try:
            self.logger.info("Starting Central Router Service...")
            await self.router.start()
            self.running = True
            self.logger.info("Central Router Service started successfully")
            
        except Exception as e:
            self.logger.error(f"Failed to start router service: {e}")
            raise
    
    async def stop(self):
        """Stop the router service"""
        if not self.running:
            return
            
        self.logger.info("Stopping Central Router Service...")
        self.running = False
        
        try:
            await self.router.stop()
            self.logger.info("Central Router Service stopped successfully")
            
        except Exception as e:
            self.logger.error(f"Error stopping router service: {e}")
    
    async def run(self):
        """Main run loop"""
        await self.start()
        
        try:
            while self.running:
                await asyncio.sleep(1)
                
        except KeyboardInterrupt:
            self.logger.info("Received keyboard interrupt")
        finally:
            await self.stop()


async def main():
    """Main entry point"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Central Router Service")
    parser.add_argument(
        "--nats-url", 
        default="nats://localhost:4222",
        help="NATS server URL"
    )
    parser.add_argument(
        "--log-level",
        default="INFO",
        choices=["DEBUG", "INFO", "WARNING", "ERROR"],
        help="Logging level"
    )
    
    args = parser.parse_args()
    
    # Set logging level
    logging.getLogger().setLevel(getattr(logging, args.log_level))
    
    # Start router service
    service = RouterService(args.nats_url)
    await service.run()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nShutdown complete")
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1) 