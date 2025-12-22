#!/bin/bash
# Garage S3 Storage Setup Script for PriceFeed
# Run this on the VPS after docker-compose up

set -e

CONTAINER_NAME="pricefeed-garage"
BUCKET_NAME="receipts"
KEY_NAME="pricefeed-key"

echo "=== PriceFeed Garage S3 Setup ==="
echo ""

# Check if container is running
if ! podman ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
    echo "Error: Container ${CONTAINER_NAME} is not running."
    echo "Run 'podman compose up -d' first."
    exit 1
fi

echo "1. Getting node status..."
NODE_ID=$(podman exec ${CONTAINER_NAME} /garage status 2>/dev/null | grep -oE '[a-f0-9]{16}' | head -1)

if [ -z "$NODE_ID" ]; then
    echo "Error: Could not get node ID. Garage may not be ready yet."
    echo "Wait a few seconds and try again."
    exit 1
fi

echo "   Node ID: ${NODE_ID}"
echo ""

echo "2. Assigning node to cluster layout..."
podman exec ${CONTAINER_NAME} /garage layout assign -z dc1 -c 1G ${NODE_ID} 2>/dev/null || true
echo "   Done."
echo ""

echo "3. Applying layout..."
# Get the current layout version and increment
CURRENT_VERSION=$(podman exec ${CONTAINER_NAME} /garage layout show 2>/dev/null | grep -oE 'version [0-9]+' | grep -oE '[0-9]+' | tail -1 || echo "0")
NEXT_VERSION=$((CURRENT_VERSION + 1))
podman exec ${CONTAINER_NAME} /garage layout apply --version ${NEXT_VERSION} 2>/dev/null || true
echo "   Applied layout version ${NEXT_VERSION}."
echo ""

echo "4. Creating API key..."
# Check if key already exists
EXISTING_KEY=$(podman exec ${CONTAINER_NAME} /garage key list 2>/dev/null | grep "${KEY_NAME}" || true)
if [ -n "$EXISTING_KEY" ]; then
    echo "   Key '${KEY_NAME}' already exists. Getting info..."
    KEY_OUTPUT=$(podman exec ${CONTAINER_NAME} /garage key info ${KEY_NAME} 2>/dev/null)
else
    KEY_OUTPUT=$(podman exec ${CONTAINER_NAME} /garage key create ${KEY_NAME} 2>/dev/null)
fi

ACCESS_KEY=$(echo "$KEY_OUTPUT" | grep -oE 'GK[a-zA-Z0-9]+' | head -1)
SECRET_KEY=$(echo "$KEY_OUTPUT" | grep -oE '[a-f0-9]{64}' | head -1)

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
    echo "   Warning: Could not parse keys. Here's the raw output:"
    echo "$KEY_OUTPUT"
    echo ""
    echo "   Please manually extract the Access Key ID and Secret Access Key."
else
    echo "   Access Key: ${ACCESS_KEY}"
    echo "   Secret Key: ${SECRET_KEY}"
fi
echo ""

echo "5. Creating bucket '${BUCKET_NAME}'..."
podman exec ${CONTAINER_NAME} /garage bucket create ${BUCKET_NAME} 2>/dev/null || echo "   Bucket may already exist."
echo ""

echo "6. Granting key access to bucket..."
podman exec ${CONTAINER_NAME} /garage bucket allow ${BUCKET_NAME} --read --write --key ${KEY_NAME} 2>/dev/null || true
echo "   Done."
echo ""

echo "=== Setup Complete ==="
echo ""
echo "Add these to your .env file:"
echo ""
echo "S3_ENDPOINT=garage:3900"
echo "S3_ACCESS_KEY=${ACCESS_KEY}"
echo "S3_SECRET_KEY=${SECRET_KEY}"
echo "S3_BUCKET=${BUCKET_NAME}"
echo "S3_USE_SSL=false"
echo "S3_REGION=garage"
echo ""
echo "Then restart the app container:"
echo "  podman compose restart app"
echo ""
