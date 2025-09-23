#!/bin/bash

# Script to set PostgreSQL persistent volume reclaim policy to "Retain"
# This prevents data loss if the PVC is accidentally deleted

set -e

echo "🔒 Setting PostgreSQL persistent volume reclaim policy to 'Retain'..."

# Get the PV name for the postgres PVC
PV_NAME=$(kubectl get pvc postgres-storage-postgres-0 -n kmm -o jsonpath='{.spec.volumeName}' 2>/dev/null || echo "")

if [ -z "$PV_NAME" ]; then
    echo "❌ Error: PostgreSQL PVC not found. Make sure your Pulumi deployment is running."
    exit 1
fi

echo "📦 Found persistent volume: $PV_NAME"

# Check current reclaim policy
CURRENT_POLICY=$(kubectl get pv "$PV_NAME" -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')
echo "📋 Current reclaim policy: $CURRENT_POLICY"

if [ "$CURRENT_POLICY" = "Retain" ]; then
    echo "✅ Reclaim policy is already set to 'Retain'. No changes needed."
    exit 0
fi

# Patch the PV to use Retain policy
echo "🔧 Updating reclaim policy from '$CURRENT_POLICY' to 'Retain'..."
kubectl patch pv "$PV_NAME" -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'

# Verify the change
NEW_POLICY=$(kubectl get pv "$PV_NAME" -o jsonpath='{.spec.persistentVolumeReclaimPolicy}')
echo "✅ Successfully updated reclaim policy to: $NEW_POLICY"

echo ""
echo "🎉 Your PostgreSQL data is now protected!"
echo "📍 Data location: /var/lib/rancher/k3s/storage/${PV_NAME}*"
echo ""
echo "ℹ️  What this means:"
echo "   - Your database will persist across node restarts"
echo "   - Data will be retained even if the PVC is deleted"
echo "   - Manual cleanup required if you want to delete the data"