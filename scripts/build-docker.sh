#!/bin/bash

# KubeGraph Docker Build Script
# This script builds the binary locally and then creates a Docker image

set -e  # Exit on any error

# Default values
IMAGE_NAME="dcarias/kubegraph"
TAG="latest"
PUSH=false
PLATFORM="linux/amd64"
BUILD_ARGS=""
BUILD_LOCAL=true
DOCKERFILE="Dockerfile.local"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
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

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build and optionally push the KubeGraph Docker image.

OPTIONS:
    -t, --tag TAG           Docker image tag (default: latest)
    -p, --push              Push the image to Docker Hub after building
    --platform PLATFORM     Target platform (default: linux/amd64)
    --no-cache              Build without using cache
    --build-arg KEY=VALUE   Add build argument
    --docker-build          Build binary inside Docker (default: build locally)
    --dockerfile FILE       Use specific Dockerfile (default: Dockerfile.local)
    -h, --help              Show this help message

EXAMPLES:
    # Build with default settings (local build)
    $0

    # Build with specific tag
    $0 -t v1.0.0

    # Build and push
    $0 -t v1.0.0 -p

    # Build binary inside Docker
    $0 --docker-build

    # Build for multiple platforms
    $0 --platform linux/amd64,linux/arm64

    # Build without cache
    $0 --no-cache

    # Build with custom build args
    $0 --build-arg VERSION=1.0.0 --build-arg BUILD_DATE=\$(date -u +%Y-%m-%dT%H:%M:%SZ)

ENVIRONMENT VARIABLES:
    DOCKER_USERNAME          Docker Hub username (required for push)
    DOCKER_PASSWORD          Docker Hub password/token (required for push)
    DOCKER_REGISTRY          Docker registry (default: docker.io)

EOF
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Check if we're in the right directory
    if [[ ! -f "go.mod" ]]; then
        print_error "go.mod not found. Please run this script from the project root directory."
        exit 1
    fi
    
    # Check if Go is installed (for local builds)
    if [[ "$BUILD_LOCAL" == true ]]; then
        if ! command -v go &> /dev/null; then
            print_error "Go is not installed. Please install Go first for local builds."
            exit 1
        fi
    fi
    
    print_success "Prerequisites check passed"
}

# Function to build binary locally
build_binary_local() {
    print_status "Building binary locally for Linux..."
    
    # Clean previous build
    if [[ -f "kubegraph" ]]; then
        rm kubegraph
    fi
    
    # Build the binary for Linux
    if GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo -o kubegraph .; then
        print_success "Binary built successfully for Linux: kubegraph"
    else
        print_error "Failed to build binary"
        exit 1
    fi
}

# Function to login to Docker Hub
docker_login() {
    if [[ "$PUSH" == true ]]; then
        print_status "Logging in to Docker Hub..."
        
        # Check if credentials are provided
        if [[ -z "$DOCKER_USERNAME" || -z "$DOCKER_PASSWORD" ]]; then
            print_error "DOCKER_USERNAME and DOCKER_PASSWORD environment variables are required for pushing."
            print_error "Please set them or use docker login manually."
            exit 1
        fi
        
        # Login to Docker Hub
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
        print_success "Logged in to Docker Hub"
    fi
}

# Function to build the image
build_image() {
    print_status "Building Docker image: $IMAGE_NAME:$TAG"
    
    # Prepare build command
    BUILD_CMD="docker build"
    
    # Add platform if specified
    if [[ "$PLATFORM" != "linux/amd64" ]]; then
        BUILD_CMD="$BUILD_CMD --platform $PLATFORM"
    fi
    
    # Add build args
    if [[ -n "$BUILD_ARGS" ]]; then
        BUILD_CMD="$BUILD_CMD $BUILD_ARGS"
    fi
    
    # Add tag
    BUILD_CMD="$BUILD_CMD -t $IMAGE_NAME:$TAG"
    
    # Add Dockerfile
    BUILD_CMD="$BUILD_CMD -f $DOCKERFILE"
    
    # Add context
    BUILD_CMD="$BUILD_CMD ."
    
    print_status "Running: $BUILD_CMD"
    
    # Execute build
    if eval "$BUILD_CMD"; then
        print_success "Docker image built successfully: $IMAGE_NAME:$TAG"
    else
        print_error "Failed to build Docker image"
        exit 1
    fi
}

# Function to push the image
push_image() {
    if [[ "$PUSH" == true ]]; then
        print_status "Pushing Docker image: $IMAGE_NAME:$TAG"
        
        if docker push "$IMAGE_NAME:$TAG"; then
            print_success "Docker image pushed successfully: $IMAGE_NAME:$TAG"
        else
            print_error "Failed to push Docker image"
            exit 1
        fi
    fi
}

# Function to show image info
show_image_info() {
    print_status "Image information:"
    echo "  Name: $IMAGE_NAME"
    echo "  Tag: $TAG"
    echo "  Platform: $PLATFORM"
    echo "  Push: $PUSH"
    echo "  Build Method: $([[ "$BUILD_LOCAL" == true ]] && echo "Local" || echo "Docker")"
    echo "  Dockerfile: $DOCKERFILE"
    
    if [[ -n "$BUILD_ARGS" ]]; then
        echo "  Build Args: $BUILD_ARGS"
    fi
    
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --no-cache)
            BUILD_ARGS="$BUILD_ARGS --no-cache"
            shift
            ;;
        --build-arg)
            BUILD_ARGS="$BUILD_ARGS --build-arg $2"
            shift 2
            ;;
        --docker-build)
            BUILD_LOCAL=false
            DOCKERFILE="Dockerfile"
            shift
            ;;
        --dockerfile)
            DOCKERFILE="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set default registry if not specified
DOCKER_REGISTRY=${DOCKER_REGISTRY:-"docker.io"}

# Main execution
main() {
    print_status "Starting KubeGraph Docker build process..."
    echo ""
    
    show_image_info
    check_prerequisites
    
    # Build binary locally if requested
    if [[ "$BUILD_LOCAL" == true ]]; then
        build_binary_local
    fi
    
    docker_login
    build_image
    push_image
    
    print_success "Docker build process completed successfully!"
    
    if [[ "$PUSH" == true ]]; then
        echo ""
        print_status "You can now pull the image with:"
        echo "  docker pull $IMAGE_NAME:$TAG"
        echo ""
        print_status "Or run it with:"
        echo "  docker run -e NEO4J_PASSWORD=your_password $IMAGE_NAME:$TAG"
    fi
}

# Run main function
main 
