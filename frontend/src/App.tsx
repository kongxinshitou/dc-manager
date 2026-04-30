import { useState, useMemo } from 'react'
import { Layout, Menu, theme, Dropdown, Grid, Avatar } from 'antd'
import type { MenuProps } from 'antd'
import {
  DashboardOutlined, DatabaseOutlined, AuditOutlined,
  TeamOutlined, SafetyOutlined, UserOutlined, LogoutOutlined,
  HomeOutlined, CheckCircleOutlined,
  BankOutlined, LayoutOutlined,
} from '@ant-design/icons'
import Dashboard from './pages/Dashboard'
import Devices from './pages/Devices'
import Inspections from './pages/Inspections'
import Users from './pages/Users'
import Roles from './pages/Roles'
import InspectionDetail from './pages/InspectionDetail'
import DeviceDetail from './pages/DeviceDetail'
import DatacenterManage from './pages/DatacenterManage'
import DatacenterLayout from './pages/DatacenterLayout'
import Approvals from './pages/Approvals'
import Login from './pages/Login'
import type { UserInfo } from './api'

const { Sider, Content, Header } = Layout
const { useBreakpoint } = Grid

export default function App() {
  const [currentUser, setCurrentUser] = useState<UserInfo | null>(() => {
    const saved = localStorage.getItem('user')
    return saved ? JSON.parse(saved) : null
  })
  const [page, setPage] = useState('dashboard')
  const [selectedInspectionId, setSelectedInspectionId] = useState<number | null>(null)
  const [selectedDeviceId, setSelectedDeviceId] = useState<number | null>(null)
  const { token } = theme.useToken()
  const screens = useBreakpoint()
  const isMobile = !screens.md

  const permissions = useMemo(() => new Set(currentUser?.permissions || []), [currentUser])
  const isAdmin = currentUser?.role_name === 'admin'

  const handleLogin = (user: UserInfo) => {
    setCurrentUser(user)
    setPage('dashboard')
  }

  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setCurrentUser(null)
  }

  const handleViewDetail = (id: number) => {
    setSelectedInspectionId(id)
    setPage('inspection-detail')
  }

  const handleViewDeviceDetail = (id: number) => {
    setSelectedDeviceId(id)
    setPage('device-detail')
  }

  const menuItems = useMemo(() => {
    const items: MenuProps['items'] = [
      { key: 'dashboard', icon: <DashboardOutlined />, label: '总览大屏' },
      { key: 'devices', icon: <DatabaseOutlined />, label: '设备管理' },
      { key: 'inspections', icon: <AuditOutlined />, label: '巡检记录' },
      { key: 'approvals', icon: <CheckCircleOutlined />, label: '审批管理' },
      {
        key: 'datacenter-group',
        icon: <BankOutlined />,
        label: '机房管理',
        children: [
          { key: 'datacenter-manage', icon: <HomeOutlined />, label: '机房配置' },
          { key: 'datacenter-layout', icon: <LayoutOutlined />, label: '机房布局' },
        ],
      },
    ]
    if (isAdmin || permissions.has('user:manage')) {
      items.push({ key: 'users', icon: <TeamOutlined />, label: '用户管理' })
    }
    if (isAdmin || permissions.has('role:manage')) {
      items.push({ key: 'roles', icon: <SafetyOutlined />, label: '角色管理' })
    }
    return items
  }, [isAdmin, permissions])

  // Get selected key for menu highlighting (use leaf key for submenus)
  const selectedKey = page
  const openKeys = ['datacenter-group']

  const renderPage = () => {
    if (page === 'devices') {
      return (
        <Devices
          onViewDetail={handleViewDeviceDetail}
        />
      )
    }
    if (page === 'device-detail' && selectedDeviceId) return (
      <DeviceDetail
        id={selectedDeviceId}
        onBack={() => setPage('devices')}
      />
    )
    if (page === 'inspections') return (
      <Inspections
        onGoToDevice={handleViewDeviceDetail}
        onViewDetail={handleViewDetail}
        permissions={permissions}
      />
    )
    if (page === 'inspection-detail' && selectedInspectionId) return (
      <InspectionDetail
        id={selectedInspectionId}
        onBack={() => setPage('inspections')}
        onGoToDevice={handleViewDeviceDetail}
      />
    )
    if (page === 'users') return <Users />
    if (page === 'approvals') return <Approvals />
    if (page === 'roles') return <Roles />
    if (page === 'datacenter-manage') return <DatacenterManage />
    if (page === 'datacenter-layout') return <DatacenterLayout />
    return <Dashboard />
  }

  if (!currentUser) {
    return <Login onLogin={handleLogin} />
  }

  const userMenuItems = [
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {!isMobile && (
        <Sider
          width={200}
          style={{ background: token.colorBgContainer, borderRight: `1px solid ${token.colorBorderSecondary}` }}
        >
          <div style={{
            height: 56, display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontWeight: 700, fontSize: 15, color: token.colorPrimary,
            borderBottom: `1px solid ${token.colorBorderSecondary}`
          }}>
            数据中心管理
          </div>
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            defaultOpenKeys={openKeys}
            items={menuItems}
            onClick={({ key }) => setPage(key)}
            style={{ borderRight: 0, paddingTop: 8 }}
          />
        </Sider>
      )}
      <Layout>
        <Header style={{
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          padding: isMobile ? '0 16px' : '0 24px',
          height: 56, lineHeight: '56px',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        }}>
          <span style={{ fontSize: 16, fontWeight: 600, color: token.colorText }}>
            {isMobile && <span style={{ marginRight: 8, color: token.colorPrimary, fontWeight: 700 }}>DC</span>}
            {getPageTitle(page)}
          </span>
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <span style={{ fontSize: 14 }}>{currentUser.display_name || currentUser.username}</span>
            </div>
          </Dropdown>
        </Header>
        <Content style={{
          background: token.colorBgLayout, overflow: 'auto',
          paddingBottom: isMobile ? 64 : 0,
        }}>
          {renderPage()}
        </Content>
      </Layout>

      {/* Mobile bottom navigation */}
      {isMobile && (
        <div style={{
          position: 'fixed', bottom: 0, left: 0, right: 0,
          background: token.colorBgContainer,
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex', justifyContent: 'space-around',
          padding: '6px 0', zIndex: 1000,
          paddingBottom: 'calc(6px + env(safe-area-inset-bottom))',
        }}>
          {getMobileMenuItems().map(item => {
            const active = page === item.key
            return (
              <div
                key={item.key}
                onClick={() => setPage(item.key)}
                style={{
                  display: 'flex', flexDirection: 'column', alignItems: 'center',
                  color: active ? token.colorPrimary : token.colorTextSecondary,
                  fontSize: 11, cursor: 'pointer', minWidth: 56, padding: '4px 0',
                }}
              >
                <span style={{ fontSize: 18 }}>{item.icon}</span>
                <span style={{ marginTop: 2 }}>{item.label}</span>
              </div>
            )
          })}
        </div>
      )}
    </Layout>
  )
}

function getPageTitle(page: string): string {
  const titles: Record<string, string> = {
    'dashboard': '总览大屏',
    'devices': '设备管理',
    'device-detail': '设备详情',
    'inspections': '巡检记录',
    'approvals': '审批管理',
    'datacenter-manage': '机房配置',
    'datacenter-layout': '机房布局',
    'users': '用户管理',
    'roles': '角色管理',
  }
  return titles[page] || '数据中心管理'
}

function getMobileMenuItems() {
  return [
    { key: 'dashboard', icon: <DashboardOutlined />, label: '总览' },
    { key: 'devices', icon: <DatabaseOutlined />, label: '设备' },
    { key: 'inspections', icon: <AuditOutlined />, label: '巡检' },
  ]
}
