#!/usr/bin/env python3
"""
Technical Analysis Agent Process Runner

Independent process runner for the Technical Analysis Agent following
the monorepo multi-agent architecture pattern.
"""

import asyncio
import os
import sys
import signal
from pathlib import Path

# Add project root to Python path for shared imports
PROJECT_ROOT = Path(__file__).parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

from shared.process_manager import ProcessManager
from agents.technical_analysis.agent import TechnicalAnalysisAgent
from agents.technical_analysis.config import load_config


class TechnicalAnalysisProcess:
    """Process wrapper for Technical Analysis Agent"""
    
    def __init__(self):
        self.agent = None
        self.running = False
        self.setup_signal_handlers()
    
    def setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown"""
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        print(f"📡 Received signal {signum}, initiating graceful shutdown...")
        self.running = False
        if self.agent:
            asyncio.create_task(self.agent.stop())
    
    async def run(self):
        """Run the Technical Analysis Agent process"""
        try:
            # Load configuration
            config = load_config()
            
            print(f"🚀 Starting Technical Analysis Agent Process")
            print(f"   Agent Name: {config.agent_name}")
            print(f"   NATS URL: {config.nats_url}")
            print(f"   Log Level: {config.log_level}")
            print(f"   Process ID: {os.getpid()}")
            print(f"   Data Window: {config.data_window_size}")
            print(f"   Min Bars: {config.min_bars_required}")
            
            # Create and start agent
            self.agent = TechnicalAnalysisAgent(config)
            await self.agent.start()
            self.running = True
            
            print(f"✅ Technical Analysis Agent running successfully")
            
            # Main process loop
            while self.running:
                await asyncio.sleep(1)
                
        except KeyboardInterrupt:
            print(f"\n⚠️  Received keyboard interrupt")
        except Exception as e:
            print(f"❌ Agent process failed: {e}")
            import traceback
            traceback.print_exc()
            return 1
        finally:
            if self.agent:
                print(f"🛑 Stopping Technical Analysis Agent...")
                await self.agent.stop()
                print(f"✅ Agent stopped successfully")
        
        return 0


async def main():
    """Main entry point"""
    process = TechnicalAnalysisProcess()
    exit_code = await process.run()
    sys.exit(exit_code)


if __name__ == "__main__":
    print(f"🚀 Technical Analysis Agent - Monorepo Multi-Agent System")
    asyncio.run(main())