import React, { useState } from 'react'
import { Outlet, Link, useLocation } from 'react-router-dom'
import {
  Layout,
  Menu,
  theme,
  Typography,
  Space,
  Badge,
} from 'antd'
import {
  DashboardOutlined,
  CloudServerOutlined,
  AlertOutlined,
  AuditOutlined,
  DeploymentUnitOutlined,
  ToolOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

const { Header, Content, Sider } = Layout
const { Title } = Typography

type MenuItem = Required<MenuProps>['items'][number]

const menuItems: MenuItem[] = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: <Link to="/">仪表盘</Link>,
  },
  {
    key: '/devices',
    icon: <CloudServerOutlined />,
    label: <Link to="/devices">设备管理</Link>,
  },
  {
    key: '/alerts',
    icon: <AlertOutlined />,
    label: <Link to="/alerts">告警中心</Link>,
  },
  {
    key: '/audit',
    icon: <AuditOutlined />,
    label: <Link to="/audit">审计日志</Link>,
  },
  {
    key: '/topology',
    icon: <DeploymentUnitOutlined />,
    label: <Link to="/topology">网络拓扑</Link>,
  },
  {
    key: '/operations',
    icon: <ToolOutlined />,
    label: <Link to="/operations">运维工具</Link>,
  },
]

const MainLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const {
    token: { colorBgContainer, borderRadiusLG },
  } = theme.useToken()

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed}>
        <div
          style={{
            height: 64,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'white',
          }}
        >
          {collapsed ? (
            <Title level={4} style={{ color: 'white', margin: 0 }}>
              EL
            </Title>
          ) : (
            <Title level={4} style={{ color: 'white', margin: 0 }}>
              EdgeLink
            </Title>
          )}
        </div>
        <Menu
          theme="dark"
          selectedKeys={[location.pathname]}
          mode="inline"
          items={menuItems}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: colorBgContainer,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <Title level={4} style={{ margin: 0 }}>
            {import.meta.env.VITE_APP_TITLE || 'EdgeLink管理控制台'}
          </Title>
          <Space size="large">
            <Badge count={0} showZero={false}>
              <AlertOutlined style={{ fontSize: 20 }} />
            </Badge>
          </Space>
        </Header>
        <Content style={{ margin: '16px' }}>
          <div
            style={{
              padding: 24,
              minHeight: 360,
              background: colorBgContainer,
              borderRadius: borderRadiusLG,
            }}
          >
            <Outlet />
          </div>
        </Content>
      </Layout>
    </Layout>
  )
}

export default MainLayout
