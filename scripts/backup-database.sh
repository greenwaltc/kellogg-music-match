#!/bin/bash

# PostgreSQL Database Backup Script for k3s
# This script creates a backup of the Kellogg Music Match database

set -e

# Configuration
NAMESPACE="kmm"
POD_NAME="postgres-0"
DB_NAME="kellogg_music_match"
DB_USER="kellogg_user"
BACKUP_DIR="/home/cameron/backups/kellogg-music-match"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="kellogg_music_match_backup_${TIMESTAMP}.sql"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🗄️  Kellogg Music Match Database Backup${NC}"
echo "========================================"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl is not installed or not in PATH${NC}"
    exit 1
fi

# Check if the pod is running
echo -e "${YELLOW}📡 Checking database pod status...${NC}"
if ! kubectl get pod -n ${NAMESPACE} ${POD_NAME} &> /dev/null; then
    echo -e "${RED}❌ Pod ${POD_NAME} not found in namespace ${NAMESPACE}${NC}"
    exit 1
fi

POD_STATUS=$(kubectl get pod -n ${NAMESPACE} ${POD_NAME} -o jsonpath='{.status.phase}')
if [ "$POD_STATUS" != "Running" ]; then
    echo -e "${RED}❌ Pod ${POD_NAME} is not running (status: ${POD_STATUS})${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Database pod is running${NC}"

# Create backup directory
echo -e "${YELLOW}📁 Creating backup directory...${NC}"
mkdir -p ${BACKUP_DIR}

# Create database backup
echo -e "${YELLOW}💾 Creating database backup...${NC}"
kubectl exec -n ${NAMESPACE} ${POD_NAME} -- pg_dump -U ${DB_USER} -d ${DB_NAME} --verbose --clean --if-exists --create > "${BACKUP_DIR}/${BACKUP_FILE}"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Backup created successfully: ${BACKUP_DIR}/${BACKUP_FILE}${NC}"
    
    # Get backup file size
    BACKUP_SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_FILE}" | cut -f1)
    echo -e "${GREEN}📊 Backup size: ${BACKUP_SIZE}${NC}"
    
    # Show some statistics
    echo -e "${YELLOW}📈 Database statistics:${NC}"
    kubectl exec -n ${NAMESPACE} ${POD_NAME} -- psql -U ${DB_USER} -d ${DB_NAME} -c "
        SELECT 
            'Users' as table_name, COUNT(*) as count FROM users
        UNION ALL
        SELECT 
            'Artists' as table_name, COUNT(*) as count FROM artists
        UNION ALL
        SELECT 
            'User Artists' as table_name, COUNT(*) as count FROM user_artists
        UNION ALL
        SELECT 
            'Matches' as table_name, COUNT(*) as count FROM matches;" 2>/dev/null
    
    # Cleanup old backups (keep last 7)
    echo -e "${YELLOW}🧹 Cleaning up old backups (keeping last 7)...${NC}"
    ls -t ${BACKUP_DIR}/kellogg_music_match_backup_*.sql 2>/dev/null | tail -n +8 | xargs -r rm -f
    
    echo -e "${GREEN}🎉 Backup completed successfully!${NC}"
    echo -e "${GREEN}📍 Location: ${BACKUP_DIR}/${BACKUP_FILE}${NC}"
    
else
    echo -e "${RED}❌ Backup failed${NC}"
    exit 1
fi

# Optional: Compress the backup
if command -v gzip &> /dev/null; then
    echo -e "${YELLOW}🗜️  Compressing backup...${NC}"
    gzip "${BACKUP_DIR}/${BACKUP_FILE}"
    echo -e "${GREEN}✅ Backup compressed: ${BACKUP_DIR}/${BACKUP_FILE}.gz${NC}"
fi

echo -e "${GREEN}✨ All done!${NC}"