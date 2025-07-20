"""
Sentiment Analysis Agent Process Runner

Manages the Sentiment Analysis Agent as an independent process in the monorepo
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

from agents.sentiment_analysis.agent import SentimentAnalysisAgent, SentimentConfig
from shared.process_manager import create_process_manager


class SentimentAnalysisProcess:
    """Process runner for Sentiment Analysis Agent"""
    
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
        print(f"📡 Sentiment Analysis Process received signal {signum}")
        self.running = False
    
    def _info_handler(self, signum: int, frame):
        """Handle info signal to dump status"""
        if self.process_manager:
            status = self.process_manager.get_process_status()
            print(f"📊 Sentiment Analysis Process Status:")
            for key, value in status.items():
                print(f"   {key}: {value}")
    
    def create_config(self) -> SentimentConfig:
        """Create configuration from environment variables"""
        
        # Get agent name
        agent_name = os.getenv('SENTIMENT_AGENT_NAME', 'sentiment-analysis-agent')
        
        # Get NATS configuration
        nats_url = os.getenv('NATS_URL', 'nats://localhost:4222')
        
        # Get logging configuration
        log_level = os.getenv('LOG_LEVEL', 'INFO')
        
        # Get analysis parameters
        sentiment_window_minutes = int(os.getenv('SENTIMENT_WINDOW_MINUTES', '15'))
        trend_analysis_hours = int(os.getenv('SENTIMENT_TREND_HOURS', '24'))
        max_history_items = int(os.getenv('SENTIMENT_MAX_HISTORY', '2000'))
        
        # Get model settings
        use_lexicon_analysis = os.getenv('SENTIMENT_USE_LEXICON', 'true').lower() == 'true'
        use_ml_models = os.getenv('SENTIMENT_USE_ML', 'false').lower() == 'true'
        confidence_threshold = float(os.getenv('SENTIMENT_CONFIDENCE_THRESHOLD', '0.6'))
        
        # Get content filtering
        min_content_length = int(os.getenv('SENTIMENT_MIN_CONTENT_LENGTH', '10'))
        max_content_age_hours = int(os.getenv('SENTIMENT_MAX_CONTENT_AGE_HOURS', '48'))
        
        # Get market sentiment settings
        track_market_emotions = os.getenv('SENTIMENT_TRACK_EMOTIONS', 'true').lower() == 'true'
        weight_source_credibility = os.getenv('SENTIMENT_WEIGHT_CREDIBILITY', 'true').lower() == 'true'
        aggregate_by_symbol = os.getenv('SENTIMENT_AGGREGATE_BY_SYMBOL', 'true').lower() == 'true'
        
        # Get trend analysis settings
        detect_sentiment_shifts = os.getenv('SENTIMENT_DETECT_SHIFTS', 'true').lower() == 'true'
        shift_threshold = float(os.getenv('SENTIMENT_SHIFT_THRESHOLD', '0.3'))
        min_samples_for_trend = int(os.getenv('SENTIMENT_MIN_SAMPLES_TREND', '10'))
        
        return SentimentConfig(
            agent_name=agent_name,
            nats_url=nats_url,
            log_level=log_level,
            sentiment_window_minutes=sentiment_window_minutes,
            trend_analysis_hours=trend_analysis_hours,
            max_history_items=max_history_items,
            use_lexicon_analysis=use_lexicon_analysis,
            use_ml_models=use_ml_models,
            confidence_threshold=confidence_threshold,
            min_content_length=min_content_length,
            max_content_age_hours=max_content_age_hours,
            track_market_emotions=track_market_emotions,
            weight_source_credibility=weight_source_credibility,
            aggregate_by_symbol=aggregate_by_symbol,
            detect_sentiment_shifts=detect_sentiment_shifts,
            shift_threshold=shift_threshold,
            min_samples_for_trend=min_samples_for_trend
        )
    
    async def run(self):
        """Main process loop"""
        try:
            # Setup signal handlers
            self.setup_signal_handlers()
            
            # Create configuration
            config = self.create_config()
            
            print(f"🚀 Starting Sentiment Analysis Agent Process")
            print(f"   Agent Name: {config.agent_name}")
            print(f"   NATS URL: {config.nats_url}")
            print(f"   Log Level: {config.log_level}")
            print(f"   Process ID: {os.getpid()}")
            print(f"   Sentiment Window: {config.sentiment_window_minutes} minutes")
            print(f"   Confidence Threshold: {config.confidence_threshold}")
            print(f"   Use ML Models: {config.use_ml_models}")
            print(f"   Track Emotions: {config.track_market_emotions}")
            print()
            
            # Create agent
            self.agent = SentimentAnalysisAgent(config)
            
            # Create process manager
            self.process_manager = create_process_manager(config.agent_name, self.agent)
            
            # Start agent
            self.running = True
            await self.process_manager.run()
            
        except KeyboardInterrupt:
            print(f"\n⚠️  Sentiment Analysis Process interrupted")
        except Exception as e:
            print(f"❌ Sentiment Analysis Process error: {e}")
            import traceback
            traceback.print_exc()
            return 1
        finally:
            self.running = False
            if self.agent:
                await self.agent.stop()
        
        return 0


async def main():
    """Main entry point for Sentiment Analysis Agent process"""
    process = SentimentAnalysisProcess()
    exit_code = await process.run()
    return exit_code


if __name__ == "__main__":
    try:
        exit_code = asyncio.run(main())
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print(f"\n⚠️  Sentiment Analysis startup interrupted")
        sys.exit(0)
    except Exception as e:
        print(f"❌ Sentiment Analysis startup failed: {e}")
        sys.exit(1)