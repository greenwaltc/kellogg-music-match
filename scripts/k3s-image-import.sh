#!/bin/bash

# k3s Docker Image Import Script
# This script imports locally built Docker images directly into k3s containerd

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Configuration
IMAGES=(
    "kellogg-music-match-backend:latest"
    "kellogg-music-match-ui:latest"
    "kellogg-music-match-postgres:latest"
)

# External (upstream) images we rely on at runtime but do not build locally.
# These will be docker pulled (if not already) and then imported into k3s.
EXTERNAL_IMAGES=(
    "flyway/flyway:10-alpine"  # Pin Flyway version used by init container
)

NAMESPACE="kmm"

# Function to import a single image
import_image() {
    local image_name="$1"
    log "Importing Docker image: $image_name"
    
    # Check if image exists in Docker
    if ! docker image inspect "$image_name" >/dev/null 2>&1; then
        error "Docker image '$image_name' not found. Please build it first."
        return 1
    fi
    
    # Save Docker image to tar file
    local tar_file="/tmp/${image_name//[:\/]/_}.tar"
    log "Exporting Docker image to $tar_file"
    docker save "$image_name" -o "$tar_file"
    
    # Import into k3s containerd
    log "Importing into k3s containerd"
    sudo k3s ctr images import "$tar_file"
    
    # Clean up tar file
    rm -f "$tar_file"
    
    success "Successfully imported $image_name"
}

# Function to build and import all images
build_and_import() {
    log "Building all Docker images first..."
    
    # Navigate to project root directory
    cd "$(dirname "$0")/.."
    
    # Build all images
    if command -v make >/dev/null 2>&1; then
        log "Using Makefile to build images"
        make docker-build
    else
        log "Using docker-compose to build images"
        docker-compose build
    fi
    
    success "All locally built images processed"

    # Pull external images (pinning versions) so they exist locally
    for ext in "${EXTERNAL_IMAGES[@]}"; do
        log "Ensuring external image present: $ext"
        if ! docker image inspect "$ext" >/dev/null 2>&1; then
            docker pull "$ext"
        fi
        import_image "$ext"
    done

    # Import each local project image (skip if missing to avoid hard failure when optional)
    for image in "${IMAGES[@]}"; do
        if docker image inspect "$image" >/dev/null 2>&1; then
            import_image "$image"
        else
            warn "Skipping missing local image: $image"
        fi
    done
}

# Function to setup local registry (alternative approach)
setup_local_registry() {
    log "Setting up local Docker registry for k3s..."
    
    # Create registry configuration
    sudo mkdir -p /etc/rancher/k3s
    
    cat << EOF | sudo tee /etc/rancher/k3s/registries.yaml
mirrors:
  docker.io:
    endpoint:
      - "https://registry-1.docker.io"
  localhost:5000:
    endpoint:
      - "http://localhost:5000"
configs:
  "localhost:5000":
    insecure: true
EOF

    # Start local registry if not running
    if ! docker ps | grep -q registry:2; then
        log "Starting local Docker registry..."
        docker run -d \
            --name registry \
            --restart=always \
            -p 5000:5000 \
            registry:2
        
        success "Local registry started on localhost:5000"
    else
        log "Local registry already running"
    fi
    
    # Restart k3s to pick up new registry config
    log "Restarting k3s to apply registry configuration..."
    sudo systemctl restart k3s
    
    success "k3s configured to use local registry"
}

# Function to push images to local registry
push_to_local_registry() {
    log "Pushing images to local registry..."
    
    for image in "${IMAGES[@]}"; do
        local local_tag="localhost:5000/$image"
        
        log "Tagging $image as $local_tag"
        docker tag "$image" "$local_tag"
        
        log "Pushing $local_tag"
        docker push "$local_tag"
        
        success "Pushed $image to local registry"
    done
}

# Function to show current images in k3s
show_k3s_images() {
    log "Current images in k3s containerd:"
    sudo k3s ctr images list | grep -E "(kellogg|DIGEST)" | head -20
}

# Function to clean up unused images
cleanup_images() {
    log "Cleaning up unused images in k3s..."
    sudo k3s ctr images prune
    success "Cleanup completed"
}

# Main script logic
case "${1:-import}" in
    "import"|"")
        build_and_import
        show_k3s_images
        ;;
    "registry")
        setup_local_registry
        ;;
    "push")
        push_to_local_registry
        ;;
    "show")
        show_k3s_images
        ;;
    "cleanup")
        cleanup_images
        ;;
    "help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  import    - Build Docker images and import into k3s (default)"
        echo "  registry  - Setup local Docker registry for k3s"
        echo "  push      - Push images to local registry"
        echo "  show      - Show current k3s images"
        echo "  cleanup   - Clean up unused k3s images"
        echo "  help      - Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0                 # Build and import all images"
        echo "  $0 import          # Same as above"
        echo "  $0 registry        # Setup local registry"
        echo "  $0 show            # Show current images"
        ;;
    *)
        error "Unknown command: $1"
        echo "Run '$0 help' for usage information"
        exit 1
        ;;
esac