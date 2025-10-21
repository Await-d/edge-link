# 端到端直连系统方案设计

## 架构蓝图

- **总体结构**：客户端(TUN/虚拟网卡) ↔ 控制平面(REST/WebSocket、STUN协程、任务调度、PostgreSQL/Redis) ↔ 管理界面SPA；数据面基于WireGuard隧道直连或回退中继
- **通信流程**：客户端启动→输入服务端地址/交互密钥→生成密钥对→注册与配置下发→NAT探测/打洞→隧道建立→定期心跳与指标上报→控制面监控与告警
- **技术栈**：Go(gin/fx、gRPC)、PostgreSQL/Redis、WireGuard userspace/内核、STUN/TURN服务、前端React19 + Ant Design5 + ECharts

## 客户端设计

- **平台封装**：
- 桌面(Linux/Win/mac)：Go + CGO 调用 wireguard-go 或原生驱动；打包成GUI/CLI双形态
- 移动(Android/iOS)：集成 WireGuard 官方 SDK，统一业务逻辑层使用 Go Mobile/Swift/Kotlin 封装
- 物联网/容器：提供精简CLI/Daemon，可运行在ARM/嵌入式
- **配置流程**：
- 首次启动输入服务端地址与预共享交互密钥
- 生成设备密钥对，调用 `/api/v1/device/register` 提交公钥、设备指纹、网络能力
- 获取虚拟IP、子网、对等端列表、打洞服务器(STUN/TURN)配置
- 写入本地配置(加密保存)，支持一键导入导出
- **运行模块**：
- 守护进程：监控WireGuard接口、自动重连、密钥轮换提醒
- 打洞器：STUN检测NAT类型，UDP打洞；失败时降级使用TURN
- 遥测上报：周期性发送心跳、带宽、延迟、失败统计到 `/api/v1/device/metrics`
- 日志与诊断：统一日志格式，支持推送日志包给控制面或本地导出

## 控制平面/服务端

- **API网关层**：
- gRPC + gRPC-Gateway/REST，统一鉴权使用服务端生成的交互密钥 + 设备密钥签名
- 支持WebSocket推送实时状态与告警
- **业务服务**：
- 设备管理服务：处理注册、配置发放、密钥轮换、状态更新
- 拓扑/路由服务：维护虚拟子网/IP分配、生成对等端策略路由
- 打洞协调服务：STUN/NAT探测、P2P候选信息交换、TURN中继管理
- 审计与告警服务：记录所有控制操作、阈值触发告警、发送邮件/企业微信
- **数据层**：
- PostgreSQL：实体模型(组织、设备、密钥、虚拟网络、会话、告警、审计日志)
- Redis：存储在线设备状态、临时令牌、心跳信息、速率限制
- 对象存储：保存客户端上传的诊断包、历史报告
- **后台任务**：
- 定时清理失活设备、过期密钥、失效告警
- 密钥轮换计划(预警 → 生效 → 回滚)
- 定期NAT能力测试与中继节点健康检查
- **部署与可扩展性**：
- 模块化微服务，可容器化部署至K8s；使用Helm或Terraform管理
- 控制面高可用：API节点+协程服务多副本，Redis哨兵，PostgreSQL主从/云托管

## 管理界面

- **总体结构**：React 19 + Vite + TypeScript；Ant Design 5 布局；React Query/SWR 请求；WebSocket 实时数据
- **核心页面**：
- 仪表盘：在线设备数、流量、延迟、隧道成功率、告警概览(ECharts)
- 设备详情：状态、虚拟IP、NAT类型、WireGuard配置、最近会话、日志下载
- 网络拓扑：显示组织内设备互联关系、隧道连通状态
- 密钥管理：预共享交互密钥列表、设备密钥生命周期、轮换计划
- 告警与审计：时间轴 + 过滤条件(设备、级别、动作)
- 运维工具：远程下发配置、触发诊断、下线/拉黑设备
- **权限体系**：
- RBAC：系统管理员、网络运维、审计员；支持自定义角色
- 登录安全：支持OIDC/SAML接入、可选2FA、操作确认弹窗

## 支撑体系

- **监控与日志**：
- Prometheus/Grafana：采集控制面、STUN/TURN、WireGuard节点指标；自定义告警规则
- Loki/ELK：集中管理服务端日志，关联设备ID和会话
- 分布式追踪：Jaeger/OpenTelemetry 标记关键流程(注册、打洞等)
- **测试策略**：
- 单元/集成测试：Go test、WireGuard接口模拟、Postman API集合
- NAT模拟场景：利用容器或专用NAT测试工具覆盖全类型
- E2E测试：使用Playwright驱动管理界面 + 自动化脚本模拟客户端注册/打洞
- 压测：k6/Vegeta 测 REST API；对 WireGuard 隧道做吞吐与稳定性测试
- **交付运维**：
- CI/CD：GitHub Actions 构建、单元测试、镜像推送、Helm 发布
- 配置管理：集中配置(Consul/ETCD) + 环境隔离(dev/staging/prod)
- 灰度与回滚：控制平面支持蓝绿发布；客户端通过版本通道滚动升级
- **文档/支持**：
- 自动生成API文档(Swagger/OpenAPI + Stoplight)
以图形向导辅助客户部署；提供FAQ与故障排查指引