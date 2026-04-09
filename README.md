# DeliveryDesk - Cloud Delivery Workbench

<p align="center">
  <strong>Intelligent Cloud Delivery Service Workbench</strong><br/>
  A unified delivery operations platform built with Go + React + MySQL + RabbitMQ
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
  <a href="./README_CN.md">中文文档</a> | English
</p>

---

## Table of Contents

- [Overview](#overview)
- [System Architecture](#system-architecture)
- [Core Features](#core-features)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Data Models](#data-models)
- [Configuration](#configuration)
- [Development Guide](#development-guide)
- [FAQ](#faq)
- [License](#license)

---

## Overview

**DeliveryDesk** is an intelligent workbench platform designed for cloud delivery teams. It integrates enterprise LDAP authentication, AI-powered agents, a website navigation hub, and user management into a single platform, aiming to boost productivity and collaboration for delivery engineers.

### Why DeliveryDesk?

- **Unified Portal**: Consolidate scattered delivery tools, documentation, and systems into one platform
- **AI-Powered Assistance**: Leverage AI agents for instant knowledge queries (delivery directory, skill guides)
- **Enterprise Authentication**: LDAP/AD integration for seamless Single Sign-On within enterprise IT infrastructure
- **Quick Access**: Categorized website navigation with one-click access to 32+ commonly used tools and systems

---

## System Architecture

### High-Level Architecture

```
                        ┌─────────────────────────────────────────────┐
                        │              User Browser (Client)           │
                        └────────────────────┬────────────────────────┘
                                             │ HTTP/HTTPS
                                             ▼
                        ┌─────────────────────────────────────────────┐
                        │         Nginx Reverse Proxy (Port 80)        │
                        │  ┌──────────────┐  ┌──────────────────────┐ │
                        │  │ Static Files │  │ /api/* -> Backend    │ │
                        │  │ React SPA    │  │ Proxy (Port 8080)   │ │
                        │  └──────────────┘  └──────────────────────┘ │
                        └────────────────────┬────────────────────────┘
                                             │
                        ┌────────────────────┼────────────────────────┐
                        │                    ▼                        │
                        │  ┌─────────────────────────────────────┐    │
                        │  │     Go Backend (Gin Framework)      │    │
                        │  │            Port 8080                │    │
                        │  │                                     │    │
                        │  │  ┌───────────┐  ┌──────────────┐   │    │
                        │  │  │ JWT Auth  │  │ CORS Middleware│   │    │
                        │  │  └─────┬─────┘  └──────────────┘   │    │
                        │  │        │                            │    │
                        │  │  ┌─────▼─────────────────────────┐  │    │
                        │  │  │       API Handler Layer       │  │    │
                        │  │  │                               │  │    │
                        │  │  │ /api/login       Auth         │  │    │
                        │  │  │ /api/agents      AI Agents    │  │    │
                        │  │  │ /api/skills      Skills       │  │    │
                        │  │  │ /api/chat        AI Chat      │  │    │
                        │  │  │ /api/websites    Navigation   │  │    │
                        │  │  │ /api/users       Users        │  │    │
                        │  │  │ /api/ldap        LDAP Config  │  │    │
                        │  │  │ /api/ai-providers Models      │  │    │
                        │  │  └─────┬─────────────────────────┘  │    │
                        │  │        │                            │    │
                        │  │  ┌─────▼─────────────────────────┐  │    │
                        │  │  │       Service Layer           │  │    │
                        │  │  │                               │  │    │
                        │  │  │ AuthService  (Authentication) │  │    │
                        │  │  │ ChatService  (AI Chat)        │  │    │
                        │  │  │ UserService  (User Mgmt)      │  │    │
                        │  │  └───────┬───────────┬───────────┘  │    │
                        │  │          │           │              │    │
                        │  └──────────┼───────────┼──────────────┘    │
                        │             │           │                   │
                        │   ┌─────────▼──┐  ┌────▼───────────┐       │
                        │   │  MySQL 8.0 │  │  RabbitMQ 3.13 │       │
                        │   │ Port 3306  │  │  Port 5672     │       │
                        │   │            │  │  Mgmt UI 15672 │       │
                        │   │ Persistent │  │  Async Tasks   │       │
                        │   │ Storage    │  │  Queue         │       │
                        │   └────────────┘  └────────────────┘       │
                        │           Docker Compose Orchestration      │
                        └─────────────────────────────────────────────┘
                                             │
                        ┌────────────────────┼────────────────────────┐
                        │   External Services│                        │
                        │                    ▼                        │
                        │  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
                        │  │ Enterprise│  │ AI Model │  │ External │  │
                        │  │ LDAP / AD │  │ APIs     │  │ Systems  │  │
                        │  └──────────┘  └──────────┘  └──────────┘  │
                        └─────────────────────────────────────────────┘
```

### Frontend-Backend Interaction

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Frontend (React 18 + Vite)                   │
│                                                                     │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────┐ ┌────────────────┐  │
│  │  Zustand     │ │  React      │ │  Axios   │ │  Tailwind CSS  │  │
│  │  State Mgmt  │ │  Router     │ │  HTTP    │ │  Styling       │  │
│  └──────┬──────┘ └──────┬──────┘ └────┬─────┘ └────────────────┘  │
│         │               │             │                            │
│  ┌──────▼───────────────▼─────────────▼──────────────────────────┐ │
│  │                      Page Components                          │ │
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
│  │                    Handler Layer (REST API)                    │  │
│  │                                                               │  │
│  │  Auth Handler ── Agent Handler ── Chat Handler                │  │
│  │  Skill Handler ── User Handler ── LDAP Handler                │  │
│  │  AIProvider Handler ── Website Handler ── Log Handler          │  │
│  └───────────────────────────┬───────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼───────────────────────────────────┐  │
│  │                    Service Layer (Business Logic)              │  │
│  │                                                               │  │
│  │  AuthService ── ChatService ── UserService                    │  │
│  │  (JWT, LDAP Auth, Password Hashing)                           │  │
│  │  (AI Dialogue, Streaming, Context Management)                 │  │
│  │  (User CRUD, Role Management)                                 │  │
│  └─────────┬──────────────────────────────────┬──────────────────┘  │
│            │                                  │                     │
│  ┌─────────▼───────────────────┐  ┌───────────▼─────────────────┐  │
│  │    Repository Layer         │  │    Message Queue Layer       │  │
│  │    (GORM ORM)               │  │    (RabbitMQ Client)        │  │
│  │                             │  │                             │  │
│  │  MySQL 8.0                  │  │  Async Task Publish/Consume │  │
│  │  Auto Migration             │  │  Queue: agent_task          │  │
│  │  Seed Data                  │  │  Retry Mechanism            │  │
│  └─────────────────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Authentication Flow

```
                  ┌──────────┐
                  │  User    │
                  │  Login   │
                  └─────┬────┘
                        │
                  ┌─────▼──────────┐
                  │ Select Auth    │
                  │ Method         │
                  └──┬──────────┬──┘
                     │          │
          Local Auth │          │ LDAP Auth
                     │          │
              ┌──────▼┐   ┌────▼──────────┐
              │ MySQL  │   │ Get Default   │
              │ Pass   │   │ or Specified  │
              │ Verify │   │ LDAP Config   │
              └───┬────┘   └──────┬────────┘
                  │               │
                  │          ┌────▼─────────┐
                  │          │ LDAP Bind    │
                  │          │ Auth & Search│
                  │          └────┬─────────┘
                  │               │
                  │          ┌────▼─────────┐
                  │          │ User Exists? │
                  │          └──┬────────┬──┘
                  │          No │        │ Yes
                  │       ┌────▼───┐    │
                  │       │ Auto   │    │
                  │       │ Create │    │
                  │       │ Account│    │
                  │       └────┬───┘    │
                  │            │        │
                  └─────┬──────┴────────┘
                        │
                  ┌─────▼──────┐
                  │ Generate   │
                  │ JWT Token  │
                  └─────┬──────┘
                        │
                  ┌─────▼──────┐
                  │ Return     │
                  │ Auth State │
                  └────────────┘
```

### Deployment Architecture (Docker Compose)

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Stack                     │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │  deliverydesk-frontend (Nginx:80)                │   │
│  │  - React SPA Static Assets                       │   │
│  │  - /api/* Reverse Proxy -> backend:8080          │   │
│  └──────────────────┬───────────────────────────────┘   │
│                     │ depends_on                        │
│  ┌──────────────────▼───────────────────────────────┐   │
│  │  deliverydesk-backend (Go:8080)                  │   │
│  │  - RESTful API Server                            │   │
│  │  - JWT + LDAP Authentication                     │   │
│  │  - AI Model Proxy                                │   │
│  └──────┬────────────────────────┬──────────────────┘   │
│         │ depends_on (healthy)   │ depends_on           │
│  ┌──────▼──────────┐  ┌─────────▼──────────────────┐   │
│  │ deliverydesk-   │  │  deliverydesk-rabbitmq     │   │
│  │ mysql (3306)    │  │  (5672 / 15672)            │   │
│  │                 │  │                            │   │
│  │ Persistent:     │  │  Management UI: 15672      │   │
│  │ mysql_data      │  │  Persistent: rabbitmq_data │   │
│  └─────────────────┘  └────────────────────────────┘   │
│                                                          │
│  Network: deliverydesk-net (bridge)                      │
└─────────────────────────────────────────────────────────┘
```

---

## Core Features

### 1. Enterprise LDAP Authentication Management

| Feature | Description |
|---------|-------------|
| LDAP Server Management | Admin can add, edit, and delete LDAP/AD server configurations via web UI |
| Connection Testing | One-click test to verify LDAP connectivity |
| Multi-Source Support | Configure multiple LDAP sources with default selection |
| Auto-Provisioning | First-time LDAP users get local accounts created automatically |
| Dual Auth Mode | Login page supports switching between local and LDAP authentication |
| TLS Encryption | Support for LDAPS secure connections |

### 2. AI Agent System

| Feature | Description |
|---------|-------------|
| Model Provider Configuration | UI-based management of AI providers (OpenAI, DeepSeek, Qwen, GLM, SiliconFlow, etc.) |
| Agent Management | Create, edit, delete agents with custom system prompts and model parameters |
| Skills System | Bind skills to agents: Delivery Directory, Install Guide, Upgrade Manual, Ops Inspection, Network Config |
| Delivery Directory Agent | Built-in agent providing delivery knowledge base queries |
| Delivery Skills Agent | Built-in agent providing deployment/operations skill guidance |
| Real-time Chat | Chat with AI agents with context memory support |

### 3. Website Navigation Hub

32+ website links organized into 8 categories with search and one-click access:

| Category | Links Include |
|----------|--------------|
| Common Tools | Storage Package, Topology System, Redmine, Jira, Confluence, VPN, Email, Cloud Drive |
| Network & Config | Network Config Packages, Yunshan Network Packages |
| Knowledge & Docs | Confluence Guide, Delivery Directory, Expense Reports |
| Cloud Products | V6.x Standard Install Media, Product Upgrade Media |
| Deploy Manuals | V6.x Install & Deploy Manuals, Implementation Templates |
| Operations | Platform Errata, Migration Tools, On-site Ops Standards, Training |
| Product Changes | V6.x Standard Changes / Errata |
| Special Delivery | SDN Services, V6 Ops Manual Library, Postal Savings Project, Standard Image Maintenance |

### 4. System Administration

| Feature | Description |
|---------|-------------|
| User Management | Admin CRUD operations with role assignment (admin/user) |
| Operation Logs | Audit trail of all admin operations with module/action filtering |
| RBAC | Role-based access control separating regular users and administrators |
| Dashboard | Statistics overview: website links, AI models, agents, conversations |

---

## Tech Stack

### Backend

| Technology | Version | Purpose |
|------------|---------|---------|
| Go | 1.22 | Backend programming language |
| Gin | 1.10 | High-performance HTTP web framework |
| GORM | 1.30 | ORM framework supporting MySQL/SQLite |
| MySQL | 8.0 | Primary database with utf8mb4 charset |
| RabbitMQ | 3.13 | Message queue for async task processing |
| JWT | - | Stateless authentication tokens |
| LDAP | - | Enterprise identity authentication |
| Logrus | 1.9 | Structured logging framework |
| bcrypt | - | Secure password hashing |

### Frontend

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 18 | UI component framework |
| Vite | 5.4 | Next-generation build tool |
| Tailwind CSS | 3.x | Utility-first CSS framework |
| Zustand | 4.x | Lightweight state management |
| Axios | 1.x | HTTP client with retry logic |
| Lucide React | - | Icon library |
| React Router | 6.x | Client-side routing |
| React Hot Toast | - | Toast notifications |

### Infrastructure

| Technology | Purpose |
|------------|---------|
| Docker | Containerized deployment |
| Docker Compose | Multi-container orchestration |
| Nginx | Reverse proxy + static file serving |
| Multi-stage Build | Optimized Docker image size |

---

## Project Structure

```
deliverydesk/
├── README.md                          # English documentation (this file)
├── README_CN.md                       # Chinese documentation
├── docker-compose.yml                 # Docker Compose orchestration
├── .env.example                       # Environment variable template
├── .gitignore                         # Git ignore rules
│
├── backend/                           # Go backend service
│   ├── Dockerfile                     # Multi-stage Docker build
│   ├── go.mod                         # Go module dependencies
│   ├── go.sum                         # Dependency checksums
│   ├── cmd/
│   │   └── server/
│   │       └── main.go                # Application entry, route definitions
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go              # Configuration loader (env vars)
│   │   ├── handler/
│   │   │   └── handlers.go            # HTTP API handlers
│   │   ├── middleware/
│   │   │   └── auth.go                # JWT auth + admin RBAC middleware
│   │   ├── model/
│   │   │   └── models.go              # GORM data model definitions
│   │   ├── mq/
│   │   │   └── rabbitmq.go            # RabbitMQ client
│   │   ├── repository/
│   │   │   └── db.go                  # Database init + seed data
│   │   └── service/
│   │       ├── auth_service.go        # Auth service (JWT/LDAP)
│   │       ├── chat_service.go        # Chat service (AI calls)
│   │       └── user_service.go        # User management service
│   └── pkg/
│       ├── logger/
│       │   └── logger.go              # Logger wrapper (Logrus)
│       └── response/
│           └── response.go            # Unified API response format
│
├── frontend/                          # React frontend application
│   ├── Dockerfile                     # Multi-stage Docker build
│   ├── nginx.conf                     # Nginx reverse proxy config
│   ├── index.html                     # HTML entry point
│   ├── package.json                   # NPM dependencies
│   ├── vite.config.js                 # Vite build configuration
│   ├── tailwind.config.js             # Tailwind CSS config
│   ├── postcss.config.js              # PostCSS config
│   └── src/
│       ├── main.jsx                   # React entry point
│       ├── App.jsx                    # Application routing
│       ├── components/
│       │   ├── MainLayout.jsx         # Main layout (sidebar + content)
│       │   └── Sidebar.jsx            # Sidebar navigation
│       ├── pages/
│       │   ├── LoginPage.jsx          # Login page (local/LDAP toggle)
│       │   ├── DashboardPage.jsx      # Dashboard
│       │   ├── ChatPage.jsx           # AI chat interface
│       │   ├── AgentsPage.jsx         # Agent management
│       │   ├── SkillsPage.jsx         # Skills management
│       │   ├── AIModelsPage.jsx       # AI model configuration
│       │   ├── WebsitesPage.jsx       # Website navigation hub
│       │   ├── UsersPage.jsx          # User management
│       │   ├── LDAPPage.jsx           # LDAP configuration
│       │   └── OperationLogPage.jsx   # Operation logs
│       ├── services/
│       │   └── api.js                 # API client (Axios)
│       ├── store/
│       │   └── useStore.js            # Zustand global state
│       └── styles/
│           └── index.css              # Global styles
│
├── docker/                            # Docker configuration files
├── docs/                              # Project documentation
└── scripts/                           # Utility scripts
```

---

## Quick Start

### Prerequisites

- Docker >= 20.10
- Docker Compose >= 2.0
- (Development mode) Go >= 1.22, Node.js >= 18

### Option 1: Docker Compose Deployment (Recommended)

```bash
# 1. Clone the repository
git clone https://github.com/jibiao-ai/deliverydesk.git
cd deliverydesk

# 2. Configure environment
cp .env.example .env
# Edit .env and add your AI API key
vim .env

# 3. Start all services
docker-compose up -d

# 4. Check service status
docker-compose ps

# 5. View logs
docker-compose logs -f backend
```

### Option 2: Development Mode

```bash
# Backend (supports SQLite for local dev, no MySQL needed)
cd backend
export DB_DRIVER=sqlite
go run ./cmd/server

# Frontend
cd frontend
npm install
npm run dev
# Open http://localhost:3000
```

### Default Admin Credentials

| Field | Value |
|-------|-------|
| Username | `admin` |
| Password | `Admin@2024!` |
| Role | Administrator (admin) |

### Service Ports

| Service | Port | Description |
|---------|------|-------------|
| Frontend (Nginx) | 80 | Web UI |
| Backend (Go API) | 8080 | REST API |
| MySQL | 3306 | Database |
| RabbitMQ | 5672 | Message Queue |
| RabbitMQ Management | 15672 | Management UI (guest/guest) |

---

## API Reference

### Public Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/login` | User login (local/LDAP) |

### Authenticated Endpoints (JWT Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/profile` | Get current user profile |
| GET | `/api/dashboard` | Get dashboard statistics |
| GET | `/api/agents` | List all agents |
| GET | `/api/agents/:id` | Get agent details |
| POST | `/api/agents` | Create agent |
| PUT | `/api/agents/:id` | Update agent |
| DELETE | `/api/agents/:id` | Delete agent |
| GET | `/api/conversations` | List conversations |
| POST | `/api/conversations` | Create conversation |
| DELETE | `/api/conversations/:id` | Delete conversation |
| GET | `/api/conversations/:id/messages` | Get conversation messages |
| POST | `/api/conversations/:id/messages` | Send message |
| GET | `/api/skills` | List all skills |
| GET | `/api/agents/:id/skills` | Get agent's linked skills |
| GET | `/api/ai-providers` | List AI model providers |
| PUT | `/api/ai-providers/:id` | Update AI provider config |
| POST | `/api/ai-providers/:id/test` | Test AI provider connection |
| GET | `/api/website-categories` | Get website categories & links |

### Admin Endpoints (Admin Role Required)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create user |
| PUT | `/api/users/:id` | Update user |
| DELETE | `/api/users/:id` | Delete user |
| GET | `/api/ldap-configs` | List LDAP configurations |
| POST | `/api/ldap-configs` | Create LDAP configuration |
| PUT | `/api/ldap-configs/:id` | Update LDAP configuration |
| DELETE | `/api/ldap-configs/:id` | Delete LDAP configuration |
| POST | `/api/ldap-configs/:id/test` | Test LDAP connection |
| GET | `/api/operation-logs` | Get operation logs |

---

## Data Models

### Entity Relationship Diagram

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

### Model Descriptions

| Model | Description |
|-------|-------------|
| **User** | Platform user with username, password, email, role (admin/user), and auth type (local/ldap) |
| **LDAPConfig** | Enterprise LDAP server configuration including host, port, bind DN, base DN, filters, TLS settings |
| **Agent** | AI agent with name, description, system prompt, model selection, temperature, max tokens |
| **Skill** | Agent capability with type (delivery/ops/knowledge), tool definitions, and JSON config |
| **AgentSkill** | Many-to-many join table linking agents to skills |
| **Conversation** | Chat session linking a user to an agent |
| **Message** | Individual message in a conversation with role, content, and token usage |
| **AIProvider** | AI provider configuration with API key, base URL, model, and enabled status |
| **WebsiteCategory** | Website link category with icon and sort order |
| **WebsiteLink** | Individual website link with name, URL, icon under a category |
| **TaskLog** | Async task record for RabbitMQ jobs |
| **OperationLog** | Admin operation audit with module, action, target, and details |

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Backend server port |
| `GIN_MODE` | `debug` | Gin run mode (debug/release) |
| `DB_DRIVER` | `mysql` | Database driver (mysql/sqlite) |
| `DB_HOST` | `mysql` | MySQL host |
| `DB_PORT` | `3306` | MySQL port |
| `DB_USER` | `deliverydesk` | MySQL username |
| `DB_PASSWORD` | `deliverydesk123` | MySQL password |
| `DB_NAME` | `deliverydesk` | Database name |
| `RABBITMQ_HOST` | `rabbitmq` | RabbitMQ host |
| `RABBITMQ_PORT` | `5672` | RabbitMQ port |
| `RABBITMQ_USER` | `guest` | RabbitMQ username |
| `RABBITMQ_PASSWORD` | `guest` | RabbitMQ password |
| `AI_PROVIDER` | `openai` | Default AI provider |
| `AI_API_KEY` | - | AI API key |
| `AI_BASE_URL` | `https://api.openai.com/v1` | AI API base URL |
| `AI_MODEL` | `gpt-4` | Default AI model |
| `JWT_SECRET` | `deliverydesk-secret-key-2024` | JWT signing secret |

---

## Development Guide

### Backend Development

```bash
cd backend

# Local development with SQLite (no MySQL required)
export DB_DRIVER=sqlite
go run ./cmd/server

# Run tests
go test ./...

# Build binary
go build -o deliverydesk ./cmd/server
```

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Start dev server (auto-proxies /api to localhost:8080)
npm run dev

# Build for production
npm run build
```

### Adding a New Page

1. Create page component in `frontend/src/pages/`
2. Register the page in `frontend/src/components/MainLayout.jsx`
3. Add navigation menu item in `frontend/src/components/Sidebar.jsx`

### Adding a New API Endpoint

1. Define data model in `backend/internal/model/models.go`
2. Add handler method in `backend/internal/handler/handlers.go`
3. Register route in `backend/cmd/server/main.go`
4. Add API call function in `frontend/src/services/api.js`

---

## FAQ

### Q: Database connection failed?
Verify MySQL is running. Check `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` environment variables. In Docker Compose mode, the backend automatically waits for MySQL health check to pass before starting.

### Q: Does RabbitMQ connection failure affect functionality?
No. The system is designed to only log a warning when RabbitMQ connection fails. Core functionality continues to work normally. Async task features become available once MQ recovers.

### Q: How to configure LDAP authentication?
1. Login with admin credentials
2. Navigate to "LDAP Configuration" page
3. Click "Add LDAP Server"
4. Fill in LDAP server details (Host, Port, Bind DN, Base DN, etc.)
5. Click "Test Connection" to verify
6. Once enabled, users can select LDAP authentication on the login page

### Q: How to add a new AI model provider?
Navigate to the "AI Models" page. The system comes pre-configured with OpenAI, DeepSeek, Qwen, GLM, SiliconFlow, and more. Simply enter the corresponding API key to enable any provider.

### Q: Can I use SQLite for development?
Yes! Set `DB_DRIVER=sqlite` and the backend will use an embedded SQLite database. This is ideal for local development without needing a MySQL instance.

---

## License

This project is licensed under the [MIT License](LICENSE).

---

<p align="center">
  <strong>DeliveryDesk</strong> - Making cloud delivery smarter and more efficient
</p>
