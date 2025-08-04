#!/bin/bash

# Script to fix permissions for shared-data directories
# Based on the required permissions for Docker services

set -e

SHARED_DATA_DIR="./shared-data"

echo "🔧 Fixing permissions for shared-data directories..."

# Check if shared-data directory exists
if [ ! -d "$SHARED_DATA_DIR" ]; then
    echo "❌ Directory $SHARED_DATA_DIR does not exist!"
    exit 1
fi

# Stop all services first
echo "🛑 Stopping Docker services..."
docker compose down

# Create directories if they don't exist
echo "📁 Creating directories..."
mkdir -p "$SHARED_DATA_DIR"
mkdir -p "$SHARED_DATA_DIR/grafana"
mkdir -p "$SHARED_DATA_DIR/pgadmin"
mkdir -p "$SHARED_DATA_DIR/postgresql"
mkdir -p "$SHARED_DATA_DIR/prometheus"
mkdir -p "$SHARED_DATA_DIR/rabbitmq"
mkdir -p "$SHARED_DATA_DIR/redis"

# Check and create nobody group if needed (Ubuntu fix)
if ! getent group nobody &>/dev/null; then
    echo "🔧 Creating nobody group for Ubuntu compatibility..."
    sudo groupadd nobody
    sudo usermod -aG nobody nobody 2>/dev/null || true
fi

# Set permissions for each service
echo "🔐 Setting permissions..."

# Grafana (UID:472, GID:472, permissions:755)
echo "  - Setting grafana permissions (472:472, 755)..."
sudo chown -R 472:472 "$SHARED_DATA_DIR/grafana"
sudo chmod 755 "$SHARED_DATA_DIR/grafana"

# PgAdmin (UID:5050, GID:5050, permissions:755)
echo "  - Setting pgadmin permissions (5050:5050, 755)..."
sudo chown -R 5050:5050 "$SHARED_DATA_DIR/pgadmin"
sudo chmod 755 "$SHARED_DATA_DIR/pgadmin"

# PostgreSQL (UID:70, GID:root, permissions:700)
echo "  - Setting postgresql permissions (70:root, 700)..."
sudo chown -R 70:root "$SHARED_DATA_DIR/postgresql"
sudo chmod 700 "$SHARED_DATA_DIR/postgresql"

# Prometheus (UID:nobody, GID:nobody, permissions:775)
echo "  - Setting prometheus permissions (nobody:nobody, 775)..."
sudo chown -R nobody:nobody "$SHARED_DATA_DIR/prometheus"
sudo chmod 775 "$SHARED_DATA_DIR/prometheus"

# RabbitMQ (UID:systemd-coredump, GID:root, permissions:755)
echo "  - Setting rabbitmq permissions (systemd-coredump:root, 755)..."
sudo chown -R systemd-coredump:root "$SHARED_DATA_DIR/rabbitmq"
sudo chmod 755 "$SHARED_DATA_DIR/rabbitmq"

# Redis (UID:systemd-coredump, GID:root, permissions:755)
echo "  - Setting redis permissions (systemd-coredump:root, 755)..."
sudo chown -R systemd-coredump:root "$SHARED_DATA_DIR/redis"
sudo chmod 755 "$SHARED_DATA_DIR/redis"

echo "✅ Permissions fixed successfully!"

# Display current permissions
echo "📋 Current permissions:"
ls -la "$SHARED_DATA_DIR/"

# Start services
echo "🚀 Starting Docker services..."
docker compose up -d

echo "🎉 Done! All services should now have correct permissions."

# Check service status
echo "📊 Service status:"
docker compose ps

