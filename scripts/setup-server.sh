#!/bin/bash
# Production Server Setup Script for PriceFeed
# Enables unprivileged port binding and configures firewall
# Run with: sudo ./scripts/setup-server.sh

set -e

echo "=== PriceFeed Server Setup ==="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

echo "1. Enabling unprivileged port binding (80/443)..."
sysctl -w net.ipv4.ip_unprivileged_port_start=80

# Make persistent
echo "net.ipv4.ip_unprivileged_port_start=80" > /etc/sysctl.d/99-unprivileged-ports.conf
echo "   Done. Setting persisted to /etc/sysctl.d/99-unprivileged-ports.conf"
echo ""

echo "2. Configuring firewall..."
if command -v firewall-cmd &> /dev/null; then
    # Rocky/RHEL/Fedora
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --permanent --add-port=8181/tcp
    firewall-cmd --reload
    echo "   Firewalld configured."
elif command -v ufw &> /dev/null; then
    # Ubuntu/Debian
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 8080/tcp
    ufw allow 8181/tcp
    echo "   UFW configured."
else
    echo "   Warning: No supported firewall found (firewalld or ufw)"
    echo "   Please manually open ports: 80, 443, 8080, 8181"
fi
echo ""

echo "3. Verifying settings..."
CURRENT=$(cat /proc/sys/net/ipv4/ip_unprivileged_port_start)
if [ "$CURRENT" -eq 80 ]; then
    echo "   Unprivileged port start: $CURRENT (OK)"
else
    echo "   Warning: Unprivileged port start is $CURRENT, expected 80"
fi
echo ""

echo "=== Setup Complete ==="
echo ""
echo "You can now run the application:"
echo "  cd /home/foxx-price/price-feedv1"
echo "  podman-compose up -d"
echo ""
echo "Access points:"
echo "  - App (direct):    http://YOUR_IP:8080"
echo "  - NPM Admin:       http://YOUR_IP:8181"
echo "  - HTTP (via NPM):  http://YOUR_DOMAIN"
echo "  - HTTPS (via NPM): https://YOUR_DOMAIN"
echo ""
