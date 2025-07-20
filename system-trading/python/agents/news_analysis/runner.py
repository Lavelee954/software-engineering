"""
News Analysis Agent Process Runner

Manages the News Analysis Agent as an independent process in the monorepo
multi-agent architecture.
"""

import asyncio
import os
import signal
import sys
from pathlib import Path

# Add project root to Python path
PROJECT_ROOT = Path(__file__).parents[2]
sys.path.insert(0, str(PROJECT_ROOT))

from agents.news_analysis.agent import NewsAnalysisAgent, NewsConfig
from shared.process_manager import create_process_manager


class NewsAnalysisProcess:
    """Process runner for News Analysis Agent"""
    
    def __init__(self):
        self.agent = None
        self.process_manager = None
        self.running = False
    
    def setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown"""
        signal.signal(signal.SIGTERM, self._signal_handler)
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGUSR1, self._info_handler)
    
    def _signal_handler(self, signum: int, frame):
        """Handle shutdown signals"""
        print(f"📡 News Analysis Process received signal {signum}")
        self.running = False
    
    def _info_handler(self, signum: int, frame):
        """Handle info signal to dump status"""
        if self.process_manager:
            status = self.process_manager.get_process_status()
            print(f"📊 News Analysis Process Status:")
            for key, value in status.items():
                print(f"   {key}: {value}")
    
    def create_config(self) -> NewsConfig:
        """Create configuration from environment variables"""
        
        # Get agent name
        agent_name = os.getenv('NEWS_AGENT_NAME', 'news-analysis-agent')
        
        # Get NATS configuration
        nats_url = os.getenv('NATS_URL', 'nats://localhost:4222')
        
        # Get logging configuration
        log_level = os.getenv('LOG_LEVEL', 'INFO')
        
        # Get analysis parameters
        max_history = int(os.getenv('NEWS_MAX_HISTORY', '1000'))
        relevance_threshold = float(os.getenv('NEWS_RELEVANCE_THRESHOLD', '0.3'))
        impact_threshold = float(os.getenv('NEWS_IMPACT_THRESHOLD', '0.5'))
        
        # Get content filtering
        min_article_length = int(os.getenv('NEWS_MIN_ARTICLE_LENGTH', '100'))
        max_article_age_hours = int(os.getenv('NEWS_MAX_ARTICLE_AGE_HOURS', '24'))
        
        # Get tracked symbols and sectors from environment (comma-separated)
        tracked_symbols_str = os.getenv('NEWS_TRACKED_SYMBOLS', '')
        tracked_symbols = [s.strip().upper() for s in tracked_symbols_str.split(',') if s.strip()] if tracked_symbols_str else None
        
        tracked_sectors_str = os.getenv('NEWS_TRACKED_SECTORS', '')
        tracked_sectors = [s.strip().lower() for s in tracked_sectors_str.split(',') if s.strip()] if tracked_sectors_str else None
        
        # NLP settings
        use_advanced_nlp = os.getenv('NEWS_USE_ADVANCED_NLP', 'true').lower() == 'true'
        extract_entities = os.getenv('NEWS_EXTRACT_ENTITIES', 'true').lower() == 'true'
        
        return NewsConfig(
            agent_name=agent_name,
            nats_url=nats_url,
            log_level=log_level,
            max_history_items=max_history,
            relevance_threshold=relevance_threshold,
            impact_threshold=impact_threshold,
            min_article_length=min_article_length,
            max_article_age_hours=max_article_age_hours,
            use_advanced_nlp=use_advanced_nlp,
            extract_entities=extract_entities,
            tracked_symbols=tracked_symbols,
            tracked_sectors=tracked_sectors
        )
    
    async def run(self):
        """Main process loop"""
        try:
            # Setup signal handlers
            self.setup_signal_handlers()
            
            # Create configuration
            config = self.create_config()
            
            print(f"🚀 Starting News Analysis Agent Process")
            print(f"   Agent Name: {config.agent_name}")
            print(f"   NATS URL: {config.nats_url}")
            print(f"   Log Level: {config.log_level}")
            print(f"   Process ID: {os.getpid()}")
            print(f"   Relevance Threshold: {config.relevance_threshold}")
            print(f"   Tracked Symbols: {len(config.tracked_symbols) if config.tracked_symbols else 0}")
            print()
            
            # Create agent
            self.agent = NewsAnalysisAgent(config)
            
            # Create process manager
            self.process_manager = create_process_manager(config.agent_name, self.agent)
            
            # Start agent
            self.running = True
            await self.process_manager.run()
            
        except KeyboardInterrupt:
            print(f"\n⚠️  News Analysis Process interrupted")
        except Exception as e:
            print(f"❌ News Analysis Process error: {e}")
            import traceback
            traceback.print_exc()
            return 1
        finally:
            self.running = False
            if self.agent:
                await self.agent.stop()
        
        return 0


async def main():
    """Main entry point for News Analysis Agent process"""
    process = NewsAnalysisProcess()
    exit_code = await process.run()
    return exit_code


if __name__ == "__main__":
    try:
        exit_code = asyncio.run(main())
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print(f"\n⚠️  News Analysis startup interrupted")
        sys.exit(0)
    except Exception as e:
        print(f"❌ News Analysis startup failed: {e}")
        sys.exit(1)