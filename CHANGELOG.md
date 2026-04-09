# Changelog

All notable changes to DeliveryDesk will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

---

## [2.1.1] - 2026-04-09

### Added
- **LDAP 用户 OU 字段**：LDAP 配置新增「用户 OU」字段，可指定只同步特定组织单元的用户（如 `ou=Technology,dc=easystack,dc=cn`），留空则同步 BaseDN 下所有用户
- **LDAP 分页搜索**：使用 LDAP Paging Control 分页拉取，单次同步上限从默认值提升到 1000 个用户
- 前端 LDAP 配置表单新增「用户 OU」输入框，配置列表显示 OU 信息
- 说明卡片更新 OU 用法提示

### Changed
- LDAP 同步日志增加搜索根路径输出，方便排查问题

---

## [2.1.0] - 2026-04-09

### Added
- **真实 LDAP 连接同步**：使用 `go-ldap/ldap/v3` 库替代之前的模拟数据，同步时真实连接 LDAP 服务器拉取用户
- **用户列表服务端分页**：后端 `ListUsers` API 支持 `page`、`page_size`、`search` 参数
- 前端用户管理页新增完整分页控件（每页 10 条，首页/上一页/页码选择/下一页/末页）
- 用户统计数据独立加载，不受分页影响

### Fixed
- 修正「合适费控报销」为「合思费控报销」

---

## [2.0.0] - 2026-04-08

### Added
- **LDAP 用户管理流程优化**：LDAP 用户改为管理员手动同步，不再登录时自动创建
- 新增「同步 LDAP 用户」按钮，管理员可主动从 LDAP 服务器拉取用户
- 新建用户时强制为本地认证（`auth_type=local`），移除 LDAP 选项
- 更新用户时保留原有 `auth_type`，防止篡改
- LDAP 用户编辑时隐藏密码字段并提示「密码由 LDAP 服务器管理」
- 用户管理页底部增加 LDAP 用户管理说明卡片

### Changed
- `loginLDAP` 不再自动创建用户，未同步的 LDAP 用户登录时提示联系管理员
- 用户创建表单移除认证方式选择，统一为「新建本地用户」

---

## [1.5.0] - 2026-04-08

### Added
- **操作日志系统**：记录用户登录、智能体操作、LDAP 管理等所有关键操作
- **用户管理页面**：管理员可创建、编辑、删除用户，支持角色和权限管理
- 操作日志页面支持分页、按模块/操作/用户名筛选
- 用户列表显示角色标签、认证方式、权限范围

---

## [1.4.0] - 2026-04-08

### Added
- **公司系统导航页面**：重新设计的网站导航页，分类展示 32+ 常用工具和系统
- 支持搜索、分类筛选、收藏等功能

---

## [1.3.0] - 2026-04-08

### Fixed
- 修复后端容器 crash-loop（MySQL Error 1064 + collation 优化）
- 修复 MySQL Error 1071（key too long）导致后端无法启动
- 修复后端容器 crash-loop 及 admin 登录 bug - 全面诊断修复
- 彻底修复 admin 登录 bug - 前后端双向修复
- 修复 admin 登录、外网访问和 UI 主配色问题

---

## [1.0.0] - 2026-04-08

### Added
- **初始版本发布**
- Go + Gin 后端 API 服务
- React 18 前端 SPA
- MySQL 8.0 数据存储
- RabbitMQ 异步消息队列
- Docker Compose 一键部署
- JWT 认证与 RBAC 权限控制
- AI 智能体对话系统（多模型支持）
- LDAP/AD 企业认证集成
- 公司系统网站导航
- 仪表盘统计看板
- 技能商店
- AI 模型厂商配置管理

---

[2.1.1]: https://github.com/jibiao-ai/deliverydesk/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/jibiao-ai/deliverydesk/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.5.0...v2.0.0
[1.5.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.0.0...v1.3.0
[1.0.0]: https://github.com/jibiao-ai/deliverydesk/releases/tag/v1.0.0
