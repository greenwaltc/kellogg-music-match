#!/bin/bash
# Schema migration helper for Kellogg Music Match
# Usage: ./create-migration.sh "migration_name"

set -e

SCHEMA_DIR="backend/db/schema"
MIGRATION_NAME="$1"

if [ -z "$MIGRATION_NAME" ]; then
    echo "Usage: $0 'migration_name'"
    echo "Example: $0 'add_user_roles'"
    exit 1
fi

# Find the next migration number
LAST_NUM=$(ls "$SCHEMA_DIR"/*.sql 2>/dev/null | grep -o '[0-9]\+' | sort -n | tail -1)
if [ -z "$LAST_NUM" ]; then
    NEXT_NUM="001"
else
    NEXT_NUM=$(printf "%03d" $((LAST_NUM + 1)))
fi

# Create filename
FILENAME="${NEXT_NUM}_${MIGRATION_NAME}.sql"
FILEPATH="$SCHEMA_DIR/$FILENAME"

# Create migration file template
cat > "$FILEPATH" << EOF
-- Migration: ${MIGRATION_NAME}
-- Date: $(date +%Y-%m-%d)
-- Description: TODO: Describe what this migration does

-- TODO: Add your SQL statements here
-- Example:
-- ALTER TABLE users ADD COLUMN new_field VARCHAR(255);
-- CREATE INDEX idx_users_new_field ON users(new_field);

EOF

echo "✅ Created migration file: $FILEPATH"
echo "📝 Edit the file to add your SQL statements"
echo "🔄 Run 'make sync-schema' after editing to update DATABASE_SCHEMA.sql"
echo "🏗️ Run 'make backend-generate-sqlc' to update Go models"