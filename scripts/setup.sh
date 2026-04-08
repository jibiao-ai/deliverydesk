#!/bin/bash
# DeliveryDesk - Quick setup script
# Usage: bash scripts/setup.sh
#
# Environment variables (optional):
#   EXTERNAL_IP  - External/floating IP for access (e.g. 192.168.3.112)
#   SKIP_BUILD   - Set to 1 to skip docker build step

set -e

echo "============================================"
echo "  DeliveryDesk - Cloud Delivery Workbench"
echo "============================================"
echo ""

# Check docker
if ! command -v docker &> /dev/null; then
    echo "[ERROR] Docker is not installed. Please install Docker first."
    echo "  CentOS: yum install -y docker-ce docker-ce-cli containerd.io"
    echo "  Ubuntu: apt install -y docker.io docker-compose-plugin"
    exit 1
fi

if ! command -v docker compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "[ERROR] Docker Compose (v2) is not installed."
    echo "  Install: apt install docker-compose-plugin  OR  yum install docker-compose-plugin"
    exit 1
fi

# Copy env if needed
if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "[OK] Created .env from .env.example"
        echo "     Please edit .env to configure your AI API Key"
    else
        echo "[WARN] .env.example not found, skipping .env creation"
    fi
fi

# Build and start
echo ""
if [ "${SKIP_BUILD}" != "1" ]; then
    echo "[INFO] Building Docker images..."
    docker compose build
fi

echo "[INFO] Starting services..."
docker compose up -d

echo ""
echo "[INFO] Waiting for services to start..."
sleep 15

# Check service status
echo ""
echo "============================================"
echo "  Service Status"
echo "============================================"
docker compose ps
echo ""

# Get IP addresses
INTERNAL_IP=$(hostname -I | awk '{print $1}')
echo "============================================"
echo "  Access URLs"
echo "============================================"
echo "  Internal IP:      ${INTERNAL_IP}"
echo "  Web UI:           http://${INTERNAL_IP}"
echo "  Backend API:      http://${INTERNAL_IP}:8080/api"
echo "  RabbitMQ Admin:   http://${INTERNAL_IP}:15672"
if [ -n "${EXTERNAL_IP}" ]; then
    echo ""
    echo "  External/Float IP: ${EXTERNAL_IP}"
    echo "  Web UI (ext):     http://${EXTERNAL_IP}"
    echo "  Backend API (ext):http://${EXTERNAL_IP}:8080/api"
fi
echo ""
echo "  Default Login:    admin / Admin@2024!"
echo "============================================"
echo ""

# Open firewall ports
echo "[INFO] Configuring firewall..."
if command -v firewall-cmd &> /dev/null; then
    echo "  Detected firewalld"
    sudo firewall-cmd --permanent --add-port=80/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=8080/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=15672/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=3306/tcp 2>/dev/null || true
    sudo firewall-cmd --reload 2>/dev/null || true
    echo "  [OK] Firewall ports opened (80, 8080, 15672, 3306)"
elif command -v ufw &> /dev/null; then
    echo "  Detected ufw"
    sudo ufw allow 80/tcp 2>/dev/null || true
    sudo ufw allow 8080/tcp 2>/dev/null || true
    sudo ufw allow 15672/tcp 2>/dev/null || true
    sudo ufw allow 3306/tcp 2>/dev/null || true
    echo "  [OK] Firewall ports opened (80, 8080, 15672, 3306)"
elif command -v iptables &> /dev/null; then
    echo "  Detected iptables (adding rules)"
    sudo iptables -I INPUT -p tcp --dport 80 -j ACCEPT 2>/dev/null || true
    sudo iptables -I INPUT -p tcp --dport 8080 -j ACCEPT 2>/dev/null || true
    sudo iptables -I INPUT -p tcp --dport 15672 -j ACCEPT 2>/dev/null || true
    echo "  [OK] iptables rules added (80, 8080, 15672)"
else
    echo "  [WARN] No firewall manager found. Make sure ports 80, 8080 are open."
fi

# Verify services are actually running
echo ""
echo "[INFO] Verifying services..."
BACKEND_UP=false
for i in $(seq 1 10); do
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/login 2>/dev/null | grep -q "405\|200\|400"; then
        BACKEND_UP=true
        break
    fi
    sleep 2
done

if [ "$BACKEND_UP" = true ]; then
    echo "  [OK] Backend is responding on port 8080"
else
    echo "  [WARN] Backend may not be ready yet. Check: docker compose logs backend"
fi

FRONTEND_UP=false
if curl -s -o /dev/null -w "%{http_code}" http://localhost:80/ 2>/dev/null | grep -q "200"; then
    FRONTEND_UP=true
    echo "  [OK] Frontend is responding on port 80"
else
    echo "  [WARN] Frontend may not be ready yet. Check: docker compose logs frontend"
fi

echo ""
echo "============================================"
echo "  Troubleshooting"
echo "============================================"
echo "  View all logs:        docker compose logs -f"
echo "  View backend logs:    docker compose logs -f backend"
echo "  Restart all:          docker compose restart"
echo "  Rebuild & restart:    docker compose down && docker compose build --no-cache && docker compose up -d"
echo "  Reset database:       docker compose down -v && docker compose up -d"
echo ""
echo "  If external IP (floating IP) cannot access:"
echo "  1. Check VM security group / network ACL allows ports 80, 8080"
echo "  2. Check floating IP is correctly bound: ip addr show"
echo "  3. Check iptables: sudo iptables -L -n | grep 80"
echo "  4. Test connectivity: curl http://<external-ip>"
echo "============================================"
echo ""
echo "[OK] DeliveryDesk is ready!"
