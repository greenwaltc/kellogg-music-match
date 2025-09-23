#!/bin/bash

# Setup automatic daily backups for Kellogg Music Match database
# This script configures a cron job to run backups daily at 2 AM

BACKUP_SCRIPT="/home/cameron/projects/kellogg-music-match/scripts/backup-database.sh"
LOG_FILE="/home/cameron/backups/kellogg-music-match/backup.log"

echo "🔧 Setting up automatic database backups..."

# Create backup directory if it doesn't exist
mkdir -p /home/cameron/backups/kellogg-music-match

# Create log file
touch $LOG_FILE

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "backup-database.sh"; then
    echo "⚠️  Backup cron job already exists!"
    echo "Current cron jobs:"
    crontab -l 2>/dev/null | grep -E "(backup-database|#.*backup)"
    exit 0
fi

# Create the cron job
(crontab -l 2>/dev/null; echo "# Kellogg Music Match Database Backup - Daily at 2 AM"; echo "0 2 * * * $BACKUP_SCRIPT >> $LOG_FILE 2>&1") | crontab -

if [ $? -eq 0 ]; then
    echo "✅ Automatic backup configured successfully!"
    echo "📅 Schedule: Daily at 2:00 AM"
    echo "📍 Script: $BACKUP_SCRIPT"
    echo "📝 Logs: $LOG_FILE"
    echo ""
    echo "Current cron jobs:"
    crontab -l
else
    echo "❌ Failed to configure automatic backup"
    exit 1
fi

echo ""
echo "🎯 To disable automatic backups later:"
echo "   crontab -e"
echo "   (remove the backup-database.sh line)"