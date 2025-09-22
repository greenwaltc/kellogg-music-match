# Custom PostgreSQL image with Python scientific libraries
FROM postgres:16

# Install Python dependencies as root
USER root

# Install Python and PostgreSQL Python extension
RUN apt-get update && apt-get install -y \
    python3-pip \
    python3-dev \
    python3-numpy \
    python3-scipy \
    postgresql-plpython3-16 \
    && rm -rf /var/lib/apt/lists/*

# Verify installation
RUN python3 -c "import scipy.stats; import numpy; print('✅ scipy and numpy are available')"

# Switch back to postgres user
USER postgres

# Add health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=5 \
    CMD pg_isready -U $POSTGRES_USER -d $POSTGRES_DB