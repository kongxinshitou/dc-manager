import { useState } from 'react'
import { Layout, Menu, theme } from 'antd'
import { DashboardOutlined, DatabaseOutlined, AuditOutlined } from '@ant-design/icons'
import Dashboard from './pages/Dashboard'
import Devices from './pages/Devices'
import Inspections from './pages/Inspections'

const { Sider, Content, Header } = Layout

const menuItems = [
  { key: 'dashboard', icon: <DashboardOutlined />, label: '巡检大屏' },
  { key: 'devices', icon: <DatabaseOutlined />, label: '设备台账' },
  { key: 'inspections', icon: <AuditOutlined />, label: '巡检记录' },
]

export default function App() {
  const [page, setPage] = useState('dashboard')
  const { token } = theme.useToken()

  const renderPage = () => {
    if (page === 'devices') return <Devices />
    if (page === 'inspections') return <Inspections />
    return <Dashboard />
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
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
      <Layout>
        <Header style={{
          background: token.colorBgContainer,
          borderBottom: `1px solid ${token.colorBorderSecondary}`,
          padding: '0 24px',
          height: 56, lineHeight: '56px',
          fontSize: 16, fontWeight: 600, color: token.colorText,
        }}>
          {menuItems.find(m => m.key === page)?.label}
        </Header>
        <Content style={{ background: token.colorBgLayout, overflow: 'auto' }}>
          {renderPage()}
        </Content>
      </Layout>
    </Layout>
  )
}
