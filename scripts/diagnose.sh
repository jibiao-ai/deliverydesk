#!/bin/bash
# DeliveryDesk - Diagnostic script
# Usage: bash scripts/diagnose.sh
# Run this when backend is crash-looping or login fails

set -e

echo "============================================"
echo "  DeliveryDesk Diagnostic Tool"
echo "============================================"
echo ""

# 1. Check container status
echo "[1/6] Container Status"
echo "----------------------------------------------"
docker compose ps 2>/dev/null || docker-compose ps 2>/dev/null
echo ""

# 2. Check backend logs (last 50 lines)
echo "[2/6] Backend Logs (last 50 lines)"
echo "----------------------------------------------"
docker compose logs --tail=50 backend 2>/dev/null || docker-compose logs --tail=50 backend 2>/dev/null
echo ""

# 3. Check MySQL connectivity
echo "[3/6] MySQL Connectivity Test"
echo "----------------------------------------------"
docker compose exec -T mysql mysqladmin ping -h localhost -u root -proot123 2>/dev/null && echo "  [OK] MySQL is responding" || echo "  [FAIL] MySQL is not responding"
echo ""

# Test database and user exist
echo "  Checking database 'deliverydesk'..."
docker compose exec -T mysql mysql -u root -proot123 -e "SHOW DATABASES LIKE 'deliverydesk';" 2>/dev/null | grep -q deliverydesk && echo "  [OK] Database 'deliverydesk' exists" || echo "  [FAIL] Database 'deliverydesk' NOT found"

echo "  Checking user 'deliverydesk'..."
docker compose exec -T mysql mysql -u root -proot123 -e "SELECT User, Host FROM mysql.user WHERE User='deliverydesk';" 2>/dev/null | grep -q deliverydesk && echo "  [OK] User 'deliverydesk' exists" || echo "  [FAIL] User 'deliverydesk' NOT found"

echo "  Testing application user login..."
docker compose exec -T mysql mysql -u deliverydesk -pdeliverydesk123 -e "SELECT 1;" deliverydesk 2>/dev/null && echo "  [OK] App user can connect to database" || echo "  [FAIL] App user CANNOT connect - this is the likely crash cause!"
echo ""

# 4. Check admin user in database
echo "[4/6] Admin User Check"
echo "----------------------------------------------"
docker compose exec -T mysql mysql -u root -proot123 deliverydesk -e "SELECT id, username, LENGTH(password) as pwd_len, role, auth_type FROM users LIMIT 5;" 2>/dev/null || echo "  [INFO] Cannot query users table (may not exist yet)"
echo ""

# 5. Check RabbitMQ
echo "[5/6] RabbitMQ Status"
echo "----------------------------------------------"
docker compose exec -T rabbitmq rabbitmqctl status 2>/dev/null | head -5 && echo "  [OK] RabbitMQ is running" || echo "  [WARN] RabbitMQ check failed (non-critical)"
echo ""

# 6. Check network and ports
echo "[6/6] Port Binding Check"
echo "----------------------------------------------"
echo "  Port 80 (frontend):"
ss -tlnp 2>/dev/null | grep ":80 " || netstat -tlnp 2>/dev/null | grep ":80 " || echo "    Not bound"
echo "  Port 8080 (backend):"
ss -tlnp 2>/dev/null | grep ":8080 " || netstat -tlnp 2>/dev/null | grep ":8080 " || echo "    Not bound (backend may be crashed)"
echo ""

# Quick fix suggestions
echo "============================================"
echo "  Common Fixes"
echo "============================================"
echo ""
echo "  Backend crash-looping (Restarting status):"
echo "    1. Check logs: docker compose logs backend"
echo "    2. If 'failed to connect to database':"
echo "       docker compose down -v   # Reset volumes"
echo "       docker compose up -d     # Recreate everything"
echo "    3. If MySQL user issue:"
echo "       docker compose exec mysql mysql -u root -proot123 -e \\"
echo "         \"CREATE USER IF NOT EXISTS 'deliverydesk'@'%' IDENTIFIED BY 'deliverydesk123';\""
echo "       docker compose exec mysql mysql -u root -proot123 -e \\"
echo "         \"GRANT ALL ON deliverydesk.* TO 'deliverydesk'@'%'; FLUSH PRIVILEGES;\""
echo "       docker compose restart backend"
echo ""
echo "  Login fails with correct password:"
echo "    curl http://localhost:8080/api/health"
echo "    # Should show: admin_exists=true, admin_hash_len=60"
echo ""
echo "============================================"
