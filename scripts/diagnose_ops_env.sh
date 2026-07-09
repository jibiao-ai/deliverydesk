#!/bin/bash
# ================================================================
# DeliveryDesk - 运维环境数据为0 排查脚本
# 用法: 在部署服务器上运行 bash scripts/diagnose_ops_env.sh
# ================================================================

set -e

echo "=============================================="
echo "  DeliveryDesk OpsEnvironment 诊断脚本"
echo "=============================================="
echo ""

# --- 1. 检查 Docker 容器状态 ---
echo "【1】检查 Docker Compose 服务状态..."
echo "----------------------------------------------"
if command -v docker &>/dev/null; then
    docker compose ps 2>/dev/null || docker-compose ps 2>/dev/null || echo "❌ Docker Compose 服务未启动"
else
    echo "❌ Docker 未安装"
fi
echo ""

# --- 2. 检查后端健康状态 ---
echo "【2】检查后端 API 健康状态..."
echo "----------------------------------------------"
BACKEND_URL="${BACKEND_URL:-http://localhost:80}"
HEALTH_RESP=$(curl -s "${BACKEND_URL}/api/health" 2>/dev/null)
if [ -n "$HEALTH_RESP" ]; then
    echo "✅ 后端响应: $HEALTH_RESP"
else
    echo "❌ 后端无响应，请检查服务是否启动"
    # 尝试直接访问 8080
    HEALTH_RESP=$(curl -s "http://localhost:8080/api/health" 2>/dev/null)
    if [ -n "$HEALTH_RESP" ]; then
        echo "  → 但后端 8080 端口可直接访问: $HEALTH_RESP"
        BACKEND_URL="http://localhost:8080"
    fi
fi
echo ""

# --- 3. 检查 Jira 配置 ---
echo "【3】检查 Jira 配置（环境变量）..."
echo "----------------------------------------------"
echo "  JIRA_SERVER: ${JIRA_SERVER:-'(未设置)'}"
echo "  JIRA_USERNAME: ${JIRA_USERNAME:-'(未设置)'}"
echo "  JIRA_API_TOKEN: ${JIRA_API_TOKEN:+'(已设置, 长度='${#JIRA_API_TOKEN}')'}"
echo "  JIRA_API_TOKEN: ${JIRA_API_TOKEN:-'(未设置) ⚠️ 这是同步失败的最可能原因！'}"
echo ""

# --- 4. 检查容器内的环境变量 ---
echo "【4】检查后端容器内的 Jira 环境变量..."
echo "----------------------------------------------"
if command -v docker &>/dev/null; then
    docker exec deliverydesk-backend env 2>/dev/null | grep -i "JIRA" || echo "  (无法读取容器环境变量或未配置 JIRA 相关变量)"
fi
echo ""

# --- 5. 检查数据库中的 SystemSetting (Jira 配置) ---
echo "【5】检查数据库中的 Jira 系统设置..."
echo "----------------------------------------------"
if command -v docker &>/dev/null; then
    docker exec deliverydesk-mysql mysql -udeliverydesk -pdeliverydesk123 deliverydesk \
        -e "SELECT category, \`key\`, CASE WHEN value_type='password' THEN CONCAT(LEFT(value,4),'***') ELSE value END as value FROM system_settings WHERE category='jira';" \
        2>/dev/null || echo "  (无法连接 MySQL 或表不存在)"
fi
echo ""

# --- 6. 检查 ops_environments 表数据量 ---
echo "【6】检查 ops_environments 表数据..."
echo "----------------------------------------------"
if command -v docker &>/dev/null; then
    docker exec deliverydesk-mysql mysql -udeliverydesk -pdeliverydesk123 deliverydesk \
        -e "SELECT COUNT(*) as total_records FROM ops_environments WHERE deleted_at IS NULL; SELECT status, COUNT(*) as cnt FROM ops_environments WHERE deleted_at IS NULL GROUP BY status;" \
        2>/dev/null || echo "  (无法查询数据库)"
fi
echo ""

# --- 7. 通过 API 直接查看统计 ---
echo "【7】通过 API 获取运维环境统计..."
echo "----------------------------------------------"
# 需要先登录获取 token
echo "  尝试登录获取 token..."
LOGIN_RESP=$(curl -s -X POST "${BACKEND_URL}/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"Admin@2024!","auth_type":"local"}' 2>/dev/null)

TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('token',''))" 2>/dev/null || echo "")

if [ -n "$TOKEN" ] && [ "$TOKEN" != "" ]; then
    echo "  ✅ 登录成功"
    
    # 获取统计
    STATS=$(curl -s "${BACKEND_URL}/api/ops-env/stats" -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
    echo "  OpsEnv Stats: $STATS"
    echo ""
    
    # 获取列表
    LIST=$(curl -s "${BACKEND_URL}/api/ops-env/list?page=1&page_size=5" -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
    TOTAL=$(echo "$LIST" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('total',0))" 2>/dev/null || echo "0")
    echo "  OpsEnv List total: $TOTAL"
    echo ""
    
    # 获取系统设置(Jira配置)
    SETTINGS=$(curl -s "${BACKEND_URL}/api/settings" -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
    echo "  System Settings (Jira): $(echo "$SETTINGS" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = d.get('data', [])
jira_items = [i for i in items if i.get('category') == 'jira']
for i in jira_items:
    val = i.get('value', '')
    if i.get('value_type') == 'password' and len(val) > 4:
        val = val[:4] + '***'
    print(f\"    {i.get('key')}: {val}\")
" 2>/dev/null || echo "(解析失败)")"
    echo ""
    
    # 尝试手动触发同步
    echo "  尝试触发 OpsEnv 同步..."
    SYNC_RESP=$(curl -s -X POST "${BACKEND_URL}/api/ops-env/sync" -H "Authorization: Bearer ${TOKEN}" 2>/dev/null)
    echo "  同步触发响应: $SYNC_RESP"
else
    echo "  ❌ 登录失败: $LOGIN_RESP"
fi
echo ""

# --- 8. 查看后端日志中的 OpsEnv 相关信息 ---
echo "【8】查看后端日志中 OpsEnv 相关信息..."
echo "----------------------------------------------"
if command -v docker &>/dev/null; then
    echo "  (最近 50 行 OpsEnv/Jira 相关日志)"
    docker logs deliverydesk-backend 2>&1 | grep -i "opsenv\|ops.env\|OpsEnvironment\|CSE\|jira" | tail -50 || echo "  (无相关日志)"
fi
echo ""

echo "=============================================="
echo "  排查结论指引"
echo "=============================================="
echo ""
echo "如果看到以下问题，对应解决方案："
echo ""
echo "❌ 问题1: Jira 配置未设置"
echo "   → 解决: 在管理后台 '系统设置' 页面填入 Jira Server/Username/API Token"
echo "   → 或设置环境变量: JIRA_SERVER, JIRA_USERNAME, JIRA_API_TOKEN"
echo ""
echo "❌ 问题2: ops_environments 表记录为 0"
echo "   → 解决: 在 '运维环境' 页面点击 '同步' 按钮手动触发"
echo "   → 或 curl -X POST /api/ops-env/sync (需要 Bearer token)"
echo ""
echo "❌ 问题3: Jira API 返回 401/403"
echo "   → 解决: Jira API Token 无效或过期，请到 Atlassian 重新生成"
echo "   → https://id.atlassian.com/manage/api-tokens"
echo ""
echo "❌ 问题4: 同步日志显示 'no CSE key' 大量跳过"
echo "   → 这是正常的，部分环境详细信息 issue 未关联 CSE 父级 issue"
echo ""
echo "⚠️  注意: 运维环境数据不像 Jira Issue Cache 那样自动定时同步！"
echo "   每次需要更新数据时，需手动在页面点击同步按钮。"
echo "   建议: 可以考虑在 main.go 中增加自动定时同步。"
echo ""
