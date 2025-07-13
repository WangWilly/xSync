#!/bin/bash

# xSync ChromaDB Development Environment Launcher
# This script launches ChromaDB and Redis for the xSync tweet embedding system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="deployment-dev/docker-compose.yml"
PROJECT_NAME="xsync"

# Functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        print_error "Docker Compose is not installed or not in PATH"
        exit 1
    fi
}

check_compose_file() {
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_error "Docker Compose file not found: $COMPOSE_FILE"
        print_info "Please run this script from the project root directory"
        exit 1
    fi
}

start_services() {
    print_info "Starting ChromaDB and Redis services..."
    
    # Use docker compose (new syntax) or docker-compose (legacy)
    if docker compose version &> /dev/null; then
        docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d
    else
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d
    fi
    
    print_success "Services started successfully!"
}

stop_services() {
    print_info "Stopping ChromaDB and Redis services..."
    
    if docker compose version &> /dev/null; then
        docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    else
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    fi
    
    print_success "Services stopped successfully!"
}

restart_services() {
    print_info "Restarting ChromaDB and Redis services..."
    stop_services
    start_services
}

show_status() {
    print_info "Service status:"
    
    if docker compose version &> /dev/null; then
        docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
    else
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
    fi
}

show_logs() {
    local service=${1:-""}
    
    if [ -n "$service" ]; then
        print_info "Showing logs for service: $service"
        if docker compose version &> /dev/null; then
            docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f "$service"
        else
            docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f "$service"
        fi
    else
        print_info "Showing logs for all services:"
        if docker compose version &> /dev/null; then
            docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f
        else
            docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f
        fi
    fi
}

wait_for_services() {
    print_info "Waiting for services to be healthy..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        local chromadb_health=$(docker inspect --format='{{.State.Health.Status}}' xsync-chromadb 2>/dev/null || echo "unhealthy")
        local redis_health=$(docker inspect --format='{{.State.Health.Status}}' xsync-redis 2>/dev/null || echo "unhealthy")
        
        if [ "$chromadb_health" = "healthy" ] && [ "$redis_health" = "healthy" ]; then
            print_success "All services are healthy and ready!"
            break
        fi
        
        print_info "Attempt $attempt/$max_attempts - ChromaDB: $chromadb_health, Redis: $redis_health"
        sleep 2
        attempt=$((attempt + 1))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        print_warning "Services may not be fully ready yet. Check logs if you encounter issues."
    fi
}

show_endpoints() {
    print_success "Service endpoints:"
    echo ""
    echo -e "${GREEN}ChromaDB:${NC}"
    echo -e "  HTTP API: ${BLUE}http://localhost:8000${NC}"
    echo -e "  Health Check: ${BLUE}http://localhost:8000/api/v1/heartbeat${NC}"
    echo -e "  Auth Token: ${YELLOW}xsync-dev-token-2025${NC}"
    echo ""
    echo -e "${GREEN}Redis:${NC}"
    echo -e "  Host: ${BLUE}localhost:6379${NC}"
    echo -e "  Password: ${YELLOW}xsync-redis-2025${NC}"
    echo ""
    echo -e "${GREEN}Usage Examples:${NC}"
    echo -e "  Test ChromaDB: ${BLUE}curl -H 'X-Chroma-Token: xsync-dev-token-2025' http://localhost:8000/api/v1/heartbeat${NC}"
    echo -e "  Test Redis: ${BLUE}redis-cli -h localhost -p 6379 -a xsync-redis-2025 ping${NC}"
}

show_help() {
    echo "xSync ChromaDB Development Environment Launcher"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  start     Start ChromaDB and Redis services"
    echo "  stop      Stop ChromaDB and Redis services"
    echo "  restart   Restart ChromaDB and Redis services"
    echo "  status    Show service status"
    echo "  logs      Show logs for all services"
    echo "  logs [SERVICE]  Show logs for specific service (chromadb, redis)"
    echo "  wait      Wait for services to be healthy"
    echo "  endpoints Show service endpoints and credentials"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start              # Start all services"
    echo "  $0 logs chromadb      # Show ChromaDB logs"
    echo "  $0 status             # Check service status"
}

# Main script logic
main() {
    local command=${1:-"start"}
    
    # Check prerequisites
    check_docker
    check_compose_file
    
    case $command in
        "start")
            start_services
            wait_for_services
            show_endpoints
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            restart_services
            wait_for_services
            show_endpoints
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs "$2"
            ;;
        "wait")
            wait_for_services
            ;;
        "endpoints")
            show_endpoints
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            print_error "Unknown command: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
