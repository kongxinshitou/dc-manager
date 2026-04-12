import { useState, useMemo } from 'react'
import { Layout, Menu, theme, Dropdown, Grid, Avatar } from 'antd'
import {
  DashboardOutlined, DatabaseOutlined, AuditOutlined,
  TeamOutlined, SafetyOutlined, UserOutlined, LogoutOutlined,
} from '@ant-design/icons'
import Dashboard from './pages/Dashboard'
import Devices from './pages/Devices'
import Inspections from './pages/Inspections'
import Users from './pages/Users'
import Roles from './pages/Roles'
import InspectionDetail from './pages/InspectionDetail'
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
  const [focusDeviceId, setFocusDeviceId] = useState<number | null>(null)
  const [selectedInspectionId, setSelectedInspectionId] = useState<number | null>(null)
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

  const handleGoToDevice = (id: number) => {
    setFocusDeviceId(id)
    setPage('devices')
  }

  const handleViewDetail = (id: number) => {
    setSelectedInspectionId(id)
    setPage('inspection-detail')
  }

  const menuItems = useMemo(() => {
    const items = [
      { key: 'dashboard', icon: <DashboardOutlined />, label: '巡检大屏' },
      { key: 'devices', icon: <DatabaseOutlined />, label: '设备台账' },
      { key: 'inspections', icon: <AuditOutlined />, label: '巡检记录' },
    ]
    if (isAdmin || permissions.has('user:manage')) {
      items.push({ key: 'users', icon: <TeamOutlined />, label: '用户管理' })
    }
    if (isAdmin || permissions.has('role:manage')) {
      items.push({ key: 'roles', icon: <SafetyOutlined />, label: '角色管理' })
    }
    return items
  }, [isAdmin, permissions])

  const renderPage = () => {
    if (page === 'devices') return (
      <Devices
        focusDeviceId={focusDeviceId}
        onFocusHandled={() => setFocusDeviceId(null)}
      />
    )
    if (page === 'inspections') return (
      <Inspections
        onGoToDevice={handleGoToDevice}
        onViewDetail={handleViewDetail}
        permissions={permissions}
      />
    )
    if (page === 'inspection-detail' && selectedInspectionId) return (
      <InspectionDetail
        id={selectedInspectionId}
        onBack={() => setPage('inspections')}
        onGoToDevice={handleGoToDevice}
      />
    )
    if (page === 'users') return <Users />
    if (page === 'roles') return <Roles />
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
          width={180}
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
            selectedKeys={[page]}
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
            {menuItems.find(m => m.key === page)?.label}
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

      {/* 移动端底部导航 */}
      {isMobile && (
        <div style={{
          position: 'fixed', bottom: 0, left: 0, right: 0,
          background: token.colorBgContainer,
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          display: 'flex', justifyContent: 'space-around',
          padding: '6px 0', zIndex: 1000,
        }}>
          {menuItems.slice(0, 4).map(item => (
            <div
              key={item.key}
              onClick={() => setPage(item.key)}
              style={{
                display: 'flex', flexDirection: 'column', alignItems: 'center',
                color: page === item.key ? token.colorPrimary : token.colorTextSecondary,
                fontSize: 11, cursor: 'pointer', minWidth: 56, padding: '4px 0',
              }}
            >
              <span style={{ fontSize: 18 }}>{item.icon}</span>
              <span style={{ marginTop: 2 }}>{item.label}</span>
            </div>
          ))}
        </div>
      )}
    </Layout>
  )
}
