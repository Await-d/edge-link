# Architecture Decision Records (ADR)

本目录包含EdgeLink项目的架构决策记录（Architecture Decision Records, ADRs）。

## 什么是ADR？

架构决策记录（ADR）是一种轻量级的文档格式，用于记录项目中的重要架构决策。每个ADR描述了：

- **决策内容**：我们做了什么决定
- **背景**：为什么需要做这个决定
- **考虑的选项**：我们评估了哪些方案
- **决策结果**：我们选择了什么，为什么
- **后果**：这个决策带来的影响

## ADR原则

1. **决策即文档**：重要的架构决策都应该记录下来
2. **不可变性**：ADR一旦创建就不应修改，如果决策改变应创建新的ADR
3. **上下文完整**：记录决策时的上下文、约束和考虑因素
4. **简洁明了**：每个ADR应该简短且聚焦在单一决策上
5. **持续更新**：随着项目演进，持续记录新的架构决策

## 如何创建新的ADR

使用提供的脚本创建新的ADR：

```bash
# 创建新的ADR
./scripts/new-adr.sh "使用gRPC进行服务间通信"

# 脚本会自动：
# 1. 分配下一个序号（如 0008）
# 2. 创建文件 docs/adr/0008-use-grpc-for-service-communication.md
# 3. 使用模板填充基本结构
# 4. 更新本README的索引
```

手动创建ADR：

1. 复制 `template.md` 到新文件：`NNNN-descriptive-title.md`
2. 填写所有章节
3. 设置状态为 "Proposed"（提议中）
4. 提交PR并经过团队评审
5. 评审通过后更新状态为 "Accepted"（已接受）
6. 更新本README的索引

## ADR生命周期

ADR可以有以下状态：

- **Proposed** (提议中): 新创建的ADR，等待评审
- **Accepted** (已接受): 经过评审并被团队接受
- **Deprecated** (已废弃): 决策已过时但保留历史记录
- **Superseded** (已取代): 被新的ADR取代，链接到新ADR

## ADR索引

### 核心技术栈

| ADR | 标题 | 状态 | 日期 |
|-----|------|------|------|
| [0001](0001-use-wireguard-for-vpn.md) | 使用WireGuard作为VPN协议 | Accepted | 2025-10-01 |
| [0002](0002-choose-golang-for-backend.md) | 选择Go作为后端开发语言 | Accepted | 2025-10-01 |
| [0003](0003-use-react-antd-for-frontend.md) | 使用React 19 + Ant Design 5构建前端 | Accepted | 2025-10-02 |
| [0004](0004-postgresql-redis-data-layer.md) | 使用PostgreSQL + Redis作为数据层 | Accepted | 2025-10-02 |

### 架构模式

| ADR | 标题 | 状态 | 日期 |
|-----|------|------|------|
| [0005](0005-microservices-architecture.md) | 采用微服务架构 | Accepted | 2025-10-03 |
| [0006](0006-grpc-internal-rest-external.md) | 内部通信使用gRPC，外部API使用REST | Accepted | 2025-10-03 |

### DevOps与部署

| ADR | 标题 | 状态 | 日期 |
|-----|------|------|------|
| [0007](0007-kubernetes-deployment.md) | 使用Kubernetes进行容器编排 | Accepted | 2025-10-05 |
| [0008](0008-helm-for-deployment.md) | 使用Helm管理Kubernetes部署 | Accepted | 2025-10-05 |
| [0009](0009-deterministic-builds.md) | 实现确定性构建 | Accepted | 2025-10-20 |

### 监控与可靠性

| ADR | 标题 | 状态 | 日期 |
|-----|------|------|------|
| [0010](0010-prometheus-monitoring.md) | 使用Prometheus + Grafana进行监控 | Accepted | 2025-10-10 |
| [0011](0011-automated-rollback.md) | 基于Prometheus实现自动回滚 | Accepted | 2025-10-20 |

### 安全

| ADR | 标题 | 状态 | 日期 |
|-----|------|------|------|
| [0012](0012-pre-shared-key-authentication.md) | 使用预共享密钥进行设备认证 | Accepted | 2025-10-08 |
| [0013](0013-end-to-end-encryption.md) | WireGuard端到端加密策略 | Accepted | 2025-10-08 |

## 相关资源

- [ADR工具](https://adr.github.io/)
- [ADR最佳实践](https://github.com/joelparkerhenderson/architecture-decision-record)
- [EdgeLink架构文档](../ARCHITECTURE.md)
- [EdgeLink技术栈](../README.md#tech-stack)

## 维护指南

### 评审流程

1. 作者创建ADR并设置状态为 "Proposed"
2. 提交PR到仓库
3. 团队评审（至少2个approvals）
4. 讨论并修改ADR内容
5. 批准后更新状态为 "Accepted"
6. 合并PR

### 废弃ADR

当某个决策不再适用时：

1. 不要删除原ADR文件
2. 更新状态为 "Deprecated" 或 "Superseded"
3. 如果有新的替代决策，创建新ADR并互相链接
4. 在原ADR顶部添加废弃通知：

```markdown
> **状态**: Superseded by [ADR-0015](0015-new-decision.md)
> 
> **废弃原因**: 简要说明为什么被取代
```

### 模板维护

`template.md` 是所有新ADR的基础模板。如需修改模板：

1. 更新 `template.md`
2. 在团队会议上讨论变更
3. 更新 `scripts/new-adr.sh` 脚本（如果需要）
4. 文档化模板变更原因

## 联系方式

如有关于ADR流程的问题：
- Slack: #architecture
- Email: architecture@edgelink.com
- 团队Wiki: [ADR Guidelines](https://wiki.edgelink.com/adr)

---

**最后更新**: 2025-10-20  
**维护者**: EdgeLink Architecture Team
