# DeliveryDesk - Cloud Delivery Workbench

DeliveryDesk is an intelligent cloud delivery service workbench built with Go, React, MySQL, and RabbitMQ. It provides a unified platform for delivery teams with AI-powered agents, enterprise LDAP authentication, and quick access to commonly used tools and websites.

## Features

### 1. LDAP Enterprise Authentication
- Admin can configure enterprise LDAP/Active Directory servers via web UI
- Supports multiple LDAP sources with default server selection
- Users can choose between local or LDAP authentication at login
- Auto-provisioning: first-time LDAP users get accounts created automatically

### 2. AI Agent System
- Configure multiple AI model providers (OpenAI, DeepSeek, Qwen, GLM, SiliconFlow, etc.)
- Create and manage AI agents with customizable system prompts
- Skill system: delivery knowledge base, deployment guides, ops skills
- Built-in agents: Delivery Directory Agent, Delivery Skills Agent
- Real-time chat with AI agents

### 3. Website Navigation Hub
- All commonly used websites organized by category
- Quick one-click access to tools, documentation, and systems
- Categories: Common Tools, Network Config, Cloud Products, Deployment Manuals, Operations, etc.
- Search functionality across all links
- Data sourced from the delivery team's website collection

### 4. User Management
- Role-based access control (admin/user)
- Admin dashboard for user CRUD operations
- Operation logs for audit trail

## Tech Stack

- **Backend**: Go (Gin framework), GORM ORM
- **Frontend**: React 18, Vite, Tailwind CSS, Zustand
- **Database**: MySQL 8.0
- **Message Queue**: RabbitMQ
- **Auth**: JWT + LDAP

## Quick Start

### Using Docker Compose

```bash
# Clone the repository
git clone https://github.com/jibiao-ai/deliverydesk.git
cd deliverydesk

# Copy environment config
cp .env.example .env
# Edit .env with your AI API key

# Start all services
docker-compose up -d

# Access the application
# Frontend: http://localhost
# RabbitMQ Management: http://localhost:15672
```

### Development Mode

```bash
# Backend
cd backend
export DB_DRIVER=sqlite
go run ./cmd/server

# Frontend
cd frontend
npm install
npm run dev
```

### Default Credentials

- **Username**: admin
- **Password**: Admin@2024!

## Project Structure

```
deliverydesk/
├── backend/
│   ├── cmd/server/main.go       # Entry point
│   ├── internal/
│   │   ├── config/              # Configuration
│   │   ├── handler/             # HTTP handlers
│   │   ├── middleware/          # Auth middleware
│   │   ├── model/               # Data models
│   │   ├── mq/                  # RabbitMQ
│   │   ├── repository/          # Database & seed data
│   │   └── service/             # Business logic
│   └── pkg/                     # Shared packages
├── frontend/
│   ├── src/
│   │   ├── components/          # Layout components
│   │   ├── pages/               # Page components
│   │   ├── services/            # API client
│   │   ├── store/               # State management
│   │   └── styles/              # Global styles
│   └── public/                  # Static assets
├── docker-compose.yml
└── README.md
```

## UI Style

The UI design follows the OpsGenie AI platform style with:
- Clean sidebar navigation with collapsible groups
- Blue primary color theme (#2563eb)
- Card-based dashboard layout
- Responsive design with Tailwind CSS
- Light/Dark theme support

## License

MIT
