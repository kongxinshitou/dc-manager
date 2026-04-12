import { useState, useEffect, useCallback } from 'react'
import { Card, Button, Modal, Form, Input, Checkbox, Tag, message, Popconfirm, Space, Typography, Row, Col } from 'antd'
import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import {
  getRoles, createRole, updateRole, deleteRole, getPermissionInfo,
  type RoleInfo, type PermGroup,
} from '../api'

export default function Roles() {
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [permGroups, setPermGroups] = useState<PermGroup[]>([])
  const [_loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<RoleInfo | null>(null)
  const [selectedPerms, setSelectedPerms] = useState<string[]>([])
  const [form] = Form.useForm()

  const loadRoles = useCallback(async () => {
    setLoading(true)
    try { setRoles(await getRoles()) }
    catch { message.error('加载角色列表失败') }
    finally { setLoading(false) }
  }, [])

  const loadPerms = useCallback(async () => {
    try {
      const data = await getPermissionInfo()
      setPermGroups(data.groups)
    } catch {}
  }, [])

  useEffect(() => { loadRoles(); loadPerms() }, [loadRoles, loadPerms])

  const parsePerms = (permsStr: string): string[] => {
    try { return JSON.parse(permsStr) } catch { return [] }
  }

  const handleCreate = () => {
    setEditingRole(null)
    form.resetFields()
    setSelectedPerms([])
    setModalOpen(true)
  }

  const handleEdit = (role: RoleInfo) => {
    setEditingRole(role)
    form.setFieldsValue({ name: role.name, display_name: role.display_name })
    setSelectedPerms(parsePerms(role.permissions))
    setModalOpen(true)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (editingRole) {
        await updateRole(editingRole.id, {
          name: values.name,
          display_name: values.display_name,
          permissions: selectedPerms,
        })
        message.success('角色更新成功')
      } else {
        await createRole({
          name: values.name,
          display_name: values.display_name,
          permissions: selectedPerms,
        })
        message.success('角色创建成功')
      }
      setModalOpen(false)
      loadRoles()
    } catch (err: any) {
      if (err.response?.data?.error) message.error(err.response.data.error)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteRole(id)
      message.success('角色已删除')
      loadRoles()
    } catch (err: any) {
      message.error(err.response?.data?.error || '删除失败')
    }
  }

  const togglePerm = (code: string) => {
    setSelectedPerms(prev =>
      prev.includes(code) ? prev.filter(p => p !== code) : [...prev, code]
    )
  }

  const toggleGroup = (groupPerms: string[], allSelected: boolean) => {
    setSelectedPerms(prev => {
      const prevSet = new Set(prev)
      if (allSelected) {
        groupPerms.forEach(p => prevSet.delete(p))
      } else {
        groupPerms.forEach(p => prevSet.add(p))
      }
      return Array.from(prevSet)
    })
  }

  return (
    <div style={{ padding: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={5} style={{ margin: 0 }}>角色管理</Typography.Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadRoles}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>新建角色</Button>
        </Space>
      </div>

      <Row gutter={[16, 16]}>
        {roles.map(role => {
          const perms = parsePerms(role.permissions)
          return (
            <Col key={role.id} xs={24} sm={12} md={8} lg={6}>
              <Card
                size="small"
                title={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{role.display_name}</span>
                    {role.is_system && <Tag color="blue">系统</Tag>}
                  </div>
                }
                extra={
                  !role.is_system ? (
                    <Space size={4}>
                      <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(role)} />
                      <Popconfirm title="确认删除该角色？" onConfirm={() => handleDelete(role.id)}>
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    </Space>
                  ) : (
                    <Button type="text" size="small" icon={<EditOutlined />} onClick={() => handleEdit(role)} />
                  )
                }
              >
                <div style={{ fontSize: 12, color: '#999', marginBottom: 8 }}>
                  标识: {role.name}
                </div>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                  {perms.map(p => (
                    <Tag key={p} style={{ fontSize: 11 }}>{p}</Tag>
                  ))}
                </div>
              </Card>
            </Col>
          )
        })}
      </Row>

      <Modal
        title={editingRole ? (editingRole.is_system ? `查看角色 - ${editingRole.display_name}` : `编辑角色 - ${editingRole.display_name}`) : '新建角色'}
        open={modalOpen}
        onOk={editingRole?.is_system ? () => setModalOpen(false) : handleSubmit}
        onCancel={() => setModalOpen(false)}
        okText={editingRole?.is_system ? '关闭' : '保存'}
        cancelButtonProps={editingRole?.is_system ? { style: { display: 'none' } } : {}}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="角色标识" rules={[{ required: true }]}>
            <Input disabled={!!editingRole?.is_system} placeholder="例如: operator" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名称" rules={[{ required: true }]}>
            <Input disabled={!!editingRole?.is_system} placeholder="例如: 运维人员" />
          </Form.Item>
        </Form>

        <Typography.Text strong style={{ display: 'block', marginBottom: 12 }}>权限配置</Typography.Text>
        {permGroups.map(group => {
          const groupPerms = group.permissions.map(p => p.code)
          const allSelected = groupPerms.every(p => selectedPerms.includes(p))
          const someSelected = groupPerms.some(p => selectedPerms.includes(p))

          return (
            <div key={group.label} style={{
              marginBottom: 12, padding: 12, border: '1px solid #f0f0f0', borderRadius: 6,
            }}>
              <Checkbox
                checked={allSelected}
                indeterminate={someSelected && !allSelected}
                onChange={() => toggleGroup(groupPerms, allSelected)}
                disabled={!!editingRole?.is_system}
                style={{ fontWeight: 600, marginBottom: 8, display: 'block' }}
              >
                {group.label}
              </Checkbox>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, paddingLeft: 24 }}>
                {group.permissions.map(perm => (
                  <Checkbox
                    key={perm.code}
                    checked={selectedPerms.includes(perm.code)}
                    onChange={() => togglePerm(perm.code)}
                    disabled={!!editingRole?.is_system}
                  >
                    {perm.label}
                  </Checkbox>
                ))}
              </div>
            </div>
          )
        })}
      </Modal>
    </div>
  )
}
