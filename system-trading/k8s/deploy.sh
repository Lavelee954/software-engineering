#!/bin/bash

# Trading System Kubernetes Deployment Script
# This script deploys the complete Multi-Agent Trading System to Kubernetes

set -euo pipefail  # Exit on error, undefined variables, pipe failures

# Configuration
NAMESPACE="trading-system"
KUSTOMIZE_DIR="$(dirname "$0")"
TIMEOUT="600s"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    # Check if kustomize is installed
    if ! command -v kustomize &> /dev/null; then
        log_warning "kustomize not found, using kubectl apply -k instead"
        USE_KUSTOMIZE=false
    else
        USE_KUSTOMIZE=true
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Deploy infrastructure first
deploy_infrastructure() {
    log_info "Deploying infrastructure components..."
    
    # Create namespace first
    kubectl apply -f namespace/trading-system.yaml
    
    # Deploy storage
    kubectl apply -f storage/persistent-volumes.yaml
    
    # Deploy secrets and configmaps
    kubectl apply -f secrets/trading-secrets.yaml
    kubectl apply -f configmaps/
    
    # Deploy infrastructure in order
    log_info "Deploying NATS message bus..."
    kubectl apply -f infrastructure/nats/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=nats -n $NAMESPACE --timeout=$TIMEOUT
    
    log_info "Deploying PostgreSQL database..."
    kubectl apply -f infrastructure/postgresql/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgresql -n $NAMESPACE --timeout=$TIMEOUT
    
    log_info "Deploying Redis cache..."
    kubectl apply -f infrastructure/redis/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=redis -n $NAMESPACE --timeout=$TIMEOUT
    
    log_info "Deploying Prometheus monitoring..."
    kubectl apply -f infrastructure/prometheus/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n $NAMESPACE --timeout=$TIMEOUT
    
    log_info "Deploying Grafana dashboard..."
    kubectl apply -f infrastructure/grafana/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n $NAMESPACE --timeout=$TIMEOUT
    
    log_success "Infrastructure deployment completed"
}

# Deploy applications
deploy_applications() {
    log_info "Deploying application components..."
    
    # Deploy Central Router first (required for A2A communication)
    log_info "Deploying Central Router..."
    kubectl apply -f applications/python-agents/central-router.yaml
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=central-router -n $NAMESPACE --timeout=$TIMEOUT
    
    # Deploy Trading Core
    log_info "Deploying Trading Core..."
    kubectl apply -f applications/trading-core/
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=trading-core -n $NAMESPACE --timeout=$TIMEOUT
    
    # Deploy Python Analysis Agents
    log_info "Deploying Python Analysis Agents..."
    kubectl apply -f applications/python-agents/technical-analysis.yaml
    kubectl apply -f applications/python-agents/analysis-agents.yaml
    
    # Wait for agents to be ready
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=python-agent -n $NAMESPACE --timeout=$TIMEOUT
    
    log_success "Application deployment completed"
}

# Deploy ingress and networking
deploy_networking() {
    log_info "Deploying ingress and networking..."
    
    kubectl apply -f ingress/
    
    log_success "Networking deployment completed"
}

# Full deployment using kustomize
deploy_full() {
    log_info "Deploying complete trading system..."
    
    if [ "$USE_KUSTOMIZE" = true ]; then
        kustomize build $KUSTOMIZE_DIR | kubectl apply -f -
    else
        kubectl apply -k $KUSTOMIZE_DIR
    fi
    
    # Wait for all pods to be ready
    log_info "Waiting for all pods to be ready..."
    kubectl wait --for=condition=ready pods --all -n $NAMESPACE --timeout=$TIMEOUT
    
    log_success "Full deployment completed"
}

# Check deployment status
check_status() {
    log_info "Checking deployment status..."
    
    echo ""
    log_info "Namespace: $NAMESPACE"
    kubectl get namespaces $NAMESPACE -o wide
    
    echo ""
    log_info "Pods:"
    kubectl get pods -n $NAMESPACE -o wide
    
    echo ""
    log_info "Services:"
    kubectl get services -n $NAMESPACE -o wide
    
    echo ""
    log_info "Persistent Volume Claims:"
    kubectl get pvc -n $NAMESPACE -o wide
    
    echo ""
    log_info "Ingress:"
    kubectl get ingress -n $NAMESPACE -o wide
    
    echo ""
    log_info "HorizontalPodAutoscalers:"
    kubectl get hpa -n $NAMESPACE -o wide
}

# Get access information
get_access_info() {
    log_info "Getting access information..."
    
    echo ""
    echo "=== Trading System Access Information ==="
    echo ""
    
    # NodePort access
    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}')
    if [ -z "$NODE_IP" ]; then
        NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
    fi
    
    echo "NodePort Access:"
    echo "  Trading Core API: http://$NODE_IP:30080"
    echo "  Grafana Dashboard: http://$NODE_IP:30030 (admin/admin)"
    echo "  Prometheus: http://$NODE_IP:30090"
    echo ""
    
    # Ingress access (if configured)
    echo "Ingress Access (configure DNS):"
    echo "  Trading Core API: https://trading.yourdomain.com"
    echo "  Grafana Dashboard: https://grafana.yourdomain.com"
    echo "  Prometheus: https://prometheus.yourdomain.com"
    echo ""
    
    # Port forwarding commands
    echo "Port Forwarding Commands:"
    echo "  kubectl port-forward -n $NAMESPACE svc/trading-core-service 8080:8080"
    echo "  kubectl port-forward -n $NAMESPACE svc/grafana-service 3000:3000"
    echo "  kubectl port-forward -n $NAMESPACE svc/prometheus-service 9090:9090"
    echo ""
}

# Show help
show_help() {
    cat << EOF
Trading System Kubernetes Deployment Script

Usage: $0 [COMMAND]

Commands:
    deploy          Deploy the complete trading system (default)
    infrastructure  Deploy only infrastructure components
    applications    Deploy only application components
    networking      Deploy only ingress and networking
    status          Check deployment status
    access          Show access information
    delete          Delete the entire trading system
    help            Show this help message

Examples:
    $0                     # Deploy complete system
    $0 deploy              # Deploy complete system
    $0 infrastructure      # Deploy only infrastructure
    $0 status              # Check status
    $0 access              # Show access URLs

EOF
}

# Delete deployment
delete_deployment() {
    log_warning "This will delete the entire trading system!"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deleting trading system..."
        
        if [ "$USE_KUSTOMIZE" = true ]; then
            kustomize build $KUSTOMIZE_DIR | kubectl delete -f - --ignore-not-found=true
        else
            kubectl delete -k $KUSTOMIZE_DIR --ignore-not-found=true
        fi
        
        # Force delete namespace if it's stuck
        kubectl delete namespace $NAMESPACE --ignore-not-found=true --timeout=60s || true
        
        log_success "Trading system deleted"
    else
        log_info "Deletion cancelled"
    fi
}

# Main script logic
main() {
    local command="${1:-deploy}"
    
    case $command in
        deploy)
            check_prerequisites
            deploy_full
            check_status
            get_access_info
            ;;
        infrastructure)
            check_prerequisites
            deploy_infrastructure
            check_status
            ;;
        applications)
            check_prerequisites
            deploy_applications
            check_status
            ;;
        networking)
            check_prerequisites
            deploy_networking
            check_status
            ;;
        status)
            check_status
            ;;
        access)
            get_access_info
            ;;
        delete)
            delete_deployment
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@" 