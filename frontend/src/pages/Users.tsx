import { useState, useEffect, useCallback } from 'react'
import { Table, Button, Modal, Form, Input, Select, Tag, message, Popconfirm, Space, Typography } from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import { getUsers, createUser, updateUser, resetPassword, deleteUser, getRoles, type UserInfo, type RoleInfo } from '../api'

export default function Users() {
  const [users, setUsers] = useState<UserInfo[]>([])
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [resetModalOpen, setResetModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null)
  const [resettingUser, setResettingUser] = useState<UserInfo | null>(null)
  const [form] = Form.useForm()
  const [resetForm] = Form.useForm()

  const loadUsers = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getUsers()
      setUsers(data)
    } catch { message.error('加载用户列表失败') }
    finally { setLoading(false) }
  }, [])

  const loadRoles = useCallback(async () => {
    try { setRoles(await getRoles()) } catch {}
  }, [])

  useEffect(() => { loadUsers(); loadRoles() }, [loadUsers, loadRoles])

  const handleCreate = () => {
    setEditingUser(null)
    form.resetFields()
    setModalOpen(true)
  }

  const handleEdit = (user: UserInfo) => {
    setEditingUser(user)
    form.setFieldsValue({
      username: user.username,
      display_name: user.display_name,
      role_id: user.role_id,
      status: user.status,
    })
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingUser) {
        await updateUser(editingUser.id, values)
        message.success('更新成功')
      } else {
        await createUser(values)
        message.success('创建成功')
      }
      setModalOpen(false)
      loadUsers()
    } catch (err: any) {
      if (err.response?.data?.error) message.error(err.response.data.error)
    }
  }

  const handleResetPassword = async () => {
    if (!resettingUser) return
    try {
      const values = await resetForm.validateFields()
      if (values.new_password !== values.confirm_password) {
        message.error('两次输入的密码不一致')
        return
      }
      await resetPassword(resettingUser.id, values.new_password)
      message.success('密码重置成功')
      setResetModalOpen(false)
    } catch (err: any) {
      if (err.response?.data?.error) message.error(err.response.data.error)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteUser(id)
      message.success('用户已删除')
      loadUsers()
    } catch (err: any) {
      message.error(err.response?.data?.error || '删除失败')
    }
  }

  const columns = [
    { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
    { title: '显示名称', dataIndex: 'display_name', key: 'display_name', width: 120 },
    {
      title: '角色', key: 'role', width: 120,
      render: (_: any, r: UserInfo) => r.role?.display_name || r.role_name || '-'
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (s: string) => <Tag color={s === 'active' ? 'green' : 'red'}>{s === 'active' ? '正常' : '禁用'}</Tag>
    },
    {
      title: '操作', key: 'actions', width: 240,
      render: (_: any, r: UserInfo) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => handleEdit(r)}>编辑</Button>
          <Button type="link" size="small" onClick={() => { setResettingUser(r); resetForm.resetFields(); setResetModalOpen(true) }}>
            重置密码
          </Button>
          <Popconfirm title="确认删除该用户？" onConfirm={() => handleDelete(r.id)}>
            <Button type="link" size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      )
    },
  ]

  return (
    <div style={{ padding: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={5} style={{ margin: 0 }}>用户管理</Typography.Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadUsers}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新建用户</Button>
        </Space>
      </div>

      <Table
        dataSource={users}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={false}
        size="small"
        scroll={{ x: 700 }}
      />

      <Modal
        title={editingUser ? '编辑用户' : '新建用户'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, min: 3, max: 50 }]}>
            <Input disabled={!!editingUser} />
          </Form.Item>
          {!editingUser && (
            <Form.Item name="password" label="密码" rules={[{ required: true, min: 6 }]}>
              <Input.Password />
            </Form.Item>
          )}
          <Form.Item name="display_name" label="显示名称">
            <Input />
          </Form.Item>
          <Form.Item name="role_id" label="角色" rules={[{ required: true }]}>
            <Select>
              {roles.map(r => (
                <Select.Option key={r.id} value={r.id}>{r.display_name}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          {editingUser && (
            <Form.Item name="status" label="状态" rules={[{ required: true }]}>
              <Select>
                <Select.Option value="active">正常</Select.Option>
                <Select.Option value="disabled">禁用</Select.Option>
              </Select>
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        title={`重置密码 - ${resettingUser?.username || ''}`}
        open={resetModalOpen}
        onOk={handleResetPassword}
        onCancel={() => setResetModalOpen(false)}
        destroyOnClose
      >
        <Form form={resetForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="new_password" label="新密码" rules={[{ required: true, min: 6 }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item name="confirm_password" label="确认密码" rules={[{ required: true }]}>
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
