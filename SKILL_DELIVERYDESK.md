# DeliveryDesk 全栈开发 Skill

> 本文档是 DeliveryDesk（云交付工作台）项目的完整开发技能总结，可用于将此产品的前端/后端开发模式平移到其他项目。

---

## 一、技术栈总览

| 层 | 技术 | 版本 |
|---|---|---|
| **前端框架** | React (SPA) | 18.3 |
| **构建工具** | Vite | 5.x |
| **CSS 方案** | Tailwind CSS + CSS Variables | 3.4 |
| **状态管理** | Zustand | 4.5 |
| **路由** | React Router DOM | 6.26 |
| **HTTP 客户端** | Axios | 1.7 |
| **图表库** | Recharts | 3.9 |
| **图标库** | Lucide React | 0.424 |
| **通知** | React Hot Toast | 2.4 |
| **Markdown** | React Markdown + Highlight.js | 9.0 / 11.10 |
| **流程图** | @xyflow/react (React Flow) | 12.11 |
| **后端框架** | Go + Gin | 1.22 / 1.10 |
| **ORM** | GORM (MySQL + SQLite) | 1.30 |
| **消息队列** | RabbitMQ (amqp091-go) | 3.13 |
| **Excel 解析** | excelize/v2 | 2.7 |
| **日志** | Logrus (JSON 格式) | 1.9 |
| **认证** | JWT (自实现) + bcrypt | — |
| **LDAP** | go-ldap/ldap/v3 | 3.4 |
| **容器化** | Docker + Docker Compose | — |
| **反向代理** | Nginx (SSE + WebSocket 支持) | Alpine |

---

## 二、项目目录结构

```
project-root/
├── docker-compose.yml              # 编排: MySQL + RabbitMQ + Backend + Frontend
├── backend/
│   ├── Dockerfile                   # Go multi-stage build (golang:1.22-alpine → alpine:3.19)
│   ├── go.mod / go.sum
│   ├── cmd/server/main.go           # 入口: 初始化DB/MQ/Services, 注册路由, 启动Gin
│   ├── internal/
│   │   ├── config/config.go         # 环境变量配置加载 (全 env-based, 无 config 文件)
│   │   ├── model/models.go          # GORM 模型定义 (所有表在一个文件)
│   │   ├── repository/db.go         # DB初始化 + AutoMigrate + 种子数据
│   │   ├── handler/                 # HTTP Handler (按功能模块拆分)
│   │   │   ├── handlers.go          # 主handler: Auth/Dashboard/Agents/Skills/Users
│   │   │   ├── biz_handlers.go      # 商机管理 handler
│   │   │   ├── wbs_handlers.go      # WBS服务 handler
│   │   │   ├── workflow_handlers.go # 工作流 handler
│   │   │   ├── worktime_handlers.go # 工时管理 handler
│   │   │   ├── project_handlers.go  # 项目管理 handler
│   │   │   ├── ops_env_handlers.go  # 运维环境 handler
│   │   │   ├── kb_handlers.go       # 知识库 handler
│   │   │   └── totp_handlers.go     # 双因子认证 handler
│   │   ├── middleware/auth.go       # JWT认证 + Admin权限中间件
│   │   ├── service/                 # 业务逻辑层
│   │   └── mq/rabbitmq.go          # RabbitMQ 连接和消费
│   └── pkg/
│       ├── logger/logger.go         # Logrus JSON logger
│       └── response/response.go     # 统一API响应格式
├── frontend/
│   ├── Dockerfile                   # Node multi-stage build → nginx:alpine
│   ├── nginx.conf                   # 反向代理 /api/ → backend:8080 + SSE支持
│   ├── package.json
│   ├── vite.config.js               # dev proxy /api → localhost:8080
│   ├── tailwind.config.js           # 自定义 primary 色 + darkMode selector
│   ├── postcss.config.js
│   ├── src/
│   │   ├── main.jsx                 # ReactDOM.createRoot 入口
│   │   ├── App.jsx                  # Routes: /login + /* → PrivateRoute → MainLayout
│   │   ├── store/useStore.js        # Zustand store: auth/sidebar/theme/agents/conversations
│   │   ├── services/api.js          # Axios 实例 + 所有API函数 (拦截器自动加token/处理401)
│   │   ├── styles/index.css         # Tailwind base + CSS Variables + 完整 dark mode 覆盖
│   │   ├── components/
│   │   │   ├── MainLayout.jsx       # 页面路由表 + 顶栏(主题切换/通知铃) + 页面渲染
│   │   │   ├── Sidebar.jsx          # 侧边栏菜单 (用户菜单 + 管理员菜单分组)
│   │   │   └── FullscreenButton.jsx # 全屏按钮组件
│   │   ├── pages/                   # 每个功能一个页面组件
│   │   │   ├── DashboardPage.jsx
│   │   │   ├── ChatPage.jsx
│   │   │   ├── AgentsPage.jsx
│   │   │   ├── BizOpportunityPage.jsx   # 商机管理 (Excel上传/图表/表格/筛选)
│   │   │   ├── WorkflowPage.jsx         # 可视化工作流 (React Flow)
│   │   │   ├── WBSServicePage.jsx       # WBS报价 (GlassSelect组件模式)
│   │   │   └── ... 其他页面
│   │   └── data/
│   │       └── motivationalQuotes.js    # 励志语录数据
```

---

## 三、核心架构模式

### 3.1 后端：新增功能模块 Checklist

添加一个新业务模块（如"商机管理"）需要修改以下文件：

#### Step 1: 定义数据模型 (`model/models.go`)
```go
type BizOpportunity struct {
    ID        uint           `gorm:"primarykey" json:"id"`
    CreatedAt time.Time      `json:"created_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`     // GORM 软删除
    Month     string         `gorm:"size:16;index;not null" json:"month"`
    Name      string         `gorm:"size:512;index" json:"name"`
    Amount    float64        `json:"amount"`
    // ... 其他字段
}
```

**要点：**
- 所有模型放在一个文件 `models.go`
- 主键用 `uint` + `gorm:"primarykey"`
- 软删除用 `gorm.DeletedAt` + `gorm:"index"`
- JSON tag 用 snake_case
- 字符串字段指定 `gorm:"size:N"`，索引字段加 `index`

#### Step 2: 注册迁移 (`repository/db.go`)
```go
// 在 migrationModels map 中添加:
"BizOpportunity": &model.BizOpportunity{},

// 在 migrationOrder slice 中添加:
"BizOpportunity",
```

**要点：**
- 逐表迁移（不是 `db.AutoMigrate(model1, model2, ...)`），便于诊断
- 遇到 Error 1071 (key too long) 自动 drop+recreate
- 在数据库和表层面强制 `utf8mb4_general_ci` collation

#### Step 3: 创建 Handler (`handler/xxx_handlers.go`)
```go
type BizHandler struct{}

func NewBizHandler() *BizHandler {
    return &BizHandler{}
}

func (h *BizHandler) ListBizOpportunities(c *gin.Context) {
    // 从 query 参数获取过滤条件
    month := c.Query("month")
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    
    query := repository.DB.Model(&model.BizOpportunity{})
    if month != "" {
        query = query.Where("month = ?", month)
    }
    
    var total int64
    query.Count(&total)
    
    var records []model.BizOpportunity
    query.Order("amount DESC").Offset((page-1)*50).Limit(50).Find(&records)
    
    c.JSON(http.StatusOK, gin.H{
        "code":  0,
        "data":  records,
        "total": total,
    })
}
```

**API 响应格式约定：**
```json
// 成功
{ "code": 0, "data": ..., "message": "success" }

// 列表（带分页）
{ "code": 0, "data": [...], "total": 120, "page": 1 }

// 失败
{ "code": -1, "message": "错误信息" }
```

也可以使用 `pkg/response` 包：
```go
response.Success(c, data)
response.BadRequest(c, "参数错误")
response.Unauthorized(c, "未授权")
```

#### Step 4: 注册路由 (`cmd/server/main.go`)
```go
// 初始化 handler
bizH := handler.NewBizHandler()

// 根据权限级别选择路由组:
// auth 组 — 所有登录用户可访问
auth.GET("/biz/list", bizH.ListBizOpportunities)

// admin 组 — 仅管理员可访问
admin.POST("/biz/upload", bizH.UploadBizExcel)
admin.DELETE("/biz/history/:id", bizH.DeleteUpload)
```

**路由组层级：**
```
/api
├── /login, /health                  # 无需认证
├── /published-agents/*              # 公开API
├── auth = /api + AuthMiddleware     # 需登录
│   ├── /profile, /dashboard
│   ├── /agents, /conversations, /skills, ...
│   └── admin = auth + AdminMiddleware  # 需管理员
│       ├── /users, /ldap-configs
│       └── /biz/*, /workflows/*
```

#### Step 5: Excel 解析模式 (excelize/v2)
```go
// 上传文件
file, header, err := c.Request.FormFile("file")
f, err := excelize.OpenReader(file)
rows, err := f.GetRows(sheetName)

// 列头模糊匹配（按长度降序，避免子串冲突）
colNames := []string{"商机名称", "负责人所属核心管控单元", "负责人", ...}
sort.Slice(colNames, func(i, j int) bool {
    return len([]rune(colNames[i])) > len([]rune(colNames[j]))
})
for i, cell := range headerRow {
    for _, cn := range colNames {
        if strings.Contains(cell, cn) {
            colIdx[cn] = i
            break
        }
    }
}

// 逐行解析
for _, row := range rows[1:] {
    val := getCell(row, "商机名称")
    // ...
}

// 批量插入
repository.DB.CreateInBatches(&records, 100)
```

⚠️ **陷阱**：列头模糊匹配时，长名必须排在前面，否则"负责人"会匹配到"负责人所属核心管控单元"。

#### Step 6: 导出 Excel
```go
f := excelize.NewFile()
sheet := "数据"
f.SetSheetName("Sheet1", sheet)

// 设置表头样式
headerStyle, _ := f.NewStyle(&excelize.Style{
    Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
    Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"6C5CE7"}},
})

// 写入数据
for i, r := range records {
    f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), r.Name)
}

// 响应
c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
f.Write(c.Writer)
```

---

### 3.2 前端：新增页面 Checklist

#### Step 1: 添加 API 函数 (`services/api.js`)
```js
// 所有 API 函数集中在一个文件
export const getBizList = (params) => api.get('/biz/list', { params });
export const getBizStats = (params) => api.get('/biz/stats', { params });
export const uploadBizExcel = (file, month) => {
  const fd = new FormData();
  fd.append('file', file);
  if (month) fd.append('month', month);
  return api.post('/biz/upload', fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};
export const deleteBizUpload = (id) => api.delete(`/biz/history/${id}`);
```

**Axios 全局配置要点：**
- baseURL = `/api`（Vite dev proxy 或 nginx 反向代理）
- 拦截器自动附加 `Authorization: Bearer <token>`
- 401 自动清除 token 并跳转 `/login`
- `response.data` 自动解包

#### Step 2: 创建页面组件 (`pages/BizOpportunityPage.jsx`)
```jsx
import React, { useState, useEffect, useCallback, useRef } from 'react';
import { Upload, Search, Filter, ChevronDown, Check, ... } from 'lucide-react';
import { BarChart, Bar, PieChart, Pie, ... } from 'recharts';
import { getBizList, getBizStats, ... } from '../services/api';
import useStore from '../store/useStore';

export default function BizOpportunityPage() {
  const theme = useStore((s) => s.theme);
  const isDark = theme === 'dark';
  
  // 通用样式变量
  const cardClass = `rounded-xl border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-200'}`;
  const textMain = isDark ? 'text-slate-200' : 'text-gray-800';
  const textSub = isDark ? 'text-slate-400' : 'text-gray-500';
  const textMuted = isDark ? 'text-slate-500' : 'text-gray-400';
  
  // ... 页面逻辑
}
```

#### Step 3: 注册到侧边栏 (`components/Sidebar.jsx`)
```jsx
// 1. 添加图标 import
import { TrendingUp } from 'lucide-react';

// 2. 在对应菜单组的 items 中添加
{ id: 'biz-opportunity', label: '续保商机', icon: TrendingUp },
```

**菜单组结构：**
- `userMenuGroups` — 所有用户可见（仪表盘、公司系统、即时对话等）
- `adminMenuGroups` — 仅管理员可见
  - 智能应用组（智能体、技能、工作流、模型配置）
  - 业务应用组（工时、项目、运维、商机）
  - 系统管理组（LDAP、用户、设置、日志）

#### Step 4: 注册到主布局 (`components/MainLayout.jsx`)
```jsx
// 1. Import 页面组件
import BizOpportunityPage from '../pages/BizOpportunityPage';

// 2. 注册到 pageComponents
const pageComponents = {
  // ...
  'biz-opportunity': BizOpportunityPage,
};

// 3. 注册到 PAGE_META（标题和副标题）
const PAGE_META = {
  // ...
  'biz-opportunity': { title: '续保商机', subtitle: '维保/续保商机数据分析与TOP10可视化看板' },
};

// 4. 如需管理员权限，添加到 ADMIN_PAGES
const ADMIN_PAGES = new Set([..., 'biz-opportunity']);
```

---

### 3.3 暗色模式 (Dark Mode) 模式

#### 方案：CSS Variables + data-theme 属性 + Zustand + 内联条件类

```jsx
// 1. Zustand store 管理主题
theme: localStorage.getItem('theme') || 'light',
setTheme: (theme) => {
    localStorage.setItem('theme', theme);
    document.documentElement.setAttribute('data-theme', theme);
    set({ theme });
},

// 2. 页面组件获取 isDark
const theme = useStore((s) => s.theme);
const isDark = theme === 'dark';

// 3. 条件类名模式
className={`text-sm ${isDark ? 'text-slate-200' : 'text-gray-800'}`}

// 4. 通用样式变量（每个页面顶部定义）
const cardClass = `rounded-xl border ${isDark ? 'bg-slate-800 border-slate-700' : 'bg-white border-gray-200'}`;
```

#### Tailwind 配置:
```js
// tailwind.config.js
darkMode: ['selector', '[data-theme="dark"]'],
```

#### 全局 CSS 覆盖 (`index.css`):
- `[data-theme="dark"] .bg-white { background-color: #1e293b !important; }`
- 覆盖所有 Tailwind gray 系列颜色、表格、输入框、边框等
- 半透明背景适配：`.bg-white/70` → `rgba(30,41,59,0.85)`
- 渐变背景适配：`.from-slate-50` → `#0f172a`

**暗色模式配色方案：**
| 用途 | Light | Dark |
|------|-------|------|
| 主背景 | `#ffffff` | `#0f172a` (slate-900) |
| 卡片背景 | `#ffffff` | `#1e293b` (slate-800) |
| 悬停背景 | `#f3f4f6` | `#334155` (slate-700) |
| 主文字 | `#111827` | `#f1f5f9` (slate-100) |
| 次要文字 | `#6b7280` | `#94a3b8` (slate-400) |
| 边框 | `#e5e7eb` | `#334155` (slate-700) |

---

### 3.4 自定义下拉框组件 (CustomSelect)

项目**不使用原生 `<select>`**，而是用自定义下拉组件实现一致的 UI：

```jsx
function CustomSelect({ value, onChange, options, placeholder, isDark, className = '' }) {
  const [open, setOpen] = useState(false);
  const ref = useRef(null);
  const selected = options.find(o => o.value === value);

  // 点击外部关闭
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false);
    };
    if (open) document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [open]);

  return (
    <div ref={ref} className={`relative ${className}`}>
      <button type="button" onClick={() => setOpen(!open)}
        className={`flex items-center justify-between gap-2 px-3.5 py-2 backdrop-blur-md border rounded-xl text-sm transition-all shadow-sm min-w-[120px] ${
          isDark
            ? 'bg-slate-700/70 border-slate-600/60 hover:bg-slate-600/80 text-slate-200'
            : 'bg-white/70 border-gray-200/60 hover:bg-white/90 text-gray-800'
        }`}>
        <span className={selected ? '' : isDark ? 'text-slate-500' : 'text-gray-400'}>
          {selected?.label || placeholder}
        </span>
        <ChevronDown className={`w-4 h-4 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className={`absolute z-50 mt-1.5 w-full min-w-[160px] backdrop-blur-xl rounded-xl shadow-xl border overflow-hidden py-1 max-h-60 overflow-y-auto ${
          isDark ? 'bg-slate-800/95 border-slate-600/60' : 'bg-white/95 border-white/60'
        }`}>
          {options.map(opt => (
            <button key={opt.value}
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className={`w-full text-left px-3.5 py-2 text-sm transition-colors flex items-center gap-2 ${
                value === opt.value
                  ? isDark ? 'text-primary-300 bg-primary-900/30 font-medium' : 'text-primary-700 bg-primary-50/50 font-medium'
                  : isDark ? 'text-slate-300 hover:bg-slate-700/70' : 'text-gray-700 hover:bg-primary-50/70'
              }`}>
              {value === opt.value && <Check className="w-3.5 h-3.5 text-primary-500" />}
              <span className={value === opt.value ? '' : 'pl-5'}>{opt.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
```

**options 格式：** `[{ value: '', label: '全部' }, { value: 'A', label: 'A选项' }]`

---

### 3.5 删除确认弹窗模式

不使用浏览器原生 `window.confirm()`，统一使用自定义 Modal：

```jsx
// State
const [deleteTarget, setDeleteTarget] = useState(null);
const [deleting, setDeleting] = useState(false);

// 触发
<button onClick={() => setDeleteTarget(item)}>删除</button>

// Modal
{deleteTarget && (
  <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    onClick={() => !deleting && setDeleteTarget(null)}>
    <div className={`rounded-2xl shadow-2xl w-full max-w-sm overflow-hidden ${isDark ? 'bg-slate-800' : 'bg-white'}`}
      onClick={(e) => e.stopPropagation()}>
      {/* 顶部红色渐变条 */}
      <div className="h-1 bg-gradient-to-r from-red-400 via-red-500 to-red-600" />
      <div className="p-6 text-center">
        {/* AlertTriangle 图标 */}
        <div className={`w-14 h-14 rounded-full flex items-center justify-center mx-auto mb-4 ${isDark ? 'bg-red-900/30' : 'bg-red-50'}`}>
          <AlertTriangle className="w-7 h-7 text-red-500" />
        </div>
        <h3>确认删除</h3>
        <p>详情说明...</p>
      </div>
      <div className="px-6 pb-6 flex items-center gap-3 justify-center">
        <button onClick={() => setDeleteTarget(null)}>取消</button>
        <button onClick={handleDelete} disabled={deleting}
          className="bg-red-500 text-white rounded-xl hover:bg-red-600">
          {deleting ? <Loader2 className="animate-spin" /> : <Trash2 />}
          确认删除
        </button>
      </div>
    </div>
  </div>
)}
```

---

### 3.6 图表 (Recharts) 模式

```jsx
import { BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
  RadarChart, Radar, PolarGrid, PolarAngleAxis, PolarRadiusAxis,
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

// 颜色常量
const COLORS = ['#6C5CE7', '#00B894', '#FDCB6E', '#E17055', '#0984E3', ...];

// 暗色模式适配
<CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e5e7eb'} />
<XAxis tick={{ fontSize: 11, fill: isDark ? '#94a3b8' : '#6b7280' }} />
<Tooltip contentStyle={{
  background: isDark ? '#1e293b' : '#fff',
  border: isDark ? '1px solid #334155' : '1px solid #e5e7eb',
  borderRadius: 8, fontSize: 12
}} />
```

---

## 四、认证与权限

### 4.1 JWT 认证流程
```
POST /api/login → { token, user } → localStorage 存储 → Axios 拦截器自动附加
```

### 4.2 中间件
```go
// AuthMiddleware — 验证 JWT token，设置 user_id, username, role 到 Context
func AuthMiddleware() gin.HandlerFunc { ... }

// AdminMiddleware — 检查 role == "admin"
func AdminMiddleware() gin.HandlerFunc { ... }
```

### 4.3 前端权限控制
- `App.jsx`: `PrivateRoute` 组件检查 token 存在
- `MainLayout.jsx`: `ADMIN_PAGES` Set 控制页面访问（非管理员看到"需要管理员权限"提示）
- `Sidebar.jsx`: 根据 `user.role` 决定显示 `userMenuGroups` 还是 `adminMenuGroups`

---

## 五、部署架构

### 5.1 Docker Compose 四容器编排

```
[Browser] → :80 → [nginx/frontend] → /api/* → [backend:8080]
                                                   ↓
                                            [mysql:3306]
                                            [rabbitmq:5672]
```

| 容器 | 镜像 | 暴露端口 | 说明 |
|------|------|---------|------|
| `mysql` | mysql:8.0 | 内部 3306 | utf8mb4_general_ci |
| `rabbitmq` | rabbitmq:3.13-management-alpine | 内部 5672/15672 | 消息队列 |
| `backend` | 自建 (Go alpine) | 内部 8080 | Gin REST API |
| `frontend` | 自建 (nginx alpine) | 宿主 80 | SPA + 反向代理 |

### 5.2 Nginx 关键配置
```nginx
location /api/ {
    proxy_pass http://backend:8080;
    proxy_buffering off;       # SSE 必须关闭缓冲
    proxy_cache off;
    proxy_set_header X-Accel-Buffering no;
    proxy_http_version 1.1;    # WebSocket 支持
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 300s;   # AI 响应超时
    client_max_body_size 50m;  # 文件上传
}

location / {
    try_files $uri $uri/ /index.html;  # SPA history 模式
}
```

### 5.3 环境变量
后端通过环境变量配置，无配置文件：
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_USER`, `RABBITMQ_PASSWORD`
- `AI_PROVIDER`, `AI_API_KEY`, `AI_BASE_URL`, `AI_MODEL`
- `JWT_SECRET`, `GIN_MODE`, `SERVER_PORT`
- `DB_DRIVER` (可选 sqlite)

### 5.4 部署命令
```bash
# 首次部署
docker-compose up -d --build

# 更新某个服务
docker-compose up -d --build backend    # 仅重建后端
docker-compose up -d --build frontend   # 仅重建前端

# 查看日志
docker-compose logs -f backend

# 完全重启
docker-compose down && docker-compose up -d --build
```

---

## 六、开发常用操作

### 6.1 新增功能完整流程（以"客户管理"为例）

```bash
# ===== 后端 =====

# 1. model/models.go — 添加 Customer struct
# 2. repository/db.go — migrationModels + migrationOrder 添加 Customer
# 3. handler/customer_handlers.go — 创建 CRUD handler
# 4. cmd/server/main.go — 初始化 handler + 注册路由
# 5. 构建验证
cd backend && go build ./...

# ===== 前端 =====

# 6. services/api.js — 添加 API 函数
# 7. pages/CustomerPage.jsx — 创建页面
# 8. components/Sidebar.jsx — 添加菜单项 (icon import + items 数组)
# 9. components/MainLayout.jsx — 注册到 pageComponents/PAGE_META/ADMIN_PAGES
# 10. 构建验证
cd frontend && npm run build
```

### 6.2 Git 推送
```bash
git add -A
git commit -m "feat(customer): 新增客户管理功能"
git -c credential.helper='!gh auth git-credential' push origin main
```

### 6.3 前端调试
```bash
cd frontend && npm run dev  # localhost:3000, 自动代理 /api → localhost:8080
```

### 6.4 后端调试
```bash
cd backend && go run ./cmd/server  # localhost:8080
# 需要 MySQL 和 RabbitMQ（可用 DB_DRIVER=sqlite 跳过 MySQL）
```

---

## 七、设计规范参考

### 7.1 颜色体系
- **Primary**: `#513CC8` (深紫) — 按钮、激活态、重点信息
- **Success**: `#00B894` — 成功、赢单
- **Warning**: `#FDCB6E` — 警告
- **Danger**: `#E17055` — 错误、删除
- **Info**: `#0984E3` — 信息

### 7.2 组件圆角
- 卡片: `rounded-xl` (12px)
- 按钮: `rounded-lg` (8px) 或 `rounded-xl` (12px)
- 输入框: `rounded-lg` (8px)
- 弹窗: `rounded-2xl` (16px)
- 头像/徽章: `rounded-full`

### 7.3 间距
- 页面padding: `p-6`
- 卡片内padding: `p-4`
- 元素间距: `gap-3` 或 `space-y-4`

### 7.4 字号
- 页面标题: `text-lg font-semibold`
- 卡片标题: `text-sm font-semibold`
- 正文: `text-sm`
- 辅助文字: `text-xs`
- KPI 数值: `text-xl font-bold`

### 7.5 Tab 切换样式
```jsx
<div className={`flex rounded-lg border ${isDark ? 'border-slate-700' : 'border-gray-200'}`}>
  {tabs.map((t) => (
    <button key={t.id} onClick={() => setActiveTab(t.id)}
      className={`flex items-center gap-1.5 px-3 py-1.5 text-sm transition-colors ${
        activeTab === t.id
          ? 'bg-primary-600 text-white'
          : isDark ? 'text-slate-300 hover:bg-slate-700' : 'text-gray-600 hover:bg-gray-50'
      }`}>
      <Icon className="w-3.5 h-3.5" /> {t.label}
    </button>
  ))}
</div>
```

---

## 八、已知陷阱和经验

1. **Excel 列头匹配**：使用 `strings.Contains` 模糊匹配时，必须按字符串长度降序排列 `colNames`，否则短名会截获长名列。

2. **GORM 软删除 + utf8mb4_unicode_ci**：组合唯一索引 + DeletedAt 可能超过 3072 bytes key length 限制。项目强制使用 `utf8mb4_general_ci`。

3. **暗色模式 CSS 覆盖**：Tailwind 的 `dark:` prefix 在本项目中**不使用**，而是通过 `[data-theme="dark"]` + `!important` 全局覆盖。页面组件通过 `isDark` 三元运算符控制类名。

4. **Nginx SSE 缓冲**：AI 流式响应必须设置 `proxy_buffering off` + `proxy_cache off`，否则 token 全部缓冲后一次性返回。

5. **Docker 内网络不暴露端口**：MySQL 和 RabbitMQ 的端口不暴露到宿主机，仅通过 Docker 内部网络连接，避免端口冲突。

6. **前端 SPA 路由**：nginx 的 `try_files $uri $uri/ /index.html` 是关键，否则刷新页面会 404。

7. **原生 `<select>` 和 `window.confirm()`**：不使用，全部用自定义组件替代以保持 UI 一致性和暗色模式兼容。

8. **Axios 拦截器 401 处理**：401 响应自动清除 token 并跳转登录页，但不对 `/login` 请求本身进行 401 跳转（避免循环）。

---

## 九、平移到新项目 Checklist

如果要以此项目为基础创建一个新项目：

### 最小化启动（保留框架，去除业务）：

#### 后端保留：
- [ ] `cmd/server/main.go` — 保留框架，去除业务 handler 和 service 初始化
- [ ] `internal/config/config.go` — 原样保留
- [ ] `internal/model/models.go` — 仅保留 User + AIProvider 模型
- [ ] `internal/repository/db.go` — 仅保留 InitDB + User 迁移 + seedDefaultData（admin 账号）
- [ ] `internal/middleware/auth.go` — 原样保留
- [ ] `internal/handler/handlers.go` — 保留 Login + HealthCheck + Profile + User CRUD
- [ ] `internal/service/auth_service.go` — 原样保留（JWT 生成/验证）
- [ ] `pkg/logger/logger.go` — 原样保留
- [ ] `pkg/response/response.go` — 原样保留
- [ ] `Dockerfile` — 原样保留
- [ ] `go.mod` — 更改 module name

#### 前端保留：
- [ ] `App.jsx` — 原样保留
- [ ] `main.jsx` — 原样保留
- [ ] `store/useStore.js` — 原样保留
- [ ] `services/api.js` — 保留 Axios 实例 + login/getProfile，去除业务 API
- [ ] `styles/index.css` — 原样保留（完整暗色模式系统）
- [ ] `components/MainLayout.jsx` — 保留框架，清空 pageComponents/PAGE_META
- [ ] `components/Sidebar.jsx` — 保留框架，清空菜单组
- [ ] `pages/LoginPage.jsx` — 原样保留
- [ ] `pages/DashboardPage.jsx` — 保留或替换为空壳
- [ ] 配置文件 — vite.config.js, tailwind.config.js, postcss.config.js, package.json, nginx.conf, Dockerfile

#### 基础设施保留：
- [ ] `docker-compose.yml` — 更改容器名和网络名
- [ ] `.gitignore` — 原样保留

然后按照 **第六章** 的新增功能流程，逐个添加新业务模块即可。
