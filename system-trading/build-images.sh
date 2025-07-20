#!/bin/bash

# Trading System Docker Image Build Script
# Builds both Go Trading Core and Python Agents images

set -euo pipefail

# Configuration
REGISTRY="${DOCKER_REGISTRY:-localhost:5000}"
VERSION="${VERSION:-$(git rev-parse --short HEAD || echo 'latest')}"
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Build Go Trading Core image
build_go_image() {
    local image_name="$REGISTRY/trading-core:$VERSION"
    local latest_tag="$REGISTRY/trading-core:latest"
    
    log_info "Building Go Trading Core image..."
    log_info "Image: $image_name"
    
    docker build \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VERSION="$VERSION" \
        --tag "$image_name" \
        --tag "$latest_tag" \
        --file Dockerfile \
        .
    
    log_success "Go Trading Core image built successfully"
    return 0
}

# Build Python Agents image
build_python_image() {
    local image_name="$REGISTRY/python-agents:$VERSION"
    local latest_tag="$REGISTRY/python-agents:latest"
    
    log_info "Building Python Agents image..."
    log_info "Image: $image_name"
    
    docker build \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VERSION="$VERSION" \
        --tag "$image_name" \
        --tag "$latest_tag" \
        --file python/Dockerfile \
        python/
    
    log_success "Python Agents image built successfully"
    return 0
}

# Push images to registry
push_images() {
    if [ "$REGISTRY" == "localhost:5000" ]; then
        log_warning "Skipping push - using local registry"
        return 0
    fi
    
    log_info "Pushing images to registry..."
    
    docker push "$REGISTRY/trading-core:$VERSION"
    docker push "$REGISTRY/trading-core:latest"
    docker push "$REGISTRY/python-agents:$VERSION"
    docker push "$REGISTRY/python-agents:latest"
    
    log_success "Images pushed successfully"
}

# Display image sizes
show_image_info() {
    log_info "Image information:"
    echo ""
    docker images | grep -E "(trading-core|python-agents)" | head -10
    echo ""
}

# Cleanup old images
cleanup_old_images() {
    log_info "Cleaning up old images..."
    
    # Remove untagged images
    docker image prune -f >/dev/null 2>&1 || true
    
    # Remove old tagged images (keep last 5)
    docker images "$REGISTRY/trading-core" --format "table {{.Tag}}\t{{.ID}}" | \
        tail -n +6 | awk '{print $2}' | xargs -r docker rmi >/dev/null 2>&1 || true
    
    docker images "$REGISTRY/python-agents" --format "table {{.Tag}}\t{{.ID}}" | \
        tail -n +6 | awk '{print $2}' | xargs -r docker rmi >/dev/null 2>&1 || true
    
    log_success "Cleanup completed"
}

# Test images
test_images() {
    log_info "Testing built images..."
    
    # Test Go image
    log_info "Testing Trading Core image..."
    docker run --rm "$REGISTRY/trading-core:$VERSION" --version || \
        docker run --rm "$REGISTRY/trading-core:$VERSION" --help || true
    
    # Test Python image
    log_info "Testing Python Agents image..."
    docker run --rm "$REGISTRY/python-agents:$VERSION" python -c "import agents.technical_analysis; print('✓ Technical Analysis')"
    docker run --rm "$REGISTRY/python-agents:$VERSION" python -c "import shared.central_router; print('✓ Central Router')"
    
    log_success "Image tests completed"
}

# Show usage
show_usage() {
    cat << EOF
Trading System Docker Build Script

Usage: $0 [OPTIONS] [COMMAND]

Commands:
    build       Build both images (default)
    go          Build only Go Trading Core image
    python      Build only Python Agents image
    push        Push images to registry
    test        Test built images
    cleanup     Clean up old images
    all         Build, test, and push images

Options:
    -r, --registry  Docker registry (default: localhost:5000)
    -v, --version   Image version (default: git short hash or 'latest')
    -h, --help      Show this help

Environment Variables:
    DOCKER_REGISTRY  Docker registry URL
    VERSION          Image version tag

Examples:
    $0                                    # Build both images
    $0 -r myregistry.io -v v1.0.0 all    # Build, test, and push with custom registry
    $0 go                                 # Build only Go image
    $0 python                             # Build only Python image

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            build|go|python|push|test|cleanup|all)
                COMMAND="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

# Main execution
main() {
    local command="${COMMAND:-build}"
    
    log_info "=== Trading System Docker Build ==="
    log_info "Registry: $REGISTRY"
    log_info "Version: $VERSION"
    log_info "Command: $command"
    echo ""
    
    case $command in
        build)
            build_go_image
            build_python_image
            show_image_info
            ;;
        go)
            build_go_image
            show_image_info
            ;;
        python)
            build_python_image
            show_image_info
            ;;
        push)
            push_images
            ;;
        test)
            test_images
            ;;
        cleanup)
            cleanup_old_images
            ;;
        all)
            build_go_image
            build_python_image
            test_images
            show_image_info
            push_images
            cleanup_old_images
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
    
    log_success "Build script completed successfully!"
}

# Initialize
COMMAND=""

# Parse arguments and run
parse_args "$@"
main 