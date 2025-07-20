#!/usr/bin/env python3
"""
Universal Agent Starter

Starts any agent as an independent process within the monorepo multi-agent
architecture. Provides a unified interface for launching different agent types.
"""

import argparse
import asyncio
import importlib
import os
import sys
from pathlib import Path
from typing import Dict, Optional

# Add project root to Python path
PROJECT_ROOT = Path(__file__).parents[1]
sys.path.insert(0, str(PROJECT_ROOT))


# Registry of available agents and their runner modules
AVAILABLE_AGENTS = {
    'technical': {
        'module': 'agents.technical_analysis.runner',
        'description': 'Technical Analysis Agent - processes market data for technical indicators',
        'status': 'implemented'
    },
    'news': {
        'module': 'agents.news_analysis.runner',
        'description': 'News Analysis Agent - analyzes news articles for market impact',
        'status': 'implemented'
    },
    'sentiment': {
        'module': 'agents.sentiment_analysis.runner', 
        'description': 'Sentiment Analysis Agent - performs NLP sentiment analysis',
        'status': 'implemented'
    },
    'macro': {
        'module': 'agents.macro_economic.runner',
        'description': 'Macro Economic Agent - analyzes macroeconomic indicators',
        'status': 'planned'
    },
    'strategy': {
        'module': 'agents.strategy.runner',
        'description': 'Strategy Agent - makes trading decisions based on all insights',
        'status': 'planned'
    },
    'backtest': {
        'module': 'agents.backtest.runner',
        'description': 'Backtest Agent - runs historical simulations',
        'status': 'planned'
    }
}


class AgentStarter:
    """Universal agent starter for the monorepo architecture"""
    
    def __init__(self):
        self.agent_type = None
        self.agent_name = None
        self.config_overrides = {}
    
    def list_agents(self):
        """List all available agents"""
        print("📋 Available Agents:")
        print("=" * 50)
        for agent_type, info in AVAILABLE_AGENTS.items():
            status_icon = "✅" if info.get('status') == 'implemented' else "🚧"
            print(f"  {status_icon} {agent_type:12} - {info['description']}")
        print()
    
    def validate_agent_type(self, agent_type: str) -> bool:
        """Validate that the agent type is available"""
        if agent_type not in AVAILABLE_AGENTS:
            print(f"❌ Unknown agent type: {agent_type}")
            print(f"Available agents: {', '.join(AVAILABLE_AGENTS.keys())}")
            return False
        return True
    
    def set_environment_variables(self, agent_type: str, **kwargs):
        """Set environment variables for agent configuration"""
        
        # Set agent name if provided
        if kwargs.get('name'):
            env_var = f"{agent_type.upper()}_AGENT_NAME"
            os.environ[env_var] = kwargs['name']
            self.agent_name = kwargs['name']
        
        # Set log level if provided
        if kwargs.get('log_level'):
            os.environ['LOG_LEVEL'] = kwargs['log_level']
        
        # Set NATS URL if provided
        if kwargs.get('nats_url'):
            os.environ['NATS_URL'] = kwargs['nats_url']
        
        # Set any additional configuration overrides
        for key, value in self.config_overrides.items():
            env_var = f"{agent_type.upper()}_{key.upper()}"
            os.environ[env_var] = str(value)
    
    async def start_agent(self, agent_type: str, **kwargs) -> int:
        """Start the specified agent"""
        
        if not self.validate_agent_type(agent_type):
            return 1
        
        self.agent_type = agent_type
        
        try:
            # Set environment variables
            self.set_environment_variables(agent_type, **kwargs)
            
            # Get agent info
            agent_info = AVAILABLE_AGENTS[agent_type]
            module_path = agent_info['module']
            
            print(f"🚀 Starting {agent_info['description']}")
            print(f"   Module: {module_path}")
            if self.agent_name:
                print(f"   Instance: {self.agent_name}")
            print(f"   Process ID: {os.getpid()}")
            print()
            
            # Import and run the agent module
            try:
                module = importlib.import_module(module_path)
                
                # Check if module has main function
                if hasattr(module, 'main'):
                    await module.main()
                else:
                    print(f"❌ Module {module_path} does not have a main() function")
                    return 1
                    
            except ImportError as e:
                print(f"❌ Failed to import agent module {module_path}: {e}")
                return 1
            
            return 0
            
        except KeyboardInterrupt:
            print(f"\n⚠️  Received keyboard interrupt")
            return 0
        except Exception as e:
            print(f"❌ Failed to start agent: {e}")
            import traceback
            traceback.print_exc()
            return 1


def parse_arguments():
    """Parse command line arguments"""
    parser = argparse.ArgumentParser(
        description="Universal agent starter for the trading system",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=f"""
Available Agents:
{chr(10).join(f"  {k:12} - {v['description']}" for k, v in AVAILABLE_AGENTS.items())}

Examples:
  {sys.argv[0]} technical --name ta-agent-1 --log-level DEBUG
  {sys.argv[0]} news --name news-1 --nats-url nats://prod:4222
  {sys.argv[0]} strategy --config-preset production
        """
    )
    
    # Required arguments
    parser.add_argument(
        "agent",
        choices=list(AVAILABLE_AGENTS.keys()) + ['list'],
        help="Agent type to start or 'list' to show available agents"
    )
    
    # Optional arguments
    parser.add_argument(
        "--name",
        help="Agent instance name (overrides environment variable)"
    )
    
    parser.add_argument(
        "--log-level",
        choices=['DEBUG', 'INFO', 'WARN', 'ERROR'],
        help="Logging level"
    )
    
    parser.add_argument(
        "--nats-url",
        help="NATS server URL (e.g., nats://localhost:4222)"
    )
    
    parser.add_argument(
        "--config-preset",
        choices=['development', 'production', 'high_frequency', 'long_term'],
        help="Use predefined configuration preset"
    )
    
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be started without actually starting"
    )
    
    # Configuration overrides
    parser.add_argument(
        "--config-override",
        action="append",
        help="Override configuration value (format: key=value)"
    )
    
    return parser.parse_args()


async def main():
    """Main entry point"""
    args = parse_arguments()
    
    # Handle list command
    if args.agent == 'list':
        starter = AgentStarter()
        starter.list_agents()
        return 0
    
    # Create agent starter
    starter = AgentStarter()
    
    # Process configuration overrides
    if args.config_override:
        for override in args.config_override:
            if '=' not in override:
                print(f"❌ Invalid config override format: {override} (expected key=value)")
                return 1
            key, value = override.split('=', 1)
            starter.config_overrides[key] = value
    
    # Set preset configuration if specified
    if args.config_preset:
        os.environ['CONFIG_PRESET'] = args.config_preset
    
    # Dry run mode
    if args.dry_run:
        print(f"🧪 Dry Run Mode - Would start:")
        print(f"   Agent Type: {args.agent}")
        print(f"   Agent Name: {args.name or 'default'}")
        print(f"   Log Level: {args.log_level or 'INFO'}")
        print(f"   NATS URL: {args.nats_url or 'nats://localhost:4222'}")
        if args.config_preset:
            print(f"   Config Preset: {args.config_preset}")
        if starter.config_overrides:
            print(f"   Config Overrides: {starter.config_overrides}")
        return 0
    
    # Start the agent
    exit_code = await starter.start_agent(
        args.agent,
        name=args.name,
        log_level=args.log_level,
        nats_url=args.nats_url
    )
    
    return exit_code


if __name__ == "__main__":
    try:
        exit_code = asyncio.run(main())
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print(f"\n⚠️  Startup interrupted")
        sys.exit(0)
    except Exception as e:
        print(f"❌ Startup failed: {e}")
        sys.exit(1)