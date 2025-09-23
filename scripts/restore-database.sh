#!/bin/bash

# PostgreSQL Database Restore Script for k3s
# This script restores a backup of the Kellogg Music Match database

set -e

# Configuration
NAMESPACE="kmm"
POD_NAME="postgres-0"
DB_NAME="kellogg_music_match"
DB_USER="kellogg_user"
BACKUP_DIR="/home/cameron/backups/kellogg-music-match"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}🔄 Kellogg Music Match Database Restore${NC}"
echo "========================================"

# Check if backup file is provided
if [ $# -eq 0 ]; then
    echo -e "${YELLOW}📋 Available backups:${NC}"
    ls -la ${BACKUP_DIR}/*.sql* 2>/dev/null | awk '{print $9, $5}' | column -t || echo "No backups found in ${BACKUP_DIR}"
    echo ""
    echo -e "${BLUE}Usage: $0 <backup_file>${NC}"
    echo -e "${BLUE}Example: $0 kellogg_music_match_backup_20240923_143022.sql${NC}"
    exit 1
fi

BACKUP_FILE="$1"

# Check if backup file exists (try with and without path)
if [ ! -f "${BACKUP_FILE}" ]; then
    if [ -f "${BACKUP_DIR}/${BACKUP_FILE}" ]; then
        BACKUP_FILE="${BACKUP_DIR}/${BACKUP_FILE}"
    elif [ -f "${BACKUP_DIR}/${BACKUP_FILE}.gz" ]; then
        BACKUP_FILE="${BACKUP_DIR}/${BACKUP_FILE}.gz"
        echo -e "${YELLOW}📦 Found compressed backup, will decompress first${NC}"
    else
        echo -e "${RED}❌ Backup file not found: ${BACKUP_FILE}${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}📁 Using backup file: ${BACKUP_FILE}${NC}"

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

# Confirm restore operation
echo -e "${RED}⚠️  WARNING: This will replace all data in the database!${NC}"
read -p "Are you sure you want to restore from ${BACKUP_FILE}? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}❌ Restore cancelled${NC}"
    exit 1
fi

# Handle compressed backups
TEMP_FILE=""
if [[ "${BACKUP_FILE}" == *.gz ]]; then
    echo -e "${YELLOW}📦 Decompressing backup...${NC}"
    TEMP_FILE="/tmp/$(basename ${BACKUP_FILE%.gz})"
    gunzip -c "${BACKUP_FILE}" > "${TEMP_FILE}"
    BACKUP_FILE="${TEMP_FILE}"
fi

# Stop any applications that might be using the database
echo -e "${YELLOW}🛑 Scaling down backend deployment...${NC}"
kubectl scale deployment -n ${NAMESPACE} kmm-backend --replicas=0

# Wait for backend pods to terminate
echo -e "${YELLOW}⏳ Waiting for backend pods to terminate...${NC}"
kubectl wait --for=delete pod -n ${NAMESPACE} -l component=backend --timeout=60s || true

# Perform the restore
echo -e "${YELLOW}🔄 Restoring database...${NC}"
cat "${BACKUP_FILE}" | kubectl exec -i -n ${NAMESPACE} ${POD_NAME} -- psql -U ${DB_USER}

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Database restored successfully${NC}"
    
    # Show some statistics
    echo -e "${YELLOW}📈 Restored database statistics:${NC}"
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
    
    # Scale backend deployment back up
    echo -e "${YELLOW}🔄 Scaling backend deployment back up...${NC}"
    kubectl scale deployment -n ${NAMESPACE} kmm-backend --replicas=2
    
    # Wait for backend pods to be ready
    echo -e "${YELLOW}⏳ Waiting for backend pods to be ready...${NC}"
    kubectl wait --for=condition=ready pod -n ${NAMESPACE} -l component=backend --timeout=120s
    
    echo -e "${GREEN}🎉 Restore completed successfully!${NC}"
    
else
    echo -e "${RED}❌ Restore failed${NC}"
    
    # Scale backend deployment back up even if restore failed
    echo -e "${YELLOW}🔄 Scaling backend deployment back up...${NC}"
    kubectl scale deployment -n ${NAMESPACE} kmm-backend --replicas=2
    
    exit 1
fi

# Cleanup temporary file
if [ -n "${TEMP_FILE}" ] && [ -f "${TEMP_FILE}" ]; then
    rm -f "${TEMP_FILE}"
fi

echo -e "${GREEN}✨ All done!${NC}"