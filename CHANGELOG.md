# Changelog

All notable changes to DeliveryDesk will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

---

## [3.2.2] - 2026-04-09

### Fixed
- **重大 Bug: docker-compose up -d 后 VM 不可达**:
  - **根因**: Docker bridge 网络默认尝试启用 IPv6，触发内核 `ADDRCONF(NETDEV_UP): br-xxx: link is not ready` 错误。
    IPv6 网桥初始化失败导致宿主机网络接口状态异常，SSH 连接断开，外部无法访问 VM。
  - **修复**: `docker-compose.yml` 中 bridge 网络显式设置 `enable_ipv6: false`，配置固定子网 `172.28.0.0/16`，
    启用 `ip_masquerade`，避免 IPv6 初始化对宿主机网络的干扰。
- **端口冲突风险**: MySQL (3306)、RabbitMQ (5672/15672)、Backend (8080) 端口不再暴露到宿主机，
  仅通过 Docker 内部网络通信。只有 Frontend 的 80 端口对外暴露。减少与 VM 已有服务的端口冲突。
- **Nginx SSE 流式响应被缓冲**: 原 nginx.conf 缺少 `proxy_buffering off` 和 `X-Accel-Buffering no`，
  导致 AI 流式回复被 nginx 缓冲，前端看不到实时 token 推送。修复后流式对话正常工作。
- **Nginx 超时过短**: LDAP 同步和 AI 长对话可能超过原 120s 超时。`proxy_read_timeout` 提升至 300s，
  `proxy_send_timeout` 提升至 120s。

---

## [3.2.1] - 2026-04-09

### Fixed
- **重大 Bug 排查: LDAP 仅同步 108 个用户**:
  - **根因 1 - LDAP 服务器 SizeLimit**: `SearchWithPaging(500)` 可能超出 LDAP 服务器的 sizelimit 限制，
    导致返回 SizeLimitExceeded 错误。修复：改为渐进式分页策略，依次尝试 200 → 100 → 50 的页大小，
    每次失败后重新建立连接再尝试更小的页大小，最终降级到 plain Search 获取部分结果。
  - **根因 2 - 软删除记录冲突**: GORM 软删除导致数据库中存在 `deleted_at IS NOT NULL` 的幽灵记录，
    unique index 仍然阻止同名用户创建。修复：同步前先检查并清理同名的软删除记录。
  - **根因 3 - 错误检测过于宽泛**: 原代码使用 `strings.Contains(err, "4")` 检测 SizeLimitExceeded，
    可能误匹配其他错误。修复：改用 `ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded)` 精确检测。
  - **根因 4 - 本地用户名冲突未计数**: 与本地用户名冲突的 LDAP 用户被静默跳过，现在增加 `skipped_local_conflict`
    计数器并返回给前端。

### Added
- **LDAP 诊断端点**: 新增 `GET /api/ldap-configs/:id/diagnose` 接口，管理员可对每个 LDAP 配置执行完整诊断，
  返回连接、绑定、搜索、数据库状态的逐步检查结果，包含 LDAP 搜索条目数、空用户名数、数据库差距分析和改进建议。
- **LDAP 诊断界面**: LDAP 配置页面每个配置项新增「诊断同步」按钮（放大镜图标），点击弹出详细诊断报告对话框，
  显示连接状态、搜索结果、样本用户名、数据库同步差距和改进建议。
- **同步结果增强**: 用户管理页同步 LDAP 后的 toast 通知现包含失败数、冲突跳过数等详细信息，
  诊断日志输出到浏览器 console 供管理员排查。
- **每个创建失败的用户名记录到诊断详情**: 方便管理员定位具体哪些用户同步失败及原因。

---

## [3.2.0] - 2026-04-09

### Added
- **Feature 5 - 多智能体即时对话 (Multi-Agent Concurrent Chat)**:
  - 全新多标签页聊天界面，支持同时打开多个智能体对话，每个对话独立运行
  - 标签页支持打开、切换、关闭，每个标签独立管理消息状态
  - 正在回复的标签页显示动态闪烁图标，方便识别活跃对话

- **流式回复 (Streaming Response)**:
  - 后端新增 `POST /api/conversations/:id/messages/stream` SSE 流式端点
  - AI 回复逐 token 推送到前端，实时渲染打字效果
  - 后端使用 OpenAI `stream=true` 模式，解析 SSE 数据流并转发给客户端
  - 前端使用 Fetch API + ReadableStream 接收并实时渲染

- **中断回复 (Abort / Stop Generation)**:
  - 智能体回复期间，发送按钮自动变为红色「中断」按钮（带方块图标）
  - 点击「中断」立即停止回复，前端 abort fetch + 后端 cancel context 双重中断
  - 后端新增 `POST /api/conversations/:id/abort` 端点，取消服务端活跃流
  - 中断后已接收的部分回复内容自动保存，标记 `[回复已中断]`

- **后端流式追踪架构**:
  - `chat_service.go` 新增 `activeStreams` map 追踪每个对话的 `context.CancelFunc`
  - `RegisterStream` / `UnregisterStream` / `AbortStream` 导出函数
  - `SendMessageStream()` 方法支持 context 取消传播
  - `streamAIResponse()` 使用 `http.NewRequestWithContext()` 确保 HTTP 请求随 context 取消
  - 超时时间从 120s 提升到 180s 以支持长回复

### Changed
- 聊天页面完全重写为多标签架构，支持侧栏折叠
- 输入框改为可自动扩展的 textarea，支持 Shift+Enter 换行
- 消息气泡样式优化，添加圆角和阴影效果
- 空状态页展示功能亮点（多标签对话 / 流式回复 / 随时中断）

---

## [3.1.1] - 2026-04-09

### Fixed
- **Bug: LDAP Bind Password 首次创建无法保存**: `LDAPConfig.BindPassword` 字段使用了 `json:"-"` 标签，
  导致 Go JSON 反序列化时完全忽略该字段，首次创建 LDAP 配置时密码为空，测试连接必然报错
  `"Empty password not allowed by the client"`。修复方案：`CreateLDAPConfig` 和 `UpdateLDAPConfig`
  处理函数改用显式 struct 接收请求体，确保 `bind_password` 字段正确反序列化并存入数据库。
- **Bug: LDAP 用户 OU 仅支持单个**: 原实现只支持指定单个 OU 进行同步。现支持用 `|` 分隔多个 OU，
  例如 `ou=Tech,dc=xx,dc=cn|ou=Sales,dc=xx,dc=cn`。同步时会依次搜索每个 OU，自动去重跨 OU 的重复用户名。
- **重大 Bug: 用户管理仅显示 109 个用户**: 原因为前端统计接口使用 `page_size=9999` 请求所有用户计算
  统计数据，但后端 `ListUsers` 将 `page_size > 100` 强制归为 10，导致统计数据不准确。修复方案：
  新增专用 `GET /api/users/stats` 端点，直接通过 SQL COUNT 查询获取用户统计（总数、管理员、普通用户、
  LDAP 用户），不受分页限制。同时 LDAP 同步分页大小从 500 提升至 1000 以支持更多用户。
- **前端 LDAPPage**: 更新 OU 输入框的 placeholder 和说明卡片，展示多 OU 分隔符 `|` 用法。
- **前端 UsersPage**: 使用专用 `/users/stats` 端点获取用户统计，不再依赖 page_size hack。

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
