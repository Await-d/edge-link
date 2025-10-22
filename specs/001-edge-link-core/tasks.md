---
description: "Edge-Link核心系统的任务列表"
---

# Tasks: Edge-Link核心系统

**输入**: `/home/await/project/edge-link/specs/001-edge-link-core/`目录下的设计文档
**前提条件**: plan.md（必需）, spec.md（必需用于用户故事）, research.md, data-model.md, contracts/

**测试**: 本项目不包含测试任务，专注于核心功能实现。测试将在后续迭代中添加。

**组织**: 任务按用户故事分组，以实现每个故事的独立实现和测试。

## 格式: `[ID] [P?] [Story] 描述`
- **[P]**: 可以并行运行（不同文件，无依赖）
- **[Story]**: 此任务属于哪个用户故事（例如 US1, US2, US3）
- 在描述中包含精确的文件路径

## 路径约定
- **多组件项目**: `backend/`, `frontend/`, `clients/`, `infrastructure/`
- 下面显示的路径基于plan.md结构

---

## Phase 1: Setup（共享基础设施）

**目的**: 项目初始化和基本结构

- [X] T001 创建根目录结构（backend/, frontend/, clients/, infrastructure/, monitoring/）
- [X] T002 [P] 初始化后端Go模块 backend/go.mod with module github.com/edgelink/backend
- [X] T003 [P] 初始化前端项目 frontend/package.json with React 19, Vite 5, TypeScript 5
- [X] T004 [P] 配置Go linting tools in backend/.golangci.yml
- [X] T005 [P] 配置前端linting tools in frontend/.eslintrc.js and frontend/.prettierrc
- [X] T006 [P] 创建Docker Compose开发环境 infrastructure/docker/docker-compose.yml (PostgreSQL, Redis, MinIO)
- [X] T007 [P] 创建.gitignore文件（根目录、backend/、frontend/、clients/）
- [X] T008 [P] 创建README.md with project overview and setup instructions

---

## Phase 2: Foundational（阻塞所有用户故事的前置条件）

**目的**: 在任何用户故事实现之前必须完成的核心基础设施

**⚠️ 关键**: 在此阶段完成之前，不能开始任何用户故事工作

### 数据库基础

- [X] T009 设置数据库迁移框架 backend/internal/migrations/migrate.go using golang-migrate
- [X] T010 创建初始迁移：organizations表 in backend/internal/migrations/000001_create_organizations.up.sql
- [X] T011 创建迁移：virtual_networks表 in backend/internal/migrations/000002_create_virtual_networks.up.sql
- [X] T012 创建迁移：devices表 in backend/internal/migrations/000003_create_devices.up.sql
- [X] T013 创建迁移：device_keys表 in backend/internal/migrations/000004_create_device_keys.up.sql
- [X] T014 创建迁移：pre_shared_keys表 in backend/internal/migrations/000005_create_pre_shared_keys.up.sql
- [X] T015 创建迁移：peer_configurations表 in backend/internal/migrations/000006_create_peer_configurations.up.sql
- [X] T016 创建迁移：sessions表 in backend/internal/migrations/000007_create_sessions.up.sql
- [X] T017 创建迁移：alerts表 in backend/internal/migrations/000008_create_alerts.up.sql
- [X] T018 创建迁移：audit_logs表 in backend/internal/migrations/000009_create_audit_logs.up.sql
- [X] T019 创建迁移：diagnostic_bundles表 in backend/internal/migrations/000010_create_diagnostic_bundles.up.sql
- [X] T020 创建迁移：admin_users表 in backend/internal/migrations/000011_create_admin_users.up.sql

### 共享领域模型

- [X] T021 [P] 创建Organization领域模型 in backend/internal/domain/organization.go
- [X] T022 [P] 创建VirtualNetwork领域模型 in backend/internal/domain/virtual_network.go
- [X] T023 [P] 创建Device领域模型 in backend/internal/domain/device.go
- [X] T024 [P] 创建DeviceKey领域模型 in backend/internal/domain/device_key.go
- [X] T025 [P] 创建PreSharedKey领域模型 in backend/internal/domain/pre_shared_key.go
- [X] T026 [P] 创建PeerConfiguration领域模型 in backend/internal/domain/peer_configuration.go
- [X] T027 [P] 创建Session领域模型 in backend/internal/domain/session.go
- [X] T028 [P] 创建Alert领域模型 in backend/internal/domain/alert.go
- [X] T029 [P] 创建AuditLog领域模型 in backend/internal/domain/audit_log.go
- [X] T030 [P] 创建DiagnosticBundle领域模型 in backend/internal/domain/diagnostic_bundle.go

### 数据库访问层

- [X] T031 [P] 实现Organization仓储 in backend/internal/repository/organization_repo.go with GORM
- [X] T032 [P] 实现VirtualNetwork仓储 in backend/internal/repository/virtual_network_repo.go with GORM
- [X] T033 [P] 实现Device仓储 in backend/internal/repository/device_repo.go with GORM
- [X] T034 [P] 实现DeviceKey仓储 in backend/internal/repository/device_key_repo.go with GORM
- [X] T035 [P] 实现PreSharedKey仓储 in backend/internal/repository/pre_shared_key_repo.go with GORM

### 配置和基础设施

- [X] T036 实现配置管理 in backend/internal/config/config.go (支持环境变量、YAML配置文件)
- [X] T037 实现数据库连接池 in backend/internal/database/db.go (PostgreSQL连接，最大100连接)
- [X] T038 实现Redis缓存客户端 in backend/internal/cache/redis.go (连接池，TTL管理)
- [X] T039 [P] 实现结构化日志 in backend/internal/logger/logger.go using zap (JSON输出，关联ID)
- [X] T040 [P] 实现Prometheus指标 in backend/internal/metrics/metrics.go (HTTP持续时间，活跃设备)

### 认证框架

- [X] T041 实现预共享密钥认证 in backend/internal/auth/psk_auth.go (HMAC-SHA256验证)
- [X] T042 实现设备签名验证 in backend/internal/auth/device_signature.go (Ed25519签名验证)
- [X] T043 [P] 实现JWT令牌管理（管理员） in backend/internal/auth/jwt.go (OIDC/SAML集成占位符)

**检查点**: 基础就绪 - 用户故事实现现在可以并行开始

---

## Phase 3: 用户故事1 - 设备注册和网络连接 (Priority: P1) 🎯 MVP

**目标**: 使设备能够注册、接收虚拟IP并与对等设备建立P2P WireGuard隧道

**独立测试**: 安装桌面客户端，使用PSK注册，验证设备接收虚拟IP并可以ping另一个注册的设备

### US1 核心服务实现

- [X] T044 [P] [US1] 实现密钥生成工具 in backend/internal/crypto/keypair.go (Ed25519密钥对生成)
- [X] T045 [P] [US1] 实现WireGuard配置构建器 in backend/internal/crypto/wireguard_config.go (生成wg-quick格式配置)
- [X] T046 [US1] 实现设备注册服务 in backend/cmd/device-service/internal/service/device_service.go (PSK验证，IP分配，密钥存储)
- [X] T047 [US1] 实现拓扑服务 in backend/cmd/topology-service/internal/service/topology_service.go (虚拟网络管理，对等配置生成)
- [X] T048 [US1] 实现NAT协调器服务 in backend/cmd/nat-coordinator/internal/service/nat_service.go (STUN探测，ICE-lite，TURN分配)

### US1 API端点

- [X] T049 [US1] 实现POST /device/register端点 in backend/cmd/api-gateway/internal/handler/device_handler.go
- [X] T050 [US1] 实现GET /device/{id}/config端点 in backend/cmd/api-gateway/internal/handler/device_handler.go
- [X] T051 [US1] 实现POST /device/{id}/metrics端点 in backend/cmd/api-gateway/internal/handler/device_handler.go

### US1 gRPC服务间通信

- [X] T052 [P] [US1] 定义设备服务Protobuf in backend/pkg/api/device.proto (RegisterDevice, GetDeviceConfig RPCs)
- [X] T053 [P] [US1] 定义拓扑服务Protobuf in backend/pkg/api/topology.proto (AllocateIP, GetPeerConfig RPCs)
- [X] T054 [US1] 生成Protobuf代码 using protoc in backend/pkg/api/generate.sh (运行protoc命令)
- [X] T055 [US1] 实现设备服务gRPC服务器 in backend/cmd/device-service/internal/grpc/device_grpc_server.go
- [X] T056 [US1] 实现拓扑服务gRPC服务器 in backend/cmd/topology-service/internal/grpc/topology_grpc_server.go

### US1 微服务入口点

- [X] T057 [P] [US1] 创建API Gateway主程序 in backend/cmd/api-gateway/main.go (Gin路由器，Fx依赖注入)
- [X] T058 [P] [US1] 创建设备服务主程序 in backend/cmd/device-service/main.go (gRPC服务器，数据库连接)
- [X] T059 [P] [US1] 创建拓扑服务主程序 in backend/cmd/topology-service/main.go (gRPC服务器，IP分配逻辑)
- [X] T060 [P] [US1] 创建NAT协调器主程序 in backend/cmd/nat-coordinator/main.go (STUN客户端，TURN管理)

### US1 桌面客户端（Linux MVP）

- [X] T061 [US1] 初始化桌面客户端Go模块 clients/desktop/go.mod
- [X] T062 [P] [US1] 实现WireGuard接口管理 in clients/desktop/internal/wireguard/interface.go (检测内核模块 vs wireguard-go)
- [X] T063 [P] [US1] 实现STUN客户端 in clients/desktop/internal/stun/stun_client.go (NAT类型检测)
- [X] T064 [P] [US1] 实现配置存储 in clients/desktop/internal/config/config_store.go (加密本地配置，AES-256)
- [X] T065 [P] [US1] 实现指标报告器 in clients/desktop/internal/metrics/reporter.go (心跳，带宽，延迟)
- [X] T066 [US1] 实现设备注册CLI命令 in clients/desktop/cmd/edgelink-cli/register.go (PSK输入，注册API调用)
- [X] T067 [US1] 实现守护进程 in clients/desktop/cmd/edgelink-daemon/main.go (WireGuard监控，自动重连)
- [X] T068 [US1] 实现平台特定代码（Linux） in clients/desktop/internal/platform/linux.go (TUN接口创建，需要root权限)

### US1 Docker镜像

- [X] T069 [P] [US1] 创建API Gateway Dockerfile in infrastructure/docker/Dockerfile.api-gateway
- [X] T070 [P] [US1] 创建设备服务Dockerfile in infrastructure/docker/Dockerfile.device-service
- [X] T071 [P] [US1] 创建拓扑服务Dockerfile in infrastructure/docker/Dockerfile.topology-service
- [X] T072 [P] [US1] 创建NAT协调器Dockerfile in infrastructure/docker/Dockerfile.nat-coordinator

### US1 Kubernetes部署

- [X] T073 [US1] 创建Helm Chart结构 in infrastructure/helm/edge-link-control-plane/Chart.yaml
- [X] T074 [US1] 创建Helm values.yaml in infrastructure/helm/edge-link-control-plane/values.yaml (默认配置)
- [X] T075 [P] [US1] 创建API Gateway Deployment模板 in infrastructure/helm/edge-link-control-plane/templates/api-gateway-deployment.yaml
- [X] T076 [P] [US1] 创建设备服务Deployment模板 in infrastructure/helm/edge-link-control-plane/templates/device-service-deployment.yaml
- [X] T077 [P] [US1] 创建拓扑服务Deployment模板 in infrastructure/helm/edge-link-control-plane/templates/topology-service-deployment.yaml
- [X] T078 [P] [US1] 创建NAT协调器Deployment模板 in infrastructure/helm/edge-link-control-plane/templates/nat-coordinator-deployment.yaml
- [X] T079 [P] [US1] 创建Service模板 in infrastructure/helm/edge-link-control-plane/templates/services.yaml
- [X] T080 [US1] 创建Ingress模板 in infrastructure/helm/edge-link-control-plane/templates/ingress.yaml (NGINX ingress，TLS)

### US1 集成和验证

- [X] T081 [US1] 创建本地开发脚本 in scripts/dev-setup.sh (启动Docker Compose，运行迁移)
- [X] T082 [US1] 创建种子数据脚本 in scripts/seed-data.sh (创建测试组织，虚拟网络，PSK)
- [X] T083 [US1] 验证完整注册流程（测试指南：scripts/TESTING.md）

**检查点**: 此时，用户故事1应完全功能正常且可独立测试

---

## Phase 4: 用户故事2 - 网络管理和监控 (Priority: P2)

**目标**: 为管理员提供Web UI以查看设备状态、监控连接健康和管理设备生命周期

**独立测试**: 管理员登录Web UI，查看仪表板指标，导航到设备详情，执行一个管理操作（例如撤销设备）

### US2 补充仓储

- [X] T084 [P] [US2] 实现Session仓储 in backend/internal/repository/session_repo.go with GORM
- [X] T085 [P] [US2] 实现Alert仓储 in backend/internal/repository/alert_repo.go with GORM
- [X] T086 [P] [US2] 实现AuditLog仓储 in backend/internal/repository/audit_log_repo.go with GORM (仅插入，不可变)
- [X] T087 [P] [US2] 实现AdminUser仓储 in backend/internal/repository/admin_user_repo.go with GORM in backend/internal/repository/admin_user_repo.go with GORM

### US2 管理API端点

- [X] T088 [P] [US2] 实现GET /devices端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go (过滤，分页)
- [X] T089 [P] [US2] 实现GET /virtual-networks端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go
- [X] T090 [P] [US2] 实现POST /virtual-networks端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go
- [X] T091 [P] [US2] 实现DELETE /device/{id}端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go (撤销设备)
- [X] T092 [P] [US2] 实现GET /alerts端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go
- [X] T093 [P] [US2] 实现POST /alerts/{id}/acknowledge端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go
- [X] T094 [P] [US2] 实现GET /audit-logs端点 in backend/cmd/api-gateway/internal/handler/admin_handler.go in backend/cmd/api-gateway/internal/handler/audit_handler.go

### US2 WebSocket实时更新

- [X] T095 [US2] 实现WebSocket处理器 in backend/cmd/api-gateway/internal/websocket/ws_handler.go (gorilla/websocket)
- [X] T096 [US2] 实现WebSocket订阅管理 in backend/cmd/api-gateway/internal/websocket/subscription.go (按事件类型、设备ID过滤)
- [X] T097 [US2] 实现WebSocket事件广播 in backend/cmd/api-gateway/internal/websocket/broadcaster.go (Redis Pub/Sub用于多实例协调)
- [X] T098 [US2] 实现心跳ping/pong in backend/cmd/api-gateway/internal/websocket/heartbeat.go (30s间隔)

### US2 审计日志记录

- [X] T099 [US2] 实现审计日志中间件 in backend/internal/audit/middleware.go (拦截所有管理操作，记录前/后状态)
- [X] T100 [US2] 将审计中间件集成到API Gateway in backend/cmd/api-gateway/main.go

### US2 前端项目设置

- [X] T101 [US2] 配置Vite构建 in frontend/vite.config.ts (React插件，TypeScript路径)
- [X] T102 [US2] 配置TypeScript in frontend/tsconfig.json (严格模式，路径别名)
- [X] T103 [P] [US2] 安装核心依赖 in frontend/package.json (React 19, Ant Design 5, ECharts 5, TanStack Query, Zustand)
- [X] T104 [P] [US2] 设置路由 in frontend/src/App.tsx (React Router v6)
- [X] T105 [P] [US2] 创建API客户端 in frontend/src/services/api-client.ts (Axios，Bearer令牌认证)
- [X] T106 [P] [US2] 创建WebSocket客户端 in frontend/src/services/websocket-client.ts (重连逻辑，事件处理器)

### US2 状态管理

- [X] T107 [P] [US2] 创建Zustand UI状态store in frontend/src/stores/ui-store.ts (主题，侧边栏折叠，选定设备ID)
- [X] T108 [P] [US2] 配置TanStack Query in frontend/src/services/query-client.ts (缓存，失效策略)

### US2 核心UI组件

- [X] T109 [P] [US2] 创建布局组件 in frontend/src/components/Layout.tsx (Ant Design Layout，侧边栏，页眉)
- [X] T110 [P] [US2] 创建仪表板页面 in frontend/src/pages/Dashboard.tsx (指标卡片，ECharts图表)
- [X] T111 [P] [US2] 创建设备列表页面 in frontend/src/pages/DeviceList.tsx (Ant Design Table，过滤，分页)
- [X] T112 [P] [US2] 创建设备详情页面 in frontend/src/pages/DeviceDetails.tsx (虚拟IP，NAT类型，会话历史)
- [X] T113 [P] [US2] 创建网络拓扑页面 in frontend/src/pages/Topology.tsx (ECharts图形，设备节点，隧道边缘)
- [X] T114 [P] [US2] 创建告警页面 in frontend/src/pages/Alerts.tsx (时间轴，严重程度过滤，确认操作)
- [X] T115 [P] [US2] 创建审计日志页面 in frontend/src/pages/AuditLogs.tsx (时间轴，资源类型过滤，前/后状态diff)

### US2 实时更新集成

- [X] T116 [US2] 将WebSocket集成到仪表板 in frontend/src/pages/Dashboard.tsx (订阅METRICS_SUMMARY事件)
- [X] T117 [US2] 将WebSocket集成到设备列表 in frontend/src/pages/DeviceList.tsx (订阅DEVICE_STATUS_CHANGE事件)
- [X] T118 [US2] 将WebSocket集成到告警 in frontend/src/pages/Alerts.tsx (订阅ALERT_CREATED，ALERT_UPDATED事件)
- [X] T119 [US2] 创建前端Dockerfile in infrastructure/docker/Dockerfile.frontend (多阶段构建：npm build + nginx服务)
- [X] T120 [US2] 创建nginx配置 in infrastructure/docker/nginx.conf (SPA路由回退，API代理)
- [X] T121 [US2] 更新Helm chart以包含前端 in infrastructure/helm/edge-link-control-plane/templates/frontend-deployment.yaml

**检查点**: 此时，用户故事1和2应都能独立工作

---

## Phase 5: 用户故事3 - 自动化健康监控和告警 (Priority: P3)

**目标**: 自动检测设备健康问题并通过配置的通道发送告警

**独立测试**: 模拟故障条件（断开设备连接），验证生成告警并通过电子邮件/webhook发送

### US3 告警服务实现

- [X] T122 [US3] 创建告警服务主程序 in backend/cmd/alert-service/main.go (监控Redis流/PostgreSQL事件)
- [X] T123 [P] [US3] 实现阈值检查器 in backend/cmd/alert-service/internal/checker/threshold_checker.go (离线>5分钟，延迟>500ms)
- [X] T124 [P] [US3] 实现告警生成器 in backend/cmd/alert-service/internal/generator/alert_generator.go (创建Alert实体，设置严重程度)
- [X] T125 [P] [US3] 实现电子邮件通知器 in backend/cmd/alert-service/internal/notifier/email_notifier.go (SMTP集成，HTML模板)
- [X] T126 [P] [US3] 实现Webhook通知器 in backend/cmd/alert-service/internal/notifier/webhook_notifier.go (HTTP POST，重试逻辑)
- [X] T127 [US3] 实现通知调度器 in backend/cmd/alert-service/internal/scheduler/notification_scheduler.go (优先级队列，速率限制)

### US3 后台工作器

- [X] T128 [US3] 创建后台工作器主程序 in backend/cmd/background-worker/main.go (定时任务调度器)
- [X] T129 [P] [US3] 实现设备健康检查任务 in backend/cmd/background-worker/internal/tasks/device_health.go (检测离线设备，每分钟运行)
- [X] T130 [P] [US3] 实现性能监控任务 in backend/cmd/background-worker/internal/tasks/performance_monitor.go (检查延迟p95，每5分钟运行)
- [X] T131 [P] [US3] 实现安全监控任务 in backend/cmd/background-worker/internal/tasks/security_monitor.go (检测失败认证尝试，每分钟运行)
- [X] T132 [P] [US3] 实现密钥过期检查任务 in backend/cmd/background-worker/internal/tasks/key_expiry.go (提前30天警告，每天运行)

### US3 部署更新

- [X] T133 [P] [US3] 创建告警服务Dockerfile in infrastructure/docker/Dockerfile.alert-service
- [X] T134 [P] [US3] 创建后台工作器Dockerfile in infrastructure/docker/Dockerfile.background-worker
- [X] T135 [P] [US3] 更新Helm chart以包含告警服务 in infrastructure/helm/edge-link-control-plane/templates/alert-service-deployment.yaml
- [X] T136 [P] [US3] 更新Helm chart以包含后台工作器 in infrastructure/helm/edge-link-control-plane/templates/background-worker-deployment.yaml

**检查点**: 所有用户故事现在应独立功能正常

---

## Phase 6: 用户故事4 - 跨平台客户端支持 (Priority: P4)

**目标**: 在多个平台（Windows, macOS, Android, iOS）上提供客户端

**独立测试**: 在3个以上不同平台上安装客户端，使用相同虚拟网络注册，验证所有设备可以相互ping

**⚠️ v1.0 Scope Note**: 移动客户端任务 (T141-T150) 已延期至 v2.0。US4 v1.0 范围仅包含 Windows、macOS 和 IoT 客户端。详见 spec.md FR-002。

### US4 Windows桌面客户端

- [X] T137 [P] [US4] 实现平台特定代码（Windows） in clients/desktop/internal/platform/windows.go (Wintun集成，wireguard-go)
- [ ] T138 [US4] 创建Windows安装程序构建脚本 in clients/desktop/build/windows/build-installer.ps1 (WiX Toolset)

### US4 macOS桌面客户端

- [X] T139 [P] [US4] 实现平台特定代码（macOS） in clients/desktop/internal/platform/macos.go (wireguard-go，权限提升)
- [ ] T140 [US4] 创建macOS .app捆绑包 in clients/desktop/build/macos/build-app.sh (代码签名，公证)

### US4 iOS移动应用

- [ ] T141 [US4] 初始化iOS Xcode项目 in clients/mobile/ios/EdgeLink.xcodeproj
- [ ] T142 [P] [US4] 实现WireGuard集成 in clients/mobile/ios/EdgeLink/Services/WireGuardService.swift (WireGuardKit SDK)
- [ ] T143 [P] [US4] 实现设备注册 in clients/mobile/ios/EdgeLink/ViewModels/RegistrationViewModel.swift (API客户端，PSK输入)
- [ ] T144 [P] [US4] 创建SwiftUI主视图 in clients/mobile/ios/EdgeLink/Views/MainView.swift (连接状态，对等列表)
- [ ] T145 [P] [US4] 创建SwiftUI设置视图 in clients/mobile/ios/EdgeLink/Views/SettingsView.swift (服务器配置，PSK输入)

### US4 Android移动应用

- [ ] T146 [US4] 初始化Android Studio项目 in clients/mobile/android/app/build.gradle
- [ ] T147 [P] [US4] 实现WireGuard集成 in clients/mobile/android/app/src/main/java/com/edgelink/services/WireGuardService.kt (wireguard-android库)
- [ ] T148 [P] [US4] 实现设备注册 in clients/mobile/android/app/src/main/java/com/edgelink/viewmodels/RegistrationViewModel.kt (Retrofit API客户端)
- [ ] T149 [P] [US4] 创建Jetpack Compose主UI in clients/mobile/android/app/src/main/java/com/edgelink/ui/MainScreen.kt (连接状态，对等列表)
- [ ] T150 [P] [US4] 创建Jetpack Compose设置UI in clients/mobile/android/app/src/main/java/com/edgelink/ui/SettingsScreen.kt (服务器配置)

### US4 IoT/容器客户端

- [X] T151 [P] [US4] 创建轻量级CLI客户端 in clients/desktop/cmd/edgelink-lite/main.go (最小依赖，<10MB二进制)
- [X] T152 [P] [US4] 创建Docker sidecar镜像 in infrastructure/docker/Dockerfile.edgelink-sidecar (为容器工作负载)
- [X] T153 [US4] 创建Kubernetes DaemonSet示例 in infrastructure/helm/edgelink-sidecar/templates/daemonset.yaml

**检查点**: 所有用户故事和平台现在应独立功能正常

---

## Phase 7: 完善和跨领域关注点

**目的**: 影响多个用户故事的改进

### 监控和可观测性

- [X] T154 [P] 创建Prometheus配置 in monitoring/prometheus/prometheus.yml (抓取所有控制平面服务)
- [X] T155 [P] 创建Prometheus告警规则 in monitoring/prometheus/alerts.yml (高错误率，低p95延迟)
- [X] T156 [P] 创建Grafana仪表板：控制平面概览 in monitoring/grafana/dashboards/control-plane-overview.json
- [X] T157 [P] 创建Grafana仪表板：设备健康 in monitoring/grafana/dashboards/device-health.json
- [X] T158 [P] 创建Grafana仪表板：隧道指标 in monitoring/grafana/dashboards/tunnel-metrics.json
- [X] T159 [P] 配置Loki日志聚合 in monitoring/loki/loki-config.yml

### CI/CD管道

- [X] T160 [P] 创建GitHub Actions工作流：后端 in .github/workflows/backend.yml (Go test, build, Docker push)
- [X] T161 [P] 创建GitHub Actions工作流：前端 in .github/workflows/frontend.yml (npm test, build, Docker push)
- [X] T162 [P] 创建GitHub Actions工作流：桌面客户端 in .github/workflows/desktop-client.yml (跨平台构建)

### 文档

- [X] T163 [P] 更新根README in README.md (项目概述，快速入门，架构图)
- [X] T164 [P] 创建API文档生成脚本 in scripts/generate-api-docs.sh (从OpenAPI生成Stoplight文档)
- [X] T165 [P] 创建部署指南 in docs/deployment.md (Kubernetes，Helm，环境变量)
- [X] T166 [P] 创建故障排查指南 in docs/troubleshooting.md (常见问题，诊断步骤)

### 性能优化

- [X] T167 [P] 实现数据库连接池调优 in backend/internal/database/db.go (基于负载测试结果)
- [X] T168 [P] 实现Redis缓存策略 in backend/internal/cache/redis.go (设备在线状态，对等配置TTL)
- [X] T169 [P] 在API Gateway中实现速率限制 in backend/cmd/api-gateway/internal/middleware/rate_limit.go (每组织1000请求/分钟)

### 安全加固

- [X] T170 [P] 实现CORS配置 in backend/cmd/api-gateway/main.go (仅允许管理UI域)
- [X] T171 [P] 实现请求验证中间件 in backend/internal/middleware/validation.go (输入清理，SQL注入预防)
- [X] T172 [P] 配置TLS证书管理 in infrastructure/helm/edge-link-control-plane/templates/cert-manager.yaml (Let's Encrypt集成)

### 清理和验证

- [X] T173 验证quickstart.md中的所有步骤（按照部署指南进行端到端验证）
- [X] T174 验证所有API端点与OpenAPI规范匹配 in contracts/control-plane-api-v1.yaml
- [X] T175 验证所有WebSocket事件类型已实现 in contracts/websocket-events.md
- [X] T176 验证指标架构符合客户端实现 in contracts/client-metrics-schema.json

---

## 依赖关系和执行顺序

### 阶段依赖关系

- **Setup (Phase 1)**: 无依赖 - 可以立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - 阻塞所有用户故事
- **用户故事 (Phase 3+)**: 所有依赖Foundational阶段完成
  - 用户故事可以并行进行（如果有人员配备）
  - 或按优先级顺序依次进行（P1 → P2 → P3 → P4）
- **完善 (最终阶段)**: 依赖所有所需用户故事完成

### 用户故事依赖关系

- **用户故事1 (P1)**: Foundational (Phase 2)之后可以开始 - 对其他故事无依赖
- **用户故事2 (P2)**: Foundational (Phase 2)之后可以开始 - 可能与US1集成但应独立可测试
- **用户故事3 (P3)**: Foundational (Phase 2)之后可以开始 - 可能与US1/US2集成但应独立可测试
- **用户故事4 (P4)**: Foundational (Phase 2)之后可以开始 - 构建在US1之上但应独立可测试

### 每个用户故事内部

- 核心服务实现 → API端点
- 领域模型 → 仓储 → 服务
- 后端API → 前端集成
- 主要实现完成后再进行跨故事集成

### 并行机会

- 所有标记[P]的Setup任务可以并行运行
- 所有Foundational任务标记[P]可以在Phase 2内并行运行
- Foundational阶段完成后，所有用户故事可以并行启动（如果团队容量允许）
- 故事内标记[P]的所有任务可以并行运行
- 不同用户故事可以由不同团队成员并行处理

---

## Phase 3并行示例: 用户故事1

```bash
# 同时启动用户故事1的所有核心服务：
Task: "实现密钥生成工具 in backend/internal/crypto/keypair.go"
Task: "实现WireGuard配置构建器 in backend/internal/crypto/wireguard_config.go"

# 同时启动用户故事1的所有Protobuf定义：
Task: "定义设备服务Protobuf in backend/pkg/api/device.proto"
Task: "定义拓扑服务Protobuf in backend/pkg/api/topology.proto"

# 同时启动用户故事1的所有微服务入口点：
Task: "创建API Gateway主程序 in backend/cmd/api-gateway/main.go"
Task: "创建设备服务主程序 in backend/cmd/device-service/main.go"
Task: "创建拓扑服务主程序 in backend/cmd/topology-service/main.go"
Task: "创建NAT协调器主程序 in backend/cmd/nat-coordinator/main.go"
```

---

## 实施策略

### MVP优先（仅用户故事1）

1. 完成Phase 1: Setup
2. 完成Phase 2: Foundational (关键 - 阻塞所有故事)
3. 完成Phase 3: 用户故事1
4. **停止并验证**: 独立测试用户故事1
5. 如果准备就绪可以部署/演示

### 增量交付

1. 完成Setup + Foundational → 基础就绪
2. 添加用户故事1 → 独立测试 → 部署/演示 (MVP!)
3. 添加用户故事2 → 独立测试 → 部署/演示
4. 添加用户故事3 → 独立测试 → 部署/演示
5. 添加用户故事4 → 独立测试 → 部署/演示
6. 每个故事在不破坏先前故事的情况下增加价值

### 并行团队策略

有多个开发人员时：

1. 团队共同完成Setup + Foundational
2. Foundational完成后：
   - 开发者A: 用户故事1
   - 开发者B: 用户故事2
   - 开发者C: 用户故事3
3. 故事独立完成和集成

---

## 注意事项

- [P] 任务 = 不同文件，无依赖
- [Story] 标签将任务映射到特定用户故事以便追溯
- 每个用户故事应独立可完成和可测试
- 在每个检查点停止以独立验证故事
- 在逻辑任务或任务组后提交
- 避免：模糊任务，同一文件冲突，破坏独立性的跨故事依赖

---

## 任务统计

- **总任务数**: 176
- **Setup阶段**: 8个任务
- **Foundational阶段**: 35个任务
- **用户故事1 (P1)**: 40个任务 (MVP核心)
- **用户故事2 (P2)**: 38个任务
- **用户故事3 (P3)**: 15个任务
- **用户故事4 (P4)**: 17个任务
- **完善阶段**: 23个任务

**并行机会**: 约60%的任务标记为[P]可以并行执行

**建议的MVP范围**: Phase 1 + Phase 2 + Phase 3 (用户故事1) = 83个任务
