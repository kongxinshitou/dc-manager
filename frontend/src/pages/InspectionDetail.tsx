import { useEffect, useState, useCallback, useMemo } from 'react'
import { Descriptions, Card, Button, Spin, Tag, Space, Typography, message, Modal, Form, Select, Input, Timeline } from 'antd'
import { ArrowLeftOutlined, LinkOutlined, PlayCircleOutlined, CheckCircleOutlined, RollbackOutlined, UserSwitchOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import {
  getInspection, getInspectionEvents, getUserOptions, transitionInspection,
  type Inspection, type InspectionImage, type InspectionEvent, type UserOption
} from '../api'
import ImageGallery from '../components/ImageGallery'

interface InspectionDetailProps {
  id: number
  onBack: () => void
  onGoToDevice?: (id: number) => void
}

const severityColor: Record<string, string> = { 严重: 'red', 一般: 'orange', 轻微: 'blue' }
const statusColor: Record<string, string> = { 待处理: 'red', 处理中: 'orange', 已解决: 'green' }
const eventLabel: Record<string, string> = {
  created: '创建',
  assigned: '指派',
  started: '开始处理',
  resolved: '解决',
  reopened: '重开',
  escalated: '升级',
  updated: '更新',
  deleted: '删除',
}

export default function InspectionDetail({ id, onBack, onGoToDevice }: InspectionDetailProps) {
  const [inspection, setInspection] = useState<Inspection | null>(null)
  const [events, setEvents] = useState<InspectionEvent[]>([])
  const [userOptions, setUserOptions] = useState<UserOption[]>([])
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [assignOpen, setAssignOpen] = useState(false)
  const [assignForm] = Form.useForm()

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [data, eventData] = await Promise.all([
        getInspection(id),
        getInspectionEvents(id).catch(() => []),
      ])
      if (!data.images) data.images = []
      setInspection(data)
      setEvents(eventData)
    } catch {
      message.error('加载巡检详情失败')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchData()
    getUserOptions().then(setUserOptions).catch(() => setUserOptions([]))
  }, [fetchData])

  const handleImagesChange = (images: InspectionImage[]) => {
    if (inspection) {
      setInspection({ ...inspection, images })
    }
  }

  // 从 localStorage 获取当前用户权限
  const permissions = useMemo(() => {
    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}')
      return new Set(user.permissions || [])
    } catch { return new Set<string>() }
  }, [])

  const isAdmin = useMemo(() => {
    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}')
      return user.role_name === 'admin'
    } catch { return false }
  }, [])

  const canWriteInspection = isAdmin || permissions.has('inspection:write')

  const runTransition = async (action: string, successText: string, assigneeId?: number | null, remark?: string) => {
    if (!inspection) return
    setActionLoading(true)
    try {
      await transitionInspection(inspection.id, { action, assignee_id: assigneeId, remark })
      message.success(successText)
      await fetchData()
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败')
    } finally {
      setActionLoading(false)
    }
  }

  const handleAssign = async () => {
    const values = await assignForm.validateFields()
    await runTransition('assign', '指派成功', values.assignee_id, values.remark)
    setAssignOpen(false)
  }

  if (loading) return <div style={{ padding: 40, textAlign: 'center' }}><Spin /></div>
  if (!inspection) return <div style={{ padding: 40, textAlign: 'center' }}>未找到巡检记录</div>

  return (
    <div style={{ padding: 20, maxWidth: 1400, margin: '0 auto' }}>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={onBack}>返回列表</Button>
        <Typography.Title level={5} style={{ margin: 0 }}>
          巡检记录 #{inspection.id}
        </Typography.Title>
      </div>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" bordered>
          <Descriptions.Item label="机房">{inspection.datacenter || '-'}</Descriptions.Item>
          <Descriptions.Item label="机柜">{inspection.cabinet || '-'}</Descriptions.Item>
          <Descriptions.Item label="U位">
            {inspection.start_u != null && inspection.end_u != null
              ? (inspection.start_u === inspection.end_u ? `${inspection.start_u}U` : `${inspection.start_u}-${inspection.end_u}U`)
              : inspection.u_position || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="关联设备">
            {inspection.device ? (
              onGoToDevice && inspection.device_id ? (
                <Button type="link" size="small" icon={<LinkOutlined />} style={{ padding: 0 }}
                  onClick={() => onGoToDevice(inspection.device_id!)}>
                  {inspection.device.brand} {inspection.device.model}
                </Button>
              ) : (
                `${inspection.device.brand} ${inspection.device.model}`
              )
            ) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="发现时间">{dayjs(inspection.found_at).format('YYYY-MM-DD HH:mm')}</Descriptions.Item>
          <Descriptions.Item label="巡检人">{inspection.inspector}</Descriptions.Item>
          <Descriptions.Item label="责任人">{inspection.assignee_name || '-'}</Descriptions.Item>
          <Descriptions.Item label="问题等级">
            <Tag color={severityColor[inspection.severity]}>{inspection.severity}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColor[inspection.status]}>{inspection.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="解决时间">
            {inspection.resolved_at ? dayjs(inspection.resolved_at).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="升级等级">
            <Tag color={inspection.escalation_level > 0 ? 'volcano' : 'default'}>
              {inspection.escalation_level > 0 ? `L${inspection.escalation_level}` : '无'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="最近响应">
            {inspection.last_responded_at ? dayjs(inspection.last_responded_at).format('YYYY-MM-DD HH:mm') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="最近升级">
            {inspection.last_escalated_at ? dayjs(inspection.last_escalated_at).format('YYYY-MM-DD HH:mm') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="问题描述" span={3}>{inspection.issue || '-'}</Descriptions.Item>
          <Descriptions.Item label="备注" span={3}>{inspection.remark || '-'}</Descriptions.Item>
        </Descriptions>
        {canWriteInspection && (
          <Space wrap style={{ marginTop: 12 }}>
            <Button icon={<UserSwitchOutlined />} loading={actionLoading} onClick={() => {
              assignForm.setFieldsValue({ assignee_id: inspection.assignee_id ?? undefined, remark: '' })
              setAssignOpen(true)
            }}>指派</Button>
            {inspection.status !== '已解决' && (
              <Button icon={<PlayCircleOutlined />} loading={actionLoading} onClick={() => runTransition('start_processing', '已开始处理')}>
                开始处理
              </Button>
            )}
            {inspection.status !== '已解决' && (
              <Button type="primary" icon={<CheckCircleOutlined />} loading={actionLoading} onClick={() => runTransition('resolve', '已解决')}>
                解决
              </Button>
            )}
            {inspection.status === '已解决' && (
              <Button icon={<RollbackOutlined />} loading={actionLoading} onClick={() => runTransition('reopen', '已重开')}>
                重开
              </Button>
            )}
          </Space>
        )}
      </Card>

      <Card size="small" title="生命周期" style={{ marginBottom: 16 }}>
        {events.length > 0 ? (
          <Timeline
            items={events.map(ev => ({
              color: ev.event_type === 'escalated' ? 'red' : ev.event_type === 'resolved' ? 'green' : 'blue',
              children: (
                <div>
                  <Space size={6} wrap>
                    <strong>{eventLabel[ev.event_type] || ev.event_type}</strong>
                    {ev.from_status || ev.to_status ? <span>{ev.from_status || '-'} → {ev.to_status || '-'}</span> : null}
                    {ev.assignee_name && <Tag>{ev.assignee_name}</Tag>}
                    {ev.escalation_level > 0 && <Tag color="volcano">L{ev.escalation_level}</Tag>}
                    {ev.webhook_status === 'failed' && <Tag color="red">Webhook失败</Tag>}
                    {ev.webhook_status === 'sent' && <Tag color="green">Webhook已发</Tag>}
                  </Space>
                  <div style={{ color: '#666', fontSize: 12, marginTop: 4 }}>
                    {dayjs(ev.created_at).format('YYYY-MM-DD HH:mm:ss')}{ev.remark ? ` · ${ev.remark}` : ''}
                  </div>
                  {ev.webhook_error && <div style={{ color: '#cf1322', fontSize: 12 }}>{ev.webhook_error}</div>}
                </div>
              ),
            }))}
          />
        ) : (
          <Typography.Text type="secondary">暂无生命周期事件</Typography.Text>
        )}
      </Card>

      <Card
        size="small"
        title={<Space><span>巡检图片</span><Tag>({inspection.images?.length || 0}/9)</Tag></Space>}
      >
        <ImageGallery
          inspectionId={inspection.id}
          images={inspection.images || []}
          canUpload={isAdmin || permissions.has('image:upload')}
          canDelete={isAdmin || permissions.has('image:delete')}
          onChange={handleImagesChange}
        />
      </Card>

      <Modal
        title="指派责任人"
        open={assignOpen}
        onOk={handleAssign}
        onCancel={() => setAssignOpen(false)}
        confirmLoading={actionLoading}
        destroyOnClose
      >
        <Form form={assignForm} layout="vertical">
          <Form.Item label="责任人" name="assignee_id" rules={[{ required: true, message: '请选择责任人' }]}>
            <Select showSearch optionFilterProp="label" placeholder="选择责任人">
              {userOptions.map(u => (
                <Select.Option key={u.id} value={u.id} label={`${u.label} ${u.username}`}>
                  {u.label}{u.username && u.username !== u.label ? `（${u.username}）` : ''}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label="备注" name="remark">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
