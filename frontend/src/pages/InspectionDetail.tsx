import { useEffect, useState, useCallback, useMemo } from 'react'
import { Descriptions, Card, Button, Spin, Tag, Space, Typography, message } from 'antd'
import { ArrowLeftOutlined, LinkOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { getInspection, type Inspection, type InspectionImage } from '../api'
import ImageGallery from '../components/ImageGallery'

interface InspectionDetailProps {
  id: number
  onBack: () => void
  onGoToDevice?: (id: number) => void
}

const severityColor: Record<string, string> = { 严重: 'red', 一般: 'orange', 轻微: 'blue' }
const statusColor: Record<string, string> = { 待处理: 'red', 处理中: 'orange', 已解决: 'green' }

export default function InspectionDetail({ id, onBack, onGoToDevice }: InspectionDetailProps) {
  const [inspection, setInspection] = useState<Inspection | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getInspection(id)
      if (!data.images) data.images = []
      setInspection(data)
    } catch {
      message.error('加载巡检详情失败')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { fetchData() }, [fetchData])

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
          <Descriptions.Item label="问题等级">
            <Tag color={severityColor[inspection.severity]}>{inspection.severity}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColor[inspection.status]}>{inspection.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="解决时间">
            {inspection.resolved_at ? dayjs(inspection.resolved_at).format('YYYY-MM-DD') : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="问题描述" span={3}>{inspection.issue || '-'}</Descriptions.Item>
          <Descriptions.Item label="备注" span={3}>{inspection.remark || '-'}</Descriptions.Item>
        </Descriptions>
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
    </div>
  )
}
