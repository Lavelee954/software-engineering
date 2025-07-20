#!/bin/bash
# Stop Technical Analysis Agent Process

set -euo pipefail

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

# Default values
AGENT_NAME="${TECHNICAL_AGENT_NAME:-technical-analysis-agent}"
FORCE_KILL=false
TIMEOUT=30

# Function to find agent processes
find_agent_processes() {
    local pattern="run_technical_analysis.py"
    pgrep -f "$pattern" 2>/dev/null || true
}

# Function to stop agent gracefully
stop_agent_graceful() {
    local pids=("$@")
    local stopped_count=0
    
    for pid in "${pids[@]}"; do
        if [[ -n "$pid" ]]; then
            log_info "Sending SIGTERM to process $pid..."
            if kill -TERM "$pid" 2>/dev/null; then
                ((stopped_count++))
            else
                log_warning "Failed to send SIGTERM to process $pid"
            fi
        fi
    done
    
    if [[ $stopped_count -gt 0 ]]; then
        log_info "Waiting up to $TIMEOUT seconds for graceful shutdown..."
        
        local wait_time=0
        while [[ $wait_time -lt $TIMEOUT ]]; do
            local running_pids=()
            for pid in "${pids[@]}"; do
                if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                    running_pids+=("$pid")
                fi
            done
            
            if [[ ${#running_pids[@]} -eq 0 ]]; then
                log_success "All agents stopped gracefully"
                return 0
            fi
            
            sleep 1
            ((wait_time++))
        done
        
        log_warning "Timeout reached, some processes may still be running"
        return 1
    else
        log_warning "No processes were sent shutdown signals"
        return 1
    fi
}

# Function to force kill agent processes
force_kill_agents() {
    local pids=("$@")
    local killed_count=0
    
    for pid in "${pids[@]}"; do
        if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
            log_warning "Force killing process $pid..."
            if kill -KILL "$pid" 2>/dev/null; then
                ((killed_count++))
            else
                log_error "Failed to force kill process $pid"
            fi
        fi
    done
    
    if [[ $killed_count -gt 0 ]]; then
        log_success "Force killed $killed_count processes"
    fi
}

# Function to show process status
show_status() {
    local pids
    maparray pids < <(find_agent_processes)
    
    if [[ ${#pids[@]} -eq 0 ]]; then
        log_info "No Technical Analysis Agent processes found"
        return 0
    fi
    
    log_info "Found ${#pids[@]} Technical Analysis Agent process(es):"
    for pid in "${pids[@]}"; do
        if [[ -n "$pid" ]]; then
            # Get process info
            local cmd
            cmd=$(ps -p "$pid" -o args= 2>/dev/null || echo "Unknown")
            log_info "  PID $pid: $cmd"
        fi
    done
}

# Function to show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -f, --force             Force kill processes (SIGKILL)"
    echo "  -t, --timeout SECONDS   Timeout for graceful shutdown (default: 30)"
    echo "  -s, --status            Show process status only"
    echo "  -h, --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                      # Graceful shutdown"
    echo "  $0 --force              # Force kill"
    echo "  $0 --timeout 60         # Wait 60 seconds for graceful shutdown"
    echo "  $0 --status             # Show running processes"
}

# Parse command line arguments
SHOW_STATUS_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--force)
            FORCE_KILL=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -s|--status)
            SHOW_STATUS_ONLY=true
            shift
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
    log_info "Technical Analysis Agent Stop Script"
    log_info "===================================="
    
    # Show status
    show_status
    
    if [[ "$SHOW_STATUS_ONLY" == true ]]; then
        exit 0
    fi
    
    # Find running processes
    local pids
    maparray pids < <(find_agent_processes)
    
    if [[ ${#pids[@]} -eq 0 ]]; then
        log_success "No Technical Analysis Agent processes to stop"
        exit 0
    fi
    
    if [[ "$FORCE_KILL" == true ]]; then
        # Force kill immediately
        force_kill_agents "${pids[@]}"
    else
        # Try graceful shutdown first
        if ! stop_agent_graceful "${pids[@]}"; then
            log_warning "Graceful shutdown failed or timed out"
            
            # Check for remaining processes
            local remaining_pids
            maparray remaining_pids < <(find_agent_processes)
            
            if [[ ${#remaining_pids[@]} -gt 0 ]]; then
                log_warning "Force killing remaining processes..."
                force_kill_agents "${remaining_pids[@]}"
            fi
        fi
    fi
    
    # Final status check
    sleep 1
    local final_pids
    maparray final_pids < <(find_agent_processes)
    
    if [[ ${#final_pids[@]} -eq 0 ]]; then
        log_success "All Technical Analysis Agent processes stopped"
        exit 0
    else
        log_error "Some processes may still be running:"
        show_status
        exit 1
    fi
}

# Run main function
main "$@"