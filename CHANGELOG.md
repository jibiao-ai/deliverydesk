# Changelog

All notable changes to DeliveryDesk will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

---

## [3.1.0] - 2026-04-09

### Added
- **Feature 1 - 交付技能 (Delivery Skill)**: 技能商店新增「交付技能的skills」技能，基于上传的交付文档构建 RAG 知识库
  - 支持上传 .docx / .xlsx / .txt / .md 文档，自动解析并索引为文本块
  - 基于 TF-IDF 检索 + LLM 评分的 RAG 管线（Retrieve → Score → Filter → Synthesize）
  - 技能商店完整 CRUD 管理界面，支持查看文档列表和索引状态
  - 架构参考 CloudWeGo Eino 框架的 Workflow 模式（load → chunk → score → filter → answer）

- **Feature 2 - 交付专家 Agent (Delivery Expert)**: 发布「交付专家」智能体
  - 关联交付技能、K8S管理技能、OpenStack管理技能
  - 已发布为外部对话接口（Published Agent API）
  - 支持 `POST /api/published-agents/:id/chat` 无需登录即可对话
  - 智能体管理页支持发布/取消发布、查看 API 接口信息

- **Feature 3 - 铁律规则 (Iron Rules)**: Agent 开发铁律模式
  1. 所有指标/标签/数值必须来自技能知识库的具体结果，禁止编造数据
  2. 无技能数据时不推断根因或编造趋势
  3. 阈值必须来自具体技能数据，禁止自定义阈值
  4. 回答必须引用具体技能和环境信息
  5. 数据为空时直接回复「无有效数据，无法判断」
  6. 每个回答给出 1-10 置信度评分，低于 7 分标注 [低置信度警告]
  7. 遵守 token 限制，失败时最多重试 5 次

- **Feature 4 - 社区技能 (Community Skills)**: 拉取社区技能
  - **k8s-operator**: K8S 集群管理技能（集群管理/工作负载/网络/存储/安全/监控/故障排查）
  - **openstack-operator**: OpenStack 云平台管理技能（计算/网络/存储/镜像/认证/编排/EasyStack特有能力）

### Changed
- Skill 模型新增 `category`, `doc_count`, `chunk_count` 字段和 `SkillDocument` 关联
- Agent 模型新增 `is_published`, `iron_rules` 字段
- 技能商店页面从占位符重构为完整管理界面（文档上传、索引状态、重建索引）
- 智能体页面从占位符重构为完整管理界面（CRUD、技能关联、发布管理、API 查看）
- 仪表盘新增技能统计卡片
- AI 对话增加 RAG 管线集成，铁律模式下 AI 请求失败最多重试 5 次

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

[3.1.0]: https://github.com/jibiao-ai/deliverydesk/compare/v2.1.1...v3.1.0
[2.1.1]: https://github.com/jibiao-ai/deliverydesk/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/jibiao-ai/deliverydesk/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.5.0...v2.0.0
[1.5.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/jibiao-ai/deliverydesk/compare/v1.0.0...v1.3.0
[1.0.0]: https://github.com/jibiao-ai/deliverydesk/releases/tag/v1.0.0
