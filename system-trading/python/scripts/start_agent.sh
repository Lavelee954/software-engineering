#!/bin/bash
# Start Technical Analysis Agent Process

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default configuration
AGENT_NAME="${TECHNICAL_AGENT_NAME:-technical-analysis-agent}"
NATS_URL="${NATS_URL:-nats://localhost:4222}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
PYTHON_ENV="${PYTHON_ENV:-development}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if NATS is available
check_nats() {
    log_info "Checking NATS connection at $NATS_URL..."
    
    # Extract host and port from NATS URL
    NATS_HOST=$(echo "$NATS_URL" | sed 's|nats://||' | cut -d: -f1)
    NATS_PORT=$(echo "$NATS_URL" | sed 's|nats://||' | cut -d: -f2)
    
    if command -v nc >/dev/null 2>&1; then
        if nc -z "$NATS_HOST" "$NATS_PORT" 2>/dev/null; then
            log_success "NATS server is available"
            return 0
        else
            log_error "NATS server is not available at $NATS_HOST:$NATS_PORT"
            return 1
        fi
    else
        log_warning "netcat not available, skipping NATS check"
        return 0
    fi
}

# Function to check Python environment
check_python_env() {
    log_info "Checking Python environment..."
    
    if [[ -f "$PROJECT_DIR/venv/bin/python" ]]; then
        PYTHON_CMD="$PROJECT_DIR/venv/bin/python"
        log_success "Using virtual environment: $PYTHON_CMD"
    elif command -v python3 >/dev/null 2>&1; then
        PYTHON_CMD="python3"
        log_success "Using system Python: $PYTHON_CMD"
    else
        log_error "Python not found"
        exit 1
    fi
    
    # Check if required packages are available
    if ! "$PYTHON_CMD" -c "import agents.technical_analysis" 2>/dev/null; then
        log_error "Technical analysis agent module not found. Run 'pip install -e .' first."
        exit 1
    fi
}

# Function to create necessary directories
setup_directories() {
    log_info "Setting up directories..."
    
    mkdir -p "$PROJECT_DIR/logs"
    mkdir -p "$PROJECT_DIR/tmp"
    
    log_success "Directories created"
}

# Function to start the agent
start_agent() {
    log_info "Starting Technical Analysis Agent..."
    log_info "Configuration:"
    log_info "  Agent Name: $AGENT_NAME"
    log_info "  NATS URL: $NATS_URL"
    log_info "  Log Level: $LOG_LEVEL"
    log_info "  Environment: $PYTHON_ENV"
    log_info "  Working Directory: $PROJECT_DIR"
    log_info "  Process ID: $$"
    
    cd "$PROJECT_DIR"
    
    # Export environment variables
    export TECHNICAL_AGENT_NAME="$AGENT_NAME"
    export NATS_URL="$NATS_URL"
    export LOG_LEVEL="$LOG_LEVEL"
    export PYTHON_ENV="$PYTHON_ENV"
    
    # Start the agent
    exec "$PYTHON_CMD" run_technical_analysis.py
}

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -n, --name NAME         Agent name (default: technical-analysis-agent)"
    echo "  -u, --nats-url URL      NATS server URL (default: nats://localhost:4222)"
    echo "  -l, --log-level LEVEL   Log level (default: INFO)"
    echo "  -e, --env ENV           Environment (default: development)"
    echo "  -h, --help              Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  TECHNICAL_AGENT_NAME    Agent name"
    echo "  NATS_URL               NATS server URL"
    echo "  LOG_LEVEL              Log level (DEBUG, INFO, WARN, ERROR)"
    echo "  PYTHON_ENV             Environment (development, production)"
    echo ""
    echo "Examples:"
    echo "  $0                                          # Start with defaults"
    echo "  $0 --name ta-agent-1 --log-level DEBUG     # Custom name and debug logging"
    echo "  $0 --nats-url nats://production:4222       # Production NATS server"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            AGENT_NAME="$2"
            shift 2
            ;;
        -u|--nats-url)
            NATS_URL="$2"
            shift 2
            ;;
        -l|--log-level)
            LOG_LEVEL="$2"
            shift 2
            ;;
        -e|--env)
            PYTHON_ENV="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    log_info "Technical Analysis Agent Startup Script"
    log_info "========================================"
    
    # Pre-flight checks
    check_python_env
    setup_directories
    
    # Check NATS availability (non-blocking)
    if ! check_nats; then
        log_warning "NATS server check failed, but continuing anyway..."
    fi
    
    # Start the agent
    start_agent
}

# Trap signals for graceful shutdown
trap 'log_info "Received shutdown signal, stopping agent..."; exit 0' SIGTERM SIGINT

# Run main function
main "$@"