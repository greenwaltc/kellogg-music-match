#!/bin/bash

# Build script for Option 1: Enhanced Init Container with MusicBrainz data
# This script builds all required Docker images for the Kubernetes deployment

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

echo "🔨 Building Kellogg Music Match Docker Images..."

# Build backend image
echo "📦 Building backend image..."
cd backend
docker build -t kellogg-music-match-backend:latest .
cd ..

# Build UI image  
echo "🎨 Building UI image..."
cd ui
docker build -t kellogg-music-match-ui:latest .
cd ..

# Build MusicBrainz data loader image
echo "🎵 Building MusicBrainz data loader image..."
if [ ! -f "musicbrainz_artists_50k.csv" ]; then
    echo "❌ Error: musicbrainz_artists_50k.csv not found!"
    echo "Please ensure the CSV file exists in the project root"
    exit 1
fi

docker build -f Dockerfile.musicbrainz -t kellogg-music-match-musicbrainz:latest .

echo "✅ All Docker images built successfully!"
echo ""
echo "📋 Built images:"
echo "  - kellogg-music-match-backend:latest"
echo "  - kellogg-music-match-ui:latest"
echo "  - kellogg-music-match-musicbrainz:latest"
echo ""
echo "🚀 Ready for Pulumi deployment:"
echo "  cd pulumi && pulumi up"