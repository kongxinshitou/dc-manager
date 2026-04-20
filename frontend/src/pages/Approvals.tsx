import { useEffect, useState, useCallback } from 'react'
import {
  Table, Tabs, Tag, Space, Button, Drawer, Input, message, Popconfirm, Card, Grid,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import {
  getApprovals, approveApproval, rejectApproval, executeApproval, cancelApproval,
} from '../api'
import type { Approval, ApprovalQuery } from '../api'

const statusConfig: Record<string, { color: string; label: string }> = {
  pending: { color: 'processing', label: '待审批' },
  approved: { color: 'warning', label: '已批准' },
  rejected: { color: 'error', label: '已驳回' },
  executed: { color: 'success', label: '已执行' },
  cancelled: { color: 'default', label: '已取消' },
}

const opTypeLabel: Record<string, string> = {
  rack: '上架', dispatch: '外发', scrap: '报废', unrack: '下架',
  in_stock_new: '入库(新购)', in_stock_recycle: '入库(回收)',
}

export default function Approvals() {
  const [data, setData] = useState<Approval[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [activeTab, setActiveTab] = useState('pending')
  const [page, setPage] = useState(1)
  const [detailId, setDetailId] = useState<number | null>(null)
  const [detailData, setDetailData] = useState<any>(null)
  const [rejectRemark, setRejectRemark] = useState('')
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  const fetchData = useCallback((tab: string, p: number) => {
    setLoading(true)
    const query: ApprovalQuery = { tab, page: p, page_size: 20 }
    getApprovals(query).then(res => {
      setData(res.data || [])
      setTotal(res.total)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  useEffect(() => { fetchData(activeTab, page) }, [activeTab, page, fetchData])

  const openDetail = (id: number) => {
    setDetailId(id)
    getApproval(id).then(setDetailData)
  }

  const handleApprove = async (id: number) => {
    try {
      await approveApproval(id)
      message.success('已批准')
      fetchData(activeTab, page)
      if (detailId === id) openDetail(id)
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败')
    }
  }

  const handleReject = async (id: number) => {
    try {
      await rejectApproval(id, rejectRemark)
      message.success('已驳回')
      setRejectRemark('')
      fetchData(activeTab, page)
      if (detailId === id) openDetail(id)
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败')
    }
  }

  const handleExecute = async (id: number) => {
    try {
      await executeApproval(id)
      message.success('执行成功')
      fetchData(activeTab, page)
      if (detailId === id) openDetail(id)
    } catch (err: any) {
      message.error(err.response?.data?.error || '执行失败')
    }
  }

  const handleCancel = async (id: number) => {
    try {
      await cancelApproval(id)
      message.success('已取消')
      fetchData(activeTab, page)
      if (detailId === id) openDetail(id)
    } catch (err: any) {
      message.error(err.response?.data?.error || '取消失败')
    }
  }

  const parseRequestData = (data: string) => {
    try { return JSON.parse(data) } catch { return {} }
  }

  const columns: ColumnsType<Approval> = [
    { title: '审批单号', dataIndex: 'approval_no', width: 150, ellipsis: true },
    { title: '操作类型', dataIndex: 'operation_type', width: 90,
      render: v => <Tag>{opTypeLabel[v] || v}</Tag>
    },
    { title: '申请人', dataIndex: 'applicant_name', width: 80 },
    { title: '状态', dataIndex: 'status', width: 80,
      render: v => { const cfg = statusConfig[v] || { color: 'default', label: v }; return <Tag color={cfg.color}>{cfg.label}</Tag> }
    },
    { title: '审批人', dataIndex: 'approver_name', width: 80, render: v => v || '-' },
    { title: '申请时间', dataIndex: 'created_at', width: 140,
      render: v => dayjs(v).format('YYYY-MM-DD HH:mm')
    },
    {
      title: '操作', width: 180, fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button size="small" type="link" onClick={() => openDetail(record.id)}>详情</Button>
          {record.status === 'pending' && <Button size="small" type="link" style={{ color: '#52c41a' }} onClick={() => handleApprove(record.id)}>批准</Button>}
          {record.status === 'pending' && <Button size="small" type="link" danger onClick={() => handleReject(record.id)}>驳回</Button>}
          {record.status === 'approved' && <Button size="small" type="link" style={{ color: '#1890ff' }} onClick={() => handleExecute(record.id)}>执行</Button>}
          {record.status === 'pending' && <Button size="small" type="link" onClick={() => handleCancel(record.id)}>取消</Button>}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 16 }}>
      <Card size="small">
        <Tabs activeKey={activeTab} onChange={key => { setActiveTab(key); setPage(1) }} items={[
          { key: 'pending', label: '待审批' },
          { key: 'my_requests', label: '我的申请' },
          { key: 'all', label: '全部' },
        ]} />
      </Card>

      <Table
        dataSource={data}
        columns={columns}
        rowKey="id"
        size="small"
        loading={loading}
        scroll={{ x: 900 }}
        pagination={{
          total, pageSize: 20, current: page,
          onChange: p => setPage(p),
        }}
      />

      <Drawer
        title={detailData?.approval?.approval_no || '审批详情'}
        open={!!detailId}
        onClose={() => { setDetailId(null); setDetailData(null) }}
        width={isMobile ? '100%' : 480}
      >
        {detailData && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Tag color={statusConfig[detailData.approval.status]?.color}>
                  {statusConfig[detailData.approval.status]?.label}
                </Tag>
                <Tag>{opTypeLabel[detailData.approval.operation_type]}</Tag>
              </Space>
            </div>

            <div style={{ marginBottom: 16, lineHeight: 2 }}>
              <div><strong>审批单号:</strong> {detailData.approval.approval_no}</div>
              <div><strong>申请人:</strong> {detailData.approval.applicant_name}</div>
              <div><strong>审批人:</strong> {detailData.approval.approver_name || '-'}</div>
              <div><strong>申请时间:</strong> {dayjs(detailData.approval.created_at).format('YYYY-MM-DD HH:mm:ss')}</div>
              {detailData.approval.approved_at && <div><strong>审批时间:</strong> {dayjs(detailData.approval.approved_at).format('YYYY-MM-DD HH:mm:ss')}</div>}
              {detailData.approval.executed_at && <div><strong>执行时间:</strong> {dayjs(detailData.approval.executed_at).format('YYYY-MM-DD HH:mm:ss')}</div>}
            </div>

            {detailData.device && (
              <Card title="关联设备" size="small" style={{ marginBottom: 16 }}>
                <div style={{ lineHeight: 2 }}>
                  <div><strong>{detailData.device.brand} {detailData.device.model}</strong></div>
                  <div>序列号: {detailData.device.serial_number}</div>
                  <div>状态: {detailData.device.device_status}/{detailData.device.sub_status}</div>
                  <div>位置: {detailData.device.datacenter} / {detailData.device.cabinet}</div>
                </div>
              </Card>
            )}

            <Card title="操作详情" size="small" style={{ marginBottom: 16 }}>
              {(() => {
                const rd = parseRequestData(detailData.approval.request_data)
                return Object.entries(rd).map(([key, value]) => (
                  <div key={key} style={{ marginBottom: 4 }}>
                    <span style={{ color: '#666' }}>{key}:</span> {String(value)}
                  </div>
                ))
              })()}
            </Card>

            {detailData.approval.approve_remark && (
              <Card title="审批意见" size="small" style={{ marginBottom: 16 }}>
                {detailData.approval.approve_remark}
              </Card>
            )}

            {/* Action buttons */}
            <Space style={{ marginTop: 16 }}>
              {detailData.approval.status === 'pending' && (
                <>
                  <Button type="primary" onClick={() => handleApprove(detailData.approval.id)}>批准</Button>
                  <Popconfirm title="确认驳回？" onConfirm={() => handleReject(detailData.approval.id)}>
                    <Button danger>驳回</Button>
                  </Popconfirm>
                  <Input.TextArea
                    placeholder="审批意见(可选)"
                    value={rejectRemark}
                    onChange={e => setRejectRemark(e.target.value)}
                    rows={2}
                    style={{ width: 200 }}
                  />
                </>
              )}
              {detailData.approval.status === 'approved' && (
                <Popconfirm title="确认执行此操作？执行后设备状态将变更。" onConfirm={() => handleExecute(detailData.approval.id)}>
                  <Button type="primary">执行操作</Button>
                </Popconfirm>
              )}
              {detailData.approval.status === 'pending' && (
                <Popconfirm title="确认取消？" onConfirm={() => handleCancel(detailData.approval.id)}>
                  <Button>取消申请</Button>
                </Popconfirm>
              )}
            </Space>
          </>
        )}
      </Drawer>
    </div>
  )
}
