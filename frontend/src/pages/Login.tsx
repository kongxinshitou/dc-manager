import { useState } from 'react'
import { Form, Input, Button, message } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { login, type UserInfo } from '../api'

interface LoginProps {
  onLogin: (user: UserInfo) => void
}

export default function Login({ onLogin }: LoginProps) {
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (values: { username: string; password: string }) => {
    setLoading(true)
    try {
      const res = await login(values.username, values.password)
      localStorage.setItem('token', res.token)
      localStorage.setItem('user', JSON.stringify(res.user))
      message.success('登录成功')
      onLogin(res.user)
    } catch (err: any) {
      message.error(err.response?.data?.error || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="zl-login-bg">
      <div className="zl-login-card">
        <div className="zl-brand" style={{ marginBottom: 18 }}>
          <span className="zl-brand-mark">Z</span>
          <span className="zl-brand-text">
            <span className="zl-brand-cn">中联重科</span>
            <span className="zl-brand-en">ZOOMLION</span>
          </span>
        </div>
        <div className="zl-login-accent" />
        <h1 className="zl-login-title">数据中心管理系统</h1>
        <div className="zl-login-subtitle">DATA CENTER MANAGEMENT</div>

        <Form onFinish={handleSubmit} size="large">
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" autoComplete="username" />
          </Form.Item>
          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" autoComplete="current-password" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block>
              登录
            </Button>
          </Form.Item>
        </Form>

        <div className="zl-login-footer">
          智能制造 · 工程机械 · 绿色科技
        </div>
      </div>
    </div>
  )
}
