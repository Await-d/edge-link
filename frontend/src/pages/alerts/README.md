# 告警管理UI组件

## 概述

告警管理UI提供了完整的告警监控、管理和响应功能,包括实时更新、音频通知、统计仪表盘等。

## 文件结构

```
src/pages/alerts/
├── AlertList.tsx                    # 主页面组件
├── components/
│   ├── AlertDetail.tsx             # 告警详情弹窗
│   ├── AlertStats.tsx              # 告警统计仪表盘
│   └── index.ts                    # 组件导出
src/services/
└── api.ts                          # API接口(已扩展)
src/hooks/
└── useApi.ts                       # React Query hooks(已扩展)
src/utils/
└── alertNotification.ts            # 音频通知管理器
src/types/
└── api.ts                          # TypeScript类型定义
```

## 核心功能

### 1. 告警列表页面 (AlertList.tsx)

**功能特性:**
- 分页表格展示所有告警
- 多维度筛选:严重程度、状态、类型
- 搜索功能(标题/消息)
- 告警操作:确认、解决、查看详情
- 实时更新(30秒自动刷新 + WebSocket推送)
- 音频通知(可配置开关)
- 活跃告警数量徽章
- Critical告警行高亮显示

**使用方式:**
```tsx
import AlertList from '@/pages/alerts/AlertList'

// 在路由中使用
<Route path="/alerts" element={<AlertList />} />
```

### 2. 告警详情弹窗 (AlertDetail)

**功能特性:**
- 展示告警完整信息(ID、类型、状态、时间等)
- 元数据JSON展示
- 时间戳相对显示(如"3分钟前")
- 一键复制ID和设备ID

**Props接口:**
```tsx
interface AlertDetailProps {
  alert: Alert | null      // 告警对象
  open: boolean           // 是否打开弹窗
  onClose: () => void     // 关闭回调
}
```

**使用示例:**
```tsx
import { AlertDetail } from '@/pages/alerts/components'

<AlertDetail
  alert={selectedAlert}
  open={detailVisible}
  onClose={() => setDetailVisible(false)}
/>
```

### 3. 告警统计仪表盘 (AlertStats)

**功能特性:**
- 活跃/已确认/已解决/总计数量卡片
- 按严重程度分类统计(Critical/High/Medium/Low)
- 颜色编码指示
- 响应式网格布局

**Props接口:**
```tsx
interface AlertStatsProps {
  alerts: Alert[]         // 告警列表
  loading?: boolean       // 加载状态
}
```

**使用示例:**
```tsx
import { AlertStats } from '@/pages/alerts/components'

<AlertStats alerts={data?.data || []} loading={isLoading} />
```

### 4. 音频通知系统 (alertNotification)

**功能特性:**
- 根据严重程度播放不同音调
- 防抖机制(3秒内最多播放一次)
- 用户配置持久化(localStorage)
- 浏览器AudioContext API实现

**API:**
```typescript
import {
  alertNotification,
  initAlertNotification,
  setAlertNotificationEnabled
} from '@/utils/alertNotification'

// 初始化(从localStorage读取设置)
initAlertNotification()

// 启用/禁用
alertNotification.enable()
alertNotification.disable()

// 播放告警声音
alertNotification.play('critical')

// 测试声音
alertNotification.test()

// 设置并保存偏好
setAlertNotificationEnabled(true)
```

**声音设计:**
- **Critical**: 5次高频急促警报 (880-988Hz, 1.5秒)
- **High**: 3次中高频警报 (660-784Hz, 1.0秒)
- **Medium**: 2次中频提示音 (523-659Hz, 0.6秒)
- **Low**: 单次低频提示 (440Hz, 0.3秒)

## API集成

### 新增API端点

```typescript
// 获取告警详情
alertApi.getAlertById(alertId: string): Promise<Alert>

// 解决告警
alertApi.resolveAlert(alertId: string): Promise<void>
```

### React Query Hooks

```typescript
// 获取告警列表(30秒自动刷新)
const { data, isLoading, refetch } = useAlerts({
  severity: 'critical',
  status: 'active',
  limit: 20,
  offset: 0
})

// 获取单个告警详情
const { data: alert } = useAlertById(alertId)

// 确认告警
const acknowledgeAlert = useAcknowledgeAlert()
await acknowledgeAlert.mutateAsync({
  alertId,
  acknowledgedBy: userId
})

// 解决告警
const resolveAlert = useResolveAlert()
await resolveAlert.mutateAsync(alertId)
```

## UI/UX设计

### 颜色编码

**严重程度:**
- Critical: 红色 (#ff4d4f)
- High: 橙色 (#fa8c16)
- Medium: 金色 (#faad14)
- Low: 蓝色 (#1890ff)

**状态:**
- Active: 红色
- Acknowledged: 橙色
- Resolved: 绿色

### 响应式设计

- 统计卡片网格:
  - 手机: 单列 (xs=24)
  - 平板: 双列 (sm=12)
  - 桌面: 四列 (lg=6)

- 表格横向滚动支持(宽度<1200px)
- 操作列固定在右侧

### 交互细节

- Critical告警行背景高亮(粉红色)
- 活跃告警数量徽章
- 时间相对显示(悬停显示完整时间)
- 操作按钮确认提示
- 加载状态反馈

## WebSocket实时更新

告警页面订阅WebSocket事件:
- `alert_created`: 新告警创建
- `alert_updated`: 告警状态更新

收到事件后自动刷新列表并触发音频通知。

## 类型定义

```typescript
// 严重程度
type Severity = 'critical' | 'high' | 'medium' | 'low'

// 告警状态
type AlertStatus = 'active' | 'acknowledged' | 'resolved'

// 告警类型
type AlertType =
  | 'device_offline'
  | 'high_latency'
  | 'connection_failed'
  | 'key_expired'
  | 'system'

// 告警对象
interface Alert {
  id: string
  device_id?: string
  severity: Severity
  alert_type: AlertType
  title: string
  message: string
  status: AlertStatus
  metadata?: Record<string, any>
  acknowledged_at?: string
  acknowledged_by?: string
  resolved_at?: string
  created_at: string
}
```

## 使用指南

### 1. 查看告警

访问 `/alerts` 路由查看告警列表。页面默认按创建时间倒序排列,最新告警在最前。

### 2. 筛选告警

使用顶部筛选栏:
- 严重程度下拉框
- 状态下拉框
- 类型下拉框
- 搜索框(标题/消息)

### 3. 确认告警

对于"活跃"状态的告警:
1. 点击"确认"按钮
2. 确认对话框中点击"确定"
3. 告警状态变为"已确认"

### 4. 解决告警

对于"已确认"状态的告警:
1. 点击"解决"按钮
2. 确认对话框中点击"确定"
3. 告警状态变为"已解决"

### 5. 查看详情

点击任意告警的"详情"按钮,弹出详情对话框,查看完整信息和元数据。

### 6. 音频通知

- 点击右上角铃铛开关启用/禁用音频通知
- 首次启用时会播放测试音
- 设置会保存到浏览器localStorage

### 7. 实时更新

- 列表每30秒自动刷新一次
- WebSocket连接时收到实时推送
- 新告警会触发音频提示(如果已启用)
- 点击"刷新"按钮手动刷新

## 性能优化

1. **自动刷新**: 使用React Query的`refetchInterval`实现,不会阻塞UI
2. **WebSocket**: 仅在收到相关事件时才刷新,减少不必要的请求
3. **音频防抖**: 3秒内最多播放一次,避免声音轰炸
4. **分页加载**: 支持10/20/50/100条/页,减少单次数据量
5. **状态缓存**: React Query自动缓存,切换回页面时立即显示缓存数据

## 可访问性 (A11y)

- 语义化HTML标签
- ARIA标签适当使用(Ant Design内置)
- 键盘导航支持(表格、按钮、下拉框)
- 颜色对比度符合WCAG AA标准
- 屏幕阅读器友好(Tooltip、Tag文本)

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

**注意**: 音频通知需要AudioContext API支持,IE不支持。

## 未来扩展

可能的增强功能:
- 批量操作(批量确认/解决)
- 告警规则配置
- 告警趋势图表
- 导出CSV/PDF
- 邮件/Slack通知集成
- 告警分组/聚合
- 自定义音频文件上传
