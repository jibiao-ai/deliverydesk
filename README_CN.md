# DeliveryDesk - 云交付服务台

<p align="center">
  <strong>智能化云交付服务工作台</strong><br/>
  基于 Go + React + MySQL + RabbitMQ 构建的一站式云交付运维平台
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Version-2.1.1-blue?style=flat-square" alt="Version"/>
  <img src="https://img.shields.io/badge/Go-1.22-00ADD8?style=flat-square&logo=go" alt="Go"/>
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react" alt="React"/>
  <img src="https://img.shields.io/badge/MySQL-8.0-4479A1?style=flat-square&logo=mysql" alt="MySQL"/>
  <img src="https://img.shields.io/badge/RabbitMQ-3.13-FF6600?style=flat-square&logo=rabbitmq" alt="RabbitMQ"/>
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker" alt="Docker"/>
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License"/>
</p>

<p align="center">
  English | <a href="./README_CN.md">中文文档</a>（本文件）
</p>

---

## 目录

- [项目简介](#项目简介)
- [系统架构](#系统架构)
- [核心功能](#核心功能)
- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [快速开始](#快速开始)
- [API 接口说明](#api-接口说明)
- [数据模型](#数据模型)
- [配置说明](#配置说明)
- [开发指南](#开发指南)
- [常见问题](#常见问题)
- [许可证](#许可证)

---

## 项目简介

**DeliveryDesk（云交付服务台）** 是一个面向云交付团队的智能化工作平台。平台整合了企业 LDAP 认证、AI 智能体系统、常用网站导航、用户管理等核心功能，旨在提升交付团队的工作效率和协作能力。

### 为什么需要 DeliveryDesk？

- **统一入口**：将分散的交付工具、文档、系统统一到一个平台
- **智能辅助**：通过 AI 智能体提供交付黄页、技能知识等即时查询
- **企业级认证**：支持 LDAP/AD 统一身份认证，无缝融入企业 IT 体系
- **快速访问**：常用网站分类导航，一键直达各个系统和工具

---

## 系统架构

### 整体架构图

```
                        ┌─────────────────────────────────────────────┐
                        │              用户浏览器 (Browser)             │
                        └────────────────────┬────────────────────────┘
                                             │ HTTP/HTTPS
                                             ▼
                        ┌─────────────────────────────────────────────┐
                        │           Nginx 反向代理 (Port 80)           │
                        │  ┌──────────────┐  ┌──────────────────────┐ │
                        │  │ 静态资源服务   │  │ /api/* -> Backend    │ │
                        │  │ React SPA    │  │ 反向代理 (Port 8080) │ │
                        │  └──────────────┘  └──────────────────────┘ │
                        └────────────────────┬────────────────────────┘
                                             │
                        ┌────────────────────┼────────────────────────┐
                        │                    ▼                        │
                        │  ┌─────────────────────────────────────┐    │
                        │  │      Go Backend (Gin Framework)     │    │
                        │  │           Port 8080                 │    │
                        │  │                                     │    │
                        │  │  ┌───────────┐  ┌──────────────┐   │    │
                        │  │  │ JWT 认证   │  │ CORS 中间件   │   │    │
                        │  │  └─────┬─────┘  └──────────────┘   │    │
                        │  │        │                            │    │
                        │  │  ┌─────▼─────────────────────────┐  │    │
                        │  │  │       API Handler Layer       │  │    │
                        │  │  │                               │  │    │
                        │  │  │ /api/login     登录认证        │  │    │
                        │  │  │ /api/agents    智能体管理      │  │    │
                        │  │  │ /api/skills    技能管理        │  │    │
                        │  │  │ /api/chat      AI 对话        │  │    │
                        │  │  │ /api/websites  网站导航        │  │    │
                        │  │  │ /api/users     用户管理        │  │    │
                        │  │  │ /api/ldap      LDAP 配置      │  │    │
                        │  │  │ /api/ai-providers  模型管理    │  │    │
                        │  │  └─────┬─────────────────────────┘  │    │
                        │  │        │                            │    │
                        │  │  ┌─────▼─────────────────────────┐  │    │
                        │  │  │       Service Layer           │  │    │
                        │  │  │                               │  │    │
                        │  │  │ AuthService  (认证服务)        │  │    │
                        │  │  │ ChatService  (对话服务)        │  │    │
                        │  │  │ UserService  (用户服务)        │  │    │
                        │  │  └───────┬───────────┬───────────┘  │    │
                        │  │          │           │              │    │
                        │  └──────────┼───────────┼──────────────┘    │
                        │             │           │                   │
                        │   ┌─────────▼──┐  ┌────▼───────────┐       │
                        │   │  MySQL 8.0 │  │  RabbitMQ 3.13 │       │
                        │   │ Port 3306  │  │  Port 5672     │       │
                        │   │            │  │  管理界面 15672  │       │
                        │   │ 持久化数据  │  │  异步任务队列    │       │
                        │   └────────────┘  └────────────────┘       │
                        │           Docker Compose 编排               │
                        └─────────────────────────────────────────────┘
                                             │
                        ┌────────────────────┼────────────────────────┐
                        │    外部服务集成     │                        │
                        │                    ▼                        │
                        │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
                        │  │ 企业 LDAP │  │ AI 模型  │  │ 外部系统  │  │
                        │  │ /AD 服务  │  │ API 服务  │  │ Jira 等  │  │
                        │  └──────────┘  └──────────┘  └──────────┘  │
                        └─────────────────────────────────────────────┘
```

### 前后端交互架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend (React 18 + Vite)                   │
│                                                                     │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────┐ ┌────────────────┐  │
│  │  Zustand     │ │  React      │ │  Axios   │ │  Tailwind CSS  │  │
│  │  状态管理    │ │  Router     │ │  HTTP    │ │  样式框架       │  │
│  └──────┬──────┘ └──────┬──────┘ └────┬─────┘ └────────────────┘  │
│         │               │             │                            │
│  ┌──────▼───────────────▼─────────────▼──────────────────────────┐ │
│  │                      页面组件 (Pages)                          │ │
│  │                                                               │ │
│  │  LoginPage ─ DashboardPage ─ ChatPage ─ AgentsPage            │ │
│  │  SkillsPage ─ AIModelsPage ─ WebsitesPage ─ UsersPage         │ │
│  │  LDAPPage ─ OperationLogPage                                  │ │
│  └───────────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬──────────────────────────────────────┘
                               │  RESTful API (JSON)
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Backend (Go + Gin)                            │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Middleware Layer                            │  │
│  │  CORS ── JWT Auth ── Admin RBAC ── Logger ── Recovery         │  │
│  └───────────────────────────┬───────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼───────────────────────────────────┐  │
│  │                    Handler Layer (API)                         │  │
│  │                                                               │  │
│  │  Auth Handler ── Agent Handler ── Chat Handler                │  │
│  │  Skill Handler ── User Handler ── LDAP Handler                │  │
│  │  AIProvider Handler ── Website Handler ── Log Handler          │  │
│  └───────────────────────────┬───────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼───────────────────────────────────┐  │
│  │                    Service Layer (业务逻辑)                    │  │
│  │                                                               │  │
│  │  AuthService ── ChatService ── UserService                    │  │
│  │  (JWT生成/验证, LDAP认证, 密码加密)                             │  │
│  │  (AI对话, 流式响应, 上下文管理)                                  │  │
│  │  (用户CRUD, 角色管理)                                          │  │
│  └─────────┬──────────────────────────────────┬──────────────────┘  │
│            │                                  │                     │
│  ┌─────────▼───────────────────┐  ┌───────────▼─────────────────┐  │
│  │    Repository Layer         │  │    Message Queue Layer       │  │
│  │    (GORM ORM)               │  │    (RabbitMQ Client)        │  │
│  │                             │  │                             │  │
│  │  MySQL 8.0                  │  │  异步任务发布/消费            │  │
│  │  自动迁移 (Auto Migration)   │  │  队列: agent_task           │  │
│  │  数据种子 (Seed Data)        │  │  重试机制                   │  │
│  └─────────────────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### 认证流程架构

```
                  ┌──────────┐
                  │  用户登录  │
                  └─────┬────┘
                        │
                  ┌─────▼─────┐
                  │ 选择认证方式 │
                  └──┬─────┬──┘
                     │     │
          本地认证    │     │  LDAP认证
                     │     │
              ┌──────▼┐  ┌─▼──────────┐
              │ MySQL  │  │ 获取默认LDAP │
              │ 密码校验│  │ 或指定配置   │
              └───┬────┘  └──────┬─────┘
                  │              │
                  │         ┌────▼────────┐
                  │         │ LDAP Bind   │
                  │         │ 认证+搜索    │
                  │         └────┬────────┘
                  │              │
                  │         ┌────▼────────┐
                  │         │ 用户是否存在？│
                  │         └──┬─────┬────┘
                  │         否 │     │ 是
                  │      ┌────▼──┐  │
                  │      │自动创建│  │
                  │      │本地账户│  │
                  │      └────┬──┘  │
                  │           │     │
                  └─────┬─────┴─────┘
                        │
                  ┌─────▼─────┐
                  │ 生成 JWT   │
                  │ Token     │
                  └─────┬─────┘
                        │
                  ┌─────▼─────┐
                  │ 返回登录态  │
                  └───────────┘
```

### 部署架构（Docker Compose）

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Stack                     │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │  deliverydesk-frontend (Nginx:80)                │   │
│  │  - React SPA 静态资源                             │   │
│  │  - /api/* 反向代理至 backend:8080                 │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │ depends_on                        │
│  ┌──────────────────▼───────────────────────────────┐   │
│  │  deliverydesk-backend (Go:8080)                  │   │
│  │  - RESTful API 服务                               │   │
│  │  - JWT + LDAP 认证                                │   │
│  │  - AI 模型调用代理                                 │   │
│  └──────┬────────────────────────┬──────────────────┘   │
│         │ depends_on (healthy)   │ depends_on           │
│  ┌──────▼──────────┐  ┌─────────▼──────────────────┐   │
│  │ deliverydesk-   │  │  deliverydesk-rabbitmq     │   │
│  │ mysql (3306)    │  │  (5672 / 15672)            │   │
│  │                 │  │                            │   │
│  │ 数据持久化:      │  │  管理界面: 15672            │   │
│  │ mysql_data      │  │  数据持久化: rabbitmq_data   │   │
│  └─────────────────┘  └────────────────────────────┘   │
│                                                          │
│  Network: deliverydesk-net (bridge)                      │
└─────────────────────────────────────────────────────────┘
```

---

## 核心功能

### 1. 企业 LDAP 认证管理

| 功能 | 说明 |
|------|------|
| LDAP 服务器管理 | 管理员可在页面上添加、编辑、删除企业 LDAP/AD 服务器配置 |
| 连接测试 | 一键测试 LDAP 连接是否正常 |
| 多源支持 | 支持配置多个 LDAP 源，设置默认认证源 |
| 用户自动注册 | LDAP 用户首次登录自动创建本地账户 |
| 双模式登录 | 登录页支持切换本地认证 / LDAP 认证 |
| TLS 加密 | 支持 LDAPS 安全连接 |

### 2. AI 智能体系统

| 功能 | 说明 |
|------|------|
| 模型厂商配置 | 页面化管理 AI 模型提供商（OpenAI、DeepSeek、通义千问、智谱GLM、硅基流动等） |
| 智能体管理 | 创建、编辑、删除智能体，配置系统提示词、模型参数 |
| 技能(Skills)系统 | 为智能体绑定技能：交付黄页、安装指南、升级手册、运维巡检、网络配置 |
| 交付黄页智能体 | 内置智能体，提供交付知识库查询 |
| 交付技能智能体 | 内置智能体，提供部署/运维技能指导 |
| 实时对话 | 与 AI 智能体进行实时对话，支持上下文记忆 |

### 3. 常用网站导航

将团队常用的 32+ 个网站链接按 8 个分类组织，支持搜索和一键访问：

| 分类 | 包含链接 |
|------|---------|
| 常用工具 | 商业存储对接包、拓扑制作系统、Redmine、Jira、Confluence、VPN、企业邮箱、企业网盘 |
| 网络与配置 | 网络配置包制作、云杉网络对接包 |
| 知识与文档 | Confluence使用介绍、交付黄页、合思费控报销 |
| 云产品介质 | V6.x标准安装介质、产品升级介质 |
| 部署手册 | V6.x安装部署手册、实施阶段文档模版 |
| 运维知识 | 云平台勘误、万博迁移、驻场运维规范、技能培训 |
| 产品变更 | V6.x标准变更/勘误 |
| 专项交付 | SDN服务问题、V6运维手册库、邮储专项交付、标准镜像制作与维护 |

### 4. 系统管理

| 功能 | 说明 |
|------|------|
| 用户管理 | 管理员可 CRUD 用户，分配角色（admin/user） |
| 操作日志 | 记录所有管理员操作，支持模块、操作类型过滤 |
| RBAC 权限 | 基于角色的访问控制，区分普通用户和管理员 |
| 仪表盘 | 统计展示网站链接数、AI模型数、智能体数、对话数 |

---

## 技术栈

### 后端

| 技术 | 版本 | 说明 |
|------|------|------|
| Go | 1.22 | 后端开发语言 |
| Gin | 1.10 | 高性能 HTTP Web 框架 |
| GORM | 1.30 | Go 语言 ORM 框架，支持 MySQL/SQLite |
| MySQL | 8.0 | 主数据库，utf8mb4 字符集 |
| RabbitMQ | 3.13 | 消息队列，用于异步任务处理 |
| JWT | - | 无状态身份认证 |
| LDAP | - | 企业级统一身份认证 |
| Logrus | 1.9 | 结构化日志框架 |
| bcrypt | - | 密码安全加密 |

### 前端

| 技术 | 版本 | 说明 |
|------|------|------|
| React | 18 | 前端 UI 框架 |
| Vite | 5.4 | 下一代前端构建工具 |
| Tailwind CSS | 3.x | 原子化 CSS 框架 |
| Zustand | 4.x | 轻量级状态管理 |
| Axios | 1.x | HTTP 客户端，含重试机制 |
| Lucide React | - | 图标库 |
| React Router | 6.x | 客户端路由 |
| React Hot Toast | - | 消息通知 |

### 基础设施

| 技术 | 说明 |
|------|------|
| Docker | 容器化部署 |
| Docker Compose | 多容器编排 |
| Nginx | 反向代理 + 静态资源服务 |
| 多阶段构建 | 优化 Docker 镜像体积 |

---

## 项目结构

```
deliverydesk/
├── README.md                          # 英文文档
├── README_CN.md                       # 中文文档（本文件）
├── docker-compose.yml                 # Docker Compose 编排文件
├── .env.example                       # 环境变量模板
├── .gitignore                         # Git 忽略配置
│
├── backend/                           # Go 后端服务
│   ├── Dockerfile                     # 后端 Docker 多阶段构建
│   ├── go.mod                         # Go 模块依赖
│   ├── go.sum                         # 依赖校验
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                # 应用入口，路由定义
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go              # 配置加载（环境变量）
│   │   ├── handler/
│   │   │   └── handlers.go            # HTTP API 处理器
│   │   ├── middleware/
│   │   │   └── auth.go                # JWT认证 + 管理员权限中间件
│   │   ├── model/
│   │   │   └── models.go              # GORM 数据模型定义
│   │   ├── mq/
│   │   │   └── rabbitmq.go            # RabbitMQ 客户端
│   │   ├── repository/
│   │   │   └── db.go                  # 数据库初始化 + 种子数据
│   │   └── service/
│   │       ├── auth_service.go        # 认证服务（JWT/LDAP）
│   │       ├── chat_service.go        # 对话服务（AI调用）
│   │       └── user_service.go        # 用户管理服务
│   └── pkg/
│       ├── logger/
│       │   └── logger.go              # 日志封装（Logrus）
│       └── response/
│           └── response.go            # 统一 API 响应格式
│
├── frontend/                          # React 前端应用
│   ├── Dockerfile                     # 前端 Docker 多阶段构建
│   ├── nginx.conf                     # Nginx 反向代理配置
│   ├── index.html                     # HTML 入口
│   ├── package.json                   # NPM 依赖配置
│   ├── vite.config.js                 # Vite 构建配置
│   ├── tailwind.config.js             # Tailwind CSS 配置
│   ├── postcss.config.js              # PostCSS 配置
│   └── src/
│       ├── main.jsx                   # React 入口
│       ├── App.jsx                    # 应用路由配置
│       ├── components/
│       │   ├── MainLayout.jsx         # 主布局（侧边栏+内容区）
│       │   └── Sidebar.jsx            # 侧边栏导航
│       ├── pages/
│       │   ├── LoginPage.jsx          # 登录页（本地/LDAP切换）
│       │   ├── DashboardPage.jsx      # 仪表盘
│       │   ├── ChatPage.jsx           # AI 对话页
│       │   ├── AgentsPage.jsx         # 智能体管理
│       │   ├── SkillsPage.jsx         # 技能管理
│       │   ├── AIModelsPage.jsx       # AI 模型配置
│       │   ├── WebsitesPage.jsx       # 常用网站导航
│       │   ├── UsersPage.jsx          # 用户管理
│       │   ├── LDAPPage.jsx           # LDAP 配置管理
│       │   └── OperationLogPage.jsx   # 操作日志
│       ├── services/
│       │   └── api.js                 # API 客户端（Axios）
│       ├── store/
│       │   └── useStore.js            # Zustand 全局状态
│       └── styles/
│           └── index.css              # 全局样式
│
├── docker/                            # Docker 配置目录
├── docs/                              # 项目文档
└── scripts/                           # 脚本工具
```

---

## 快速开始

### 环境要求

- Docker >= 20.10
- Docker Compose >= 2.0
- （开发模式）Go >= 1.22, Node.js >= 18

### 方式一：Docker Compose 部署（推荐）

```bash
# 1. 克隆项目
git clone https://github.com/jibiao-ai/deliverydesk.git
cd deliverydesk

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 文件，填入你的 AI API Key
vim .env

# 3. 启动所有服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看日志
docker-compose logs -f backend
```

### 方式二：开发模式

```bash
# 后端（支持 SQLite 开发模式，无需 MySQL）
cd backend
export DB_DRIVER=sqlite
go run ./cmd/server

# 前端
cd frontend
npm install
npm run dev
# 访问 http://localhost:3000
```

### 默认管理员账号

| 项目 | 值 |
|------|------|
| 用户名 | `admin` |
| 密码 | `Admin@2024!` |
| 角色 | 管理员（admin） |

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| 前端 (Nginx) | 80 | Web 界面 |
| 后端 (Go API) | 8080 | REST API |
| MySQL | 3306 | 数据库 |
| RabbitMQ | 5672 | 消息队列 |
| RabbitMQ 管理 | 15672 | 管理界面 (guest/guest) |

---

## API 接口说明

### 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/login` | 用户登录（本地/LDAP） |

### 认证接口（需 JWT Token）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/profile` | 获取当前用户信息 |
| GET | `/api/dashboard` | 获取仪表盘统计数据 |
| GET | `/api/agents` | 获取智能体列表 |
| GET | `/api/agents/:id` | 获取智能体详情 |
| POST | `/api/agents` | 创建智能体 |
| PUT | `/api/agents/:id` | 更新智能体 |
| DELETE | `/api/agents/:id` | 删除智能体 |
| GET | `/api/conversations` | 获取对话列表 |
| POST | `/api/conversations` | 创建对话 |
| DELETE | `/api/conversations/:id` | 删除对话 |
| GET | `/api/conversations/:id/messages` | 获取对话消息 |
| POST | `/api/conversations/:id/messages` | 发送消息 |
| GET | `/api/skills` | 获取技能列表 |
| GET | `/api/agents/:id/skills` | 获取智能体关联技能 |
| GET | `/api/ai-providers` | 获取 AI 模型列表 |
| PUT | `/api/ai-providers/:id` | 更新 AI 模型配置 |
| POST | `/api/ai-providers/:id/test` | 测试 AI 模型连接 |
| GET | `/api/website-categories` | 获取网站分类及链接 |

### 管理员接口（需 Admin 角色）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/users` | 获取用户列表 |
| POST | `/api/users` | 创建用户 |
| PUT | `/api/users/:id` | 更新用户 |
| DELETE | `/api/users/:id` | 删除用户 |
| GET | `/api/ldap-configs` | 获取 LDAP 配置列表 |
| POST | `/api/ldap-configs` | 创建 LDAP 配置 |
| PUT | `/api/ldap-configs/:id` | 更新 LDAP 配置 |
| DELETE | `/api/ldap-configs/:id` | 删除 LDAP 配置 |
| POST | `/api/ldap-configs/:id/test` | 测试 LDAP 连接 |
| GET | `/api/operation-logs` | 获取操作日志 |

---

## 数据模型

### ER 关系图

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│     User     │     │    Agent     │     │    Skill     │
├──────────────┤     ├──────────────┤     ├──────────────┤
│ id           │     │ id           │     │ id           │
│ username     │     │ name         │     │ name         │
│ password     │     │ description  │     │ description  │
│ email        │     │ system_prompt│     │ type         │
│ display_name │     │ model        │     │ config       │
│ role         │     │ temperature  │     │ tool_defs    │
│ auth_type    │     │ max_tokens   │     │ is_active    │
│ avatar       │     │ is_active    │     └──────┬───────┘
└──────┬───────┘     │ created_by   │            │
       │             └──────┬───────┘            │
       │                    │                    │
       │             ┌──────▼───────┐            │
       │             │  AgentSkill  │◄───────────┘
       │             ├──────────────┤
       │             │ agent_id (FK)│
       │             │ skill_id (FK)│
       │             └──────────────┘
       │
       │  ┌──────────────┐     ┌──────────────┐
       ├─►│ Conversation │────►│   Message    │
       │  ├──────────────┤     ├──────────────┤
       │  │ id           │     │ id           │
       │  │ title        │     │ conversation │
       │  │ user_id (FK) │     │ role         │
       │  │ agent_id (FK)│     │ content      │
       │  └──────────────┘     │ tokens_used  │
       │                       │ tool_calls   │
       │                       └──────────────┘
       │
       │  ┌──────────────┐     ┌──────────────┐
       └─►│ OperationLog │     │  LDAPConfig  │
          ├──────────────┤     ├──────────────┤
          │ user_id (FK) │     │ name         │
          │ module       │     │ host / port  │
          │ action       │     │ bind_dn      │
          │ target       │     │ base_dn      │
          │ detail       │     │ user_filter  │
          └──────────────┘     │ is_default   │
                               └──────────────┘

┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  AIProvider  │  │   TaskLog    │  │ WebsiteCateg │
├──────────────┤  ├──────────────┤  ├──────────────┤
│ name / label │  │ task_id      │  │ name / icon  │
│ api_key      │  │ type/status  │  │ sort_order   │
│ base_url     │  │ input/output │  │              │
│ model        │  │ user_id (FK) │  │   ┌──────────┤
│ is_default   │  └──────────────┘  │   │ Website  │
│ is_enabled   │                    │   │  Link    │
└──────────────┘                    │   ├──────────┤
                                    │   │ name/url │
                                    │   │ icon     │
                                    └───┴──────────┘
```

---

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_PORT` | `8080` | 后端服务端口 |
| `GIN_MODE` | `debug` | Gin 运行模式 (debug/release) |
| `DB_DRIVER` | `mysql` | 数据库驱动 (mysql/sqlite) |
| `DB_HOST` | `mysql` | MySQL 主机地址 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `deliverydesk` | MySQL 用户名 |
| `DB_PASSWORD` | `deliverydesk123` | MySQL 密码 |
| `DB_NAME` | `deliverydesk` | 数据库名称 |
| `RABBITMQ_HOST` | `rabbitmq` | RabbitMQ 主机地址 |
| `RABBITMQ_PORT` | `5672` | RabbitMQ 端口 |
| `RABBITMQ_USER` | `guest` | RabbitMQ 用户名 |
| `RABBITMQ_PASSWORD` | `guest` | RabbitMQ 密码 |
| `AI_PROVIDER` | `openai` | 默认 AI 提供商 |
| `AI_API_KEY` | - | AI API 密钥 |
| `AI_BASE_URL` | `https://api.openai.com/v1` | AI API 基础地址 |
| `AI_MODEL` | `gpt-4` | 默认 AI 模型 |
| `JWT_SECRET` | `deliverydesk-secret-key-2024` | JWT 签名密钥 |

---

## 开发指南

### 后端开发

```bash
cd backend

# 使用 SQLite 进行本地开发（无需 MySQL）
export DB_DRIVER=sqlite
go run ./cmd/server

# 运行测试
go test ./...

# 构建
go build -o deliverydesk ./cmd/server
```

### 前端开发

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器（自动代理 /api 到 localhost:8080）
npm run dev

# 构建生产版本
npm run build
```

### 添加新页面

1. 在 `frontend/src/pages/` 创建新页面组件
2. 在 `frontend/src/components/MainLayout.jsx` 注册页面
3. 在 `frontend/src/components/Sidebar.jsx` 添加导航菜单项

### 添加新 API

1. 在 `backend/internal/model/models.go` 定义数据模型
2. 在 `backend/internal/handler/handlers.go` 添加 Handler 方法
3. 在 `backend/cmd/server/main.go` 注册路由
4. 在 `frontend/src/services/api.js` 添加 API 调用函数

---

## 常见问题

### Q: 数据库连接失败？
确认 MySQL 服务已启动，检查 `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` 环境变量是否正确。Docker Compose 模式下，后端会自动等待 MySQL 健康检查通过后再启动。

### Q: RabbitMQ 连接失败会影响使用吗？
不会。系统设计为 RabbitMQ 连接失败时仅打印警告日志，核心功能正常运行。异步任务功能在 MQ 恢复后自动可用。

### Q: 如何配置 LDAP 认证？
1. 使用管理员账号登录
2. 进入"LDAP 配置"页面
3. 点击"添加 LDAP 服务器"
4. 填入 LDAP 服务器信息（Host, Port, Bind DN, Base DN 等）
5. 点击"测试连接"验证配置
6. 启用配置后，用户即可在登录页选择 LDAP 认证

### Q: 如何添加新的 AI 模型提供商？
进入"AI 模型"页面，系统已预置 OpenAI、DeepSeek、通义千问、智谱GLM、硅基流动等提供商，填入对应的 API Key 即可启用。

---

## 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

---

<p align="center">
  <strong>DeliveryDesk</strong> - 让云交付更智能、更高效
</p>
