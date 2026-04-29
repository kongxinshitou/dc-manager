import { useEffect, useState, useCallback } from 'react'
import { Descriptions, Card, Button, Spin, Tag, Typography, message } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { getDevice, type Device } from '../api'

interface DeviceDetailProps {
  id: number
  onBack: () => void
}

const deviceStatusLabel: Record<string, string> = { in_stock: '入库', out_stock: '出库' }
const subStatusLabel: Record<string, string> = {
  new_purchase: '新购', recycled: '回收', racked: '上架',
  dispatched: '外发', scrapped: '报废', unracked: '下架',
}
const warrantyColor: Record<string, string> = {
  in_warranty: 'green', out_of_warranty: 'red', unknown: 'default',
}
const warrantyLabel: Record<string, string> = {
  in_warranty: '在保', out_of_warranty: '脱保', unknown: '未知',
}

const fmtDate = (s: string | null | undefined) =>
  s ? dayjs(s).format('YYYY-MM-DD') : '-'
const fmtDateTime = (s: string | null | undefined) =>
  s ? dayjs(s).format('YYYY-MM-DD HH:mm') : '-'
const v = (x: string | number | null | undefined) =>
  x === null || x === undefined || x === '' ? '-' : x

export default function DeviceDetail({ id, onBack }: DeviceDetailProps) {
  const [device, setDevice] = useState<Device | null>(null)
  const [warrantyStatus, setWarrantyStatus] = useState<string>('')
  const [loading, setLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const res = await getDevice(id)
      setDevice(res.device)
      setWarrantyStatus(res.warranty_status)
    } catch {
      message.error('加载设备详情失败')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => { fetchData() }, [fetchData])

  if (loading) return <div style={{ padding: 40, textAlign: 'center' }}><Spin /></div>
  if (!device) return <div style={{ padding: 40, textAlign: 'center' }}>未找到设备记录</div>

  const titleText = [device.brand, device.model].filter(Boolean).join(' ').trim() || `#${device.id}`

  return (
    <div style={{ padding: 20, maxWidth: 1400, margin: '0 auto' }}>
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <Button icon={<ArrowLeftOutlined />} onClick={onBack}>返回列表</Button>
        <Typography.Title level={5} style={{ margin: 0 }}>
          设备详情 · {titleText}
        </Typography.Title>
      </div>

      <Card size="small" title="基本信息" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" bordered>
          <Descriptions.Item label="ID">{device.id}</Descriptions.Item>
          <Descriptions.Item label="来源区域">{v(device.source)}</Descriptions.Item>
          <Descriptions.Item label="资产编号">{v(device.asset_number)}</Descriptions.Item>
          <Descriptions.Item label="设备状态">
            <Tag color={device.device_status === 'in_stock' ? 'blue' : 'orange'}>
              {(deviceStatusLabel[device.device_status] || '-') + '-' + (subStatusLabel[device.sub_status] || '-')}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="品牌">{v(device.brand)}</Descriptions.Item>
          <Descriptions.Item label="型号">{v(device.model)}</Descriptions.Item>
          <Descriptions.Item label="设备类型">{v(device.device_type)}</Descriptions.Item>
          <Descriptions.Item label="厂商">{v(device.vendor)}</Descriptions.Item>
          <Descriptions.Item label="序列号" span={2}>{v(device.serial_number)}</Descriptions.Item>
          <Descriptions.Item label="操作系统">{v(device.os)}</Descriptions.Item>
          <Descriptions.Item label="在保状态">
            <Tag color={warrantyColor[warrantyStatus] || 'default'}>{warrantyLabel[warrantyStatus] || warrantyStatus || '未知'}</Tag>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="位置 / 网络" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" bordered>
          <Descriptions.Item label="机房">{v(device.datacenter)}</Descriptions.Item>
          <Descriptions.Item label="机柜">{v(device.cabinet)}</Descriptions.Item>
          <Descriptions.Item label="U位">{v(device.u_position)}</Descriptions.Item>
          <Descriptions.Item label="占用U数">{v(device.u_count)}</Descriptions.Item>
          <Descriptions.Item label="存放位置" span={2}>{v(device.storage_location)}</Descriptions.Item>
          <Descriptions.Item label="业务地址" span={2}>{v(device.business_address)}</Descriptions.Item>
          <Descriptions.Item label="IP 地址">{v(device.ip_address)}</Descriptions.Item>
          <Descriptions.Item label="管理 IP">{v(device.mgmt_ip)}</Descriptions.Item>
          <Descriptions.Item label="管理口账号">{v(device.mgmt_account)}</Descriptions.Item>
          <Descriptions.Item label="系统账号">{v(device.system_account)}</Descriptions.Item>
          <Descriptions.Item label="VIP 地址" span={2}>{v(device.vip_address)}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="维保 / 合同" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" bordered>
          <Descriptions.Item label="出厂时间">{fmtDate(device.manufacture_date)}</Descriptions.Item>
          <Descriptions.Item label="到货日期">{fmtDate(device.arrival_date)}</Descriptions.Item>
          <Descriptions.Item label="维保起始">{fmtDate(device.warranty_start)}</Descriptions.Item>
          <Descriptions.Item label="维保结束">{fmtDate(device.warranty_end)}</Descriptions.Item>
          <Descriptions.Item label="原厂维保年限">{device.warranty_years || '-'}</Descriptions.Item>
          <Descriptions.Item label="合同号">{v(device.contract_no)}</Descriptions.Item>
          <Descriptions.Item label="财务编号(旧)">{v(device.finance_no)}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="责任 / 业务" style={{ marginBottom: 16 }}>
        <Descriptions column={{ xs: 1, sm: 2, md: 3, lg: 4 }} size="small" bordered>
          <Descriptions.Item label="责任人">{v(device.owner)}</Descriptions.Item>
          <Descriptions.Item label="保管员">{v(device.custodian)}</Descriptions.Item>
          <Descriptions.Item label="申请人">{v(device.applicant)}</Descriptions.Item>
          <Descriptions.Item label="所属部门">{v(device.department)}</Descriptions.Item>
          <Descriptions.Item label="所属业务">{v(device.business_unit)}</Descriptions.Item>
          <Descriptions.Item label="项目名称">{v(device.project_name)}</Descriptions.Item>
          <Descriptions.Item label="设备用途" span={2}>{v(device.purpose)}</Descriptions.Item>
          <Descriptions.Item label="外发地址" span={2}>{v(device.dispatch_address)}</Descriptions.Item>
          <Descriptions.Item label="外发保管人">{v(device.dispatch_custodian)}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card size="small" title="备注 / 时间">
        <Descriptions column={{ xs: 1, sm: 2 }} size="small" bordered>
          <Descriptions.Item label="备注" span={2}>{v(device.remark)}</Descriptions.Item>
          <Descriptions.Item label="报废备注" span={2}>{v(device.scrap_remark)}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{fmtDateTime(device.created_at)}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{fmtDateTime(device.updated_at)}</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}
