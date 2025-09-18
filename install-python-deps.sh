#!/bin/bash
# Install Python dependencies for PostgreSQL plpython3u extension
# This script installs scipy and numpy for the spearman_distance function

set -e

echo "Installing Python dependencies for plpython3u..."

# Update package list and install pip if needed
apt-get update
apt-get install -y python3-pip python3-dev

# Install required Python packages
pip3 install --no-cache-dir scipy numpy

echo "Python dependencies installed successfully."
echo "Available packages:"
pip3 list | grep -E "(scipy|numpy)"

# Test the installation
python3 -c "import scipy.stats; import numpy; print('✅ scipy and numpy are available')"

echo "Setup complete. PostgreSQL plpython3u extension ready for advanced statistics."