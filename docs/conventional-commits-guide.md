# Conventional Commits 指南

本项目使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范来标准化提交消息，并自动生成CHANGELOG。

## 快速开始

### 安装Git Hooks

首次克隆仓库后运行：

```bash
./scripts/setup-git-hooks.sh
```

这将安装commit-msg hook来验证你的提交消息格式。

## 提交消息格式

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

### Type（类型）

必须是以下之一：

| Type | 说明 | 出现在CHANGELOG |
|------|------|----------------|
| `feat` | 新功能 | ✅ Features |
| `fix` | Bug修复 | ✅ Bug Fixes |
| `perf` | 性能改进 | ✅ Performance |
| `docs` | 文档变更 | ✅ Documentation |
| `refactor` | 代码重构（不是新功能，也不是修bug） | ✅ Refactoring |
| `style` | 代码格式（不影响代码含义的变更） | ❌ Hidden |
| `test` | 添加或修改测试 | ❌ Hidden |
| `chore` | 构建过程或辅助工具的变动 | ❌ Hidden |
| `ci` | CI/CD配置变更 | ❌ Hidden |
| `build` | 影响构建系统或外部依赖的变更 | ❌ Hidden |
| `revert` | 回滚之前的提交 | ✅ Reverts |

### Scope（范围）

可选，描述提交影响的范围：

- **后端**: `api-gateway`, `device-service`, `topology-service`, `nat-coordinator`, `alert-service`, `background-worker`
- **前端**: `frontend`, `ui`, `dashboard`, `device-details`, `topology-view`
- **客户端**: `desktop-client`, `ios-client`, `android-client`
- **基础设施**: `docker`, `k8s`, `helm`, `ci`, `monitoring`
- **数据库**: `database`, `migrations`, `schema`
- **文档**: `docs`, `readme`, `api-docs`

### Subject（主题）

- 使用命令式语气（"add"而不是"added"或"adds"）
- 不要大写首字母
- 不要在末尾加句号
- 限制在50个字符以内

### Body（正文）

可选，提供更详细的说明：

- 使用命令式语气
- 包含变更的动机以及与之前行为的对比

### Footer（页脚）

可选，包含：

- **BREAKING CHANGE**: 不兼容的变更说明
- **Closes**: 关闭的Issue引用（如 `Closes #123, #456`）
- **Refs**: 相关Issue引用（如 `Refs #789`）

## 示例

### 简单的功能提交

```
feat(api-gateway): add rate limiting middleware
```

### 带范围和详细说明的提交

```
feat(device-service): implement device heartbeat monitoring

Add periodic heartbeat check for all registered devices.
Devices that miss 3 consecutive heartbeats are marked as offline.

Closes #234
```

### Bug修复

```
fix(database): resolve connection pool exhaustion

- Set MaxIdleConns to 50% of MaxOpenConns
- Added connection lifetime limits (5 minutes)
- Implemented pool monitoring with alerts

The previous configuration caused connection leaks under high load,
leading to "too many connections" errors.

Closes #456
```

### 破坏性变更

```
feat(api): redesign device registration API

BREAKING CHANGE: The device registration endpoint now requires
a `device_type` field. Update all clients to include this field.

Migration guide:
1. Update client SDK to v0.2.0+
2. Add device_type to registration payload
3. Redeploy clients

Before:
POST /api/v1/device/register
{
  "name": "my-device",
  "public_key": "..."
}

After:
POST /api/v1/device/register
{
  "name": "my-device",
  "device_type": "desktop",
  "public_key": "..."
}

Closes #789
```

### 性能改进

```
perf(cache): implement multi-level Redis caching

Introduce 4-tier TTL strategy:
- Short (5min): frequently changing data
- Medium (10min): periodically updated data
- Long (30min): relatively static data
- Very long (1hour): rarely changing data

Reduces database load by 60% during peak hours.

Refs #123
```

### 文档更新

```
docs(deployment): add Kubernetes Helm chart guide

Include step-by-step instructions for:
- Installing cert-manager
- Configuring PostgreSQL external database
- Setting up Redis Sentinel
- TLS certificate management
```

### 重构

```
refactor(frontend): migrate from Redux to Zustand

Zustand provides simpler API and better TypeScript support.
Reduces bundle size by 15KB and improves developer experience.

No functional changes.
```

### 回滚提交

```
revert: feat(api-gateway): add rate limiting middleware

This reverts commit abc123def456.

Rate limiting caused issues with WebSocket connections.
Need to redesign the middleware to exclude WS endpoints.
```

## 自动化工作流

### 提交验证

当你执行 `git commit` 时：

1. Git hook自动验证提交消息格式
2. 如果格式不正确，提交将被拒绝
3. 你需要修改提交消息重新提交

```bash
# ❌ 错误示例
git commit -m "added rate limiting"
# 错误：不符合Conventional Commits格式

# ✅ 正确示例
git commit -m "feat(api-gateway): add rate limiting middleware"
```

### CHANGELOG生成

当推送到main分支或创建版本tag时：

1. GitHub Actions自动从提交消息生成CHANGELOG
2. 更新 `CHANGELOG.md` 文件
3. 如果是tag，创建GitHub Release并附带变更日志

```bash
# 创建新版本
git tag v0.2.0
git push origin v0.2.0

# 自动生成：
# - CHANGELOG.md 更新
# - GitHub Release v0.2.0
# - Docker镜像打包（tag: v0.2.0）
```

### 版本号规则

遵循语义化版本（SemVer）：

- **主版本号（MAJOR）**: 包含 `BREAKING CHANGE` 的提交
- **次版本号（MINOR）**: `feat` 类型的提交
- **修订号（PATCH）**: `fix`, `perf` 类型的提交

示例：
```
v0.1.0 -> v0.1.1  (fix: bug修复)
v0.1.1 -> v0.2.0  (feat: 新功能)
v0.2.0 -> v1.0.0  (feat with BREAKING CHANGE)
```

## 常见问题

### Q: 如果我忘记遵循格式怎么办？

A: Git hook会阻止你提交。你需要使用 `git commit --amend` 修改提交消息：

```bash
git commit --amend
# 在编辑器中修改提交消息
```

### Q: 可以跳过hook验证吗？

A: 可以使用 `--no-verify`，但**强烈不推荐**：

```bash
git commit -m "quick fix" --no-verify  # ❌ 不推荐
```

这会导致CHANGELOG生成失败和版本管理混乱。

### Q: 一次提交涉及多个scope怎么办？

A: 优先选择影响最大的scope，或者拆分成多个提交：

```bash
# 方案1: 选择主要scope
git commit -m "feat(api-gateway): add authentication and rate limiting"

# 方案2: 拆分提交（更好）
git commit -m "feat(api-gateway): add authentication middleware"
git commit -m "feat(api-gateway): add rate limiting middleware"
```

### Q: 如何处理WIP（Work In Progress）提交？

A: 在本地分支可以使用非标准格式，但合并到main前需要squash：

```bash
# 本地开发
git commit -m "wip: trying different approach"
git commit -m "wip: fix syntax error"

# 合并前squash
git rebase -i main
# 将多个wip提交合并为一个规范的提交

# 推送到main
git commit -m "feat(frontend): implement device topology visualization"
```

### Q: 如何引用多个Issue？

A: 在footer中使用逗号分隔：

```
feat(api): add bulk device registration

Closes #123, #456, #789
Refs #234
```

## 最佳实践

1. **每个提交只做一件事**
   - ✅ `feat(auth): add JWT validation`
   - ❌ `feat(auth): add JWT validation and fix database connection pool`

2. **提交消息要清晰**
   - ✅ `fix(api-gateway): prevent memory leak in WebSocket handler`
   - ❌ `fix: fix bug`

3. **使用命令式语气**
   - ✅ `add`, `fix`, `update`, `remove`
   - ❌ `added`, `fixed`, `updating`, `removed`

4. **Breaking changes必须明确**
   ```
   feat(api): remove deprecated /v1/legacy endpoint

   BREAKING CHANGE: The /v1/legacy endpoint has been removed.
   Migrate to /v2/device endpoint.
   ```

5. **参考相关Issue**
   ```
   fix(database): resolve deadlock in transaction handling

   The deadlock occurred when multiple workers attempted to
   update the same device simultaneously.

   Closes #567
   ```

## 工具推荐

### 命令行工具

安装 [commitizen](https://github.com/commitizen/cz-cli) 进行交互式提交：

```bash
npm install -g commitizen cz-conventional-changelog

# 使用
git cz
```

### IDE插件

- **VSCode**: [Conventional Commits](https://marketplace.visualstudio.com/items?itemName=vivaxy.vscode-conventional-commits)
- **IntelliJ IDEA**: [Conventional Commit](https://plugins.jetbrains.com/plugin/13389-conventional-commit)

## 参考资料

- [Conventional Commits 官方规范](https://www.conventionalcommits.org/)
- [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)
- [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)
- [Angular Commit Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)

---

**维护者**: EdgeLink Team
**最后更新**: 2025-10-20
