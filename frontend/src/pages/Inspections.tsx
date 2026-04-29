import { useEffect, useState, useCallback, useRef } from 'react'
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  DatePicker, message, Popconfirm, Card, Tooltip, Row, Col, Typography, Upload, Grid
} from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, LinkOutlined, UploadOutlined, EyeOutlined, FilterOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { ColumnsType, TableProps } from 'antd/es/table'
import type { SorterResult } from 'antd/es/table/interface'
import ResponsiveTable from '../components/ResponsiveTable'
import {
  getInspections, createInspection, updateInspection, deleteInspection,
  getDeviceOptions, getDeviceCabinets, getDeviceByLocation, batchDeleteInspections,
  importInspectionsPreview, importInspectionsConfirm,
} from '../api'
import type { Inspection, InspectionQuery, Device } from '../api'

const { Option } = Select
const { RangePicker } = DatePicker
const { Text } = Typography

const severityColor: Record<string, string> = { 严重: 'red', 一般: 'orange', 轻微: 'blue' }
const statusColor: Record<string, string> = { 待处理: 'red', 处理中: 'orange', 已解决: 'green' }

interface InspectionsProps {
  onGoToDevice?: (id: number) => void
  onViewDetail?: (id: number) => void
  permissions?: Set<string>
}

export default function Inspections({ onGoToDevice, onViewDetail, permissions }: InspectionsProps) {
  const [data, setData] = useState<Inspection[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState<InspectionQuery>({ page: 1, page_size: 20 })
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Inspection | null>(null)
  const [datacenterOptions, setDatacenterOptions] = useState<string[]>([])
  const [cabinetOptions, setCabinetOptions] = useState<string[]>([])
  const [form] = Form.useForm()
  const [searchForm] = Form.useForm()
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  // Mobile filter panel toggle (only used when isMobile)
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false)

  // Import state
  const [importLoading, setImportLoading] = useState(false)
  const [importPreviewOpen, setImportPreviewOpen] = useState(false)
  const [importPreviewData, setImportPreviewData] = useState<Inspection[]>([])
  const [importCount, setImportCount] = useState(0)
  const importFileRef = useRef<File | null>(null)

  // Auto-found device in form
  const [foundDevice, setFoundDevice] = useState<Device | null | undefined>(undefined)
  const [lookingUp, setLookingUp] = useState(false)

  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  const fetchData = useCallback((q: InspectionQuery) => {
    setLoading(true)
    getInspections(q).then(res => {
      setData(res.data || [])
      setTotal(res.total)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetchData(query)
    getDeviceOptions().then(opts => setDatacenterOptions(opts.datacenters || []))
  }, [])

  const handleSearch = (values: any) => {
    const q: InspectionQuery = {
      ...query,
      ...values,
      page: 1,
      start_time: values.time_range?.[0]?.format('YYYY-MM-DD'),
      end_time: values.time_range?.[1]?.format('YYYY-MM-DD'),
    }
    delete (q as any).time_range
    setQuery(q)
    fetchData(q)
  }

  const handleReset = () => {
    searchForm.resetFields()
    const q: InspectionQuery = { page: 1, page_size: 20 }
    setQuery(q)
    fetchData(q)
  }

  const handleTableChange: TableProps<Inspection>['onChange'] = (_, __, sorter) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter as SorterResult<Inspection>
    const q: InspectionQuery = {
      ...query,
      page: 1,
      order_by: s.order ? (s.field as string) : undefined,
      sort: s.order === 'ascend' ? 'asc' : s.order === 'descend' ? 'desc' : undefined,
    }
    setQuery(q)
    fetchData(q)
  }

  // Load cabinets when datacenter changes
  const handleDatacenterChange = async (dc: string) => {
    setCabinetOptions([])
    form.setFieldValue('cabinet', undefined)
    setFoundDevice(undefined)
    if (dc) {
      const res = await getDeviceCabinets(dc).catch(() => ({ cabinets: [] }))
      setCabinetOptions(res.cabinets || [])
    }
  }

  // Lookup device when datacenter + cabinet + start_u/end_u change
  const triggerDeviceLookup = async () => {
    const datacenter = form.getFieldValue('datacenter')
    const cabinet = form.getFieldValue('cabinet') || ''
    const rawStartU = form.getFieldValue('start_u')
    const rawEndU = form.getFieldValue('end_u')
    const startU: number | null = rawStartU != null && rawStartU !== '' ? parseInt(rawStartU, 10) : null
    const endU: number | null = rawEndU != null && rawEndU !== '' ? parseInt(rawEndU, 10) : null
    if (!datacenter || startU == null) {
      setFoundDevice(undefined)
      return
    }
    setLookingUp(true)
    try {
      const res = await getDeviceByLocation(datacenter, cabinet, startU, endU ?? startU)
      setFoundDevice(res.device)
      form.setFieldValue('device_id', res.device?.id ?? null)
    } catch {
      setFoundDevice(undefined)
    } finally {
      setLookingUp(false)
    }
  }

  const openCreate = () => {
    setEditing(null)
    setFoundDevice(undefined)
    setCabinetOptions([])
    form.resetFields()
    form.setFieldsValue({ found_at: dayjs(), status: '待处理', severity: '一般' })
    setModalOpen(true)
  }

  const openEdit = async (record: Inspection) => {
    setEditing(record)
    setFoundDevice(record.device ?? undefined)
    if (record.datacenter) {
      const res = await getDeviceCabinets(record.datacenter).catch(() => ({ cabinets: [] }))
      setCabinetOptions(res.cabinets || [])
    }
    form.setFieldsValue({
      ...record,
      found_at: record.found_at ? dayjs(record.found_at) : dayjs(),
      resolved_at: record.resolved_at ? dayjs(record.resolved_at) : null,
      device_id: record.device_id ?? undefined,
    })
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    await deleteInspection(id)
    message.success('已删除')
    fetchData(query)
  }

  const handleBatchDelete = async () => {
    await batchDeleteInspections(selectedIds)
    message.success(`已删除 ${selectedIds.length} 条记录`)
    setSelectedIds([])
    fetchData(query)
  }

  const handleImportFile = async (file: File) => {
    if (!file.name.endsWith('.xlsx') && !file.name.endsWith('.xls')) {
      message.error('请上传 .xlsx 或 .xls 格式文件')
      return false
    }
    importFileRef.current = file
    setImportLoading(true)
    try {
      const res = await importInspectionsPreview(file)
      setImportPreviewData(res.preview || [])
      setImportCount(res.count)
      setImportPreviewOpen(true)
    } catch (err: any) {
      message.error(err.response?.data?.error || '文件解析失败')
    } finally {
      setImportLoading(false)
    }
    return false
  }

  const handleImportConfirm = async () => {
    if (!importFileRef.current) return
    setImportLoading(true)
    try {
      const res = await importInspectionsConfirm(importFileRef.current)
      message.success(res.message || `导入成功，新增${res.inserted}条`)
      setImportPreviewOpen(false)
      importFileRef.current = null
      fetchData(query)
    } catch (err: any) {
      message.error(err.response?.data?.error || '导入失败')
    } finally {
      setImportLoading(false)
    }
  }

  const handleSubmit = async () => {
    const values = await form.validateFields()
    const startU = values.start_u != null && values.start_u !== '' ? parseInt(values.start_u, 10) : null
    const endU = values.end_u != null && values.end_u !== '' ? parseInt(values.end_u, 10) : startU
    const payload = {
      ...values,
      found_at: values.found_at?.toISOString(),
      resolved_at: values.resolved_at?.toISOString() ?? null,
      device_id: values.device_id ?? null,
      start_u: startU,
      end_u: endU,
      u_position: startU != null
        ? endU != null && endU !== startU
          ? `${String(startU).padStart(2, '0')}-${String(endU).padStart(2, '0')}U`
          : `${String(startU).padStart(2, '0')}U`
        : values.u_position ?? '',
    }
    if (editing) {
      await updateInspection(editing.id, payload)
      message.success('更新成功')
    } else {
      await createInspection(payload)
      message.success('创建成功')
    }
    setModalOpen(false)
    fetchData(query)
  }

  const previewColumns: ColumnsType<Inspection> = [
    { title: '机房', dataIndex: 'datacenter', width: 100 },
    { title: '机柜', dataIndex: 'cabinet', width: 80 },
    { title: 'U位', dataIndex: 'u_position', width: 80 },
    { title: '巡检人', dataIndex: 'inspector', width: 80 },
    { title: '问题描述', dataIndex: 'issue', ellipsis: true },
    { title: '等级', dataIndex: 'severity', width: 70,
      render: (v: string) => <Tag color={severityColor[v]}>{v}</Tag> },
    { title: '状态', dataIndex: 'status', width: 80,
      render: (v: string) => <Tag color={statusColor[v]}>{v}</Tag> },
  ]

  const columns: ColumnsType<Inspection> = [
    { title: 'ID', dataIndex: 'id', width: 60, fixed: 'left', sorter: true },
    { title: '发现时间', dataIndex: 'found_at', width: 150, sorter: true,
      render: v => dayjs(v).format('YYYY-MM-DD HH:mm') },
    { title: '机房', dataIndex: 'datacenter', width: 120, ellipsis: true, sorter: true },
    { title: '机柜', dataIndex: 'cabinet', width: 80 },
    { title: 'U位', key: 'u_range', width: 90,
      render: (_: any, r: Inspection) => {
        if (r.start_u != null && r.end_u != null) {
          return r.start_u === r.end_u ? `${r.start_u}U` : `${r.start_u}-${r.end_u}U`
        }
        return r.u_position || '-'
      }
    },
    { title: '关联设备', key: 'device', width: 160,
      render: (_, r) => {
        if (!r.device) return '-'
        const label = `${r.device.brand} ${r.device.model}`
        if (onGoToDevice && r.device_id) {
          return (
            <Button
              type="link"
              size="small"
              icon={<LinkOutlined />}
              style={{ padding: 0, height: 'auto' }}
              onClick={() => onGoToDevice(r.device_id!)}
            >
              {label}
            </Button>
          )
        }
        return label
      }
    },
    { title: '问题描述', dataIndex: 'issue', ellipsis: true },
    { title: '等级', dataIndex: 'severity', width: 70, sorter: true,
      render: v => <Tag color={severityColor[v]}>{v}</Tag> },
    { title: '状态', dataIndex: 'status', width: 80, sorter: true,
      render: v => <Tag color={statusColor[v]}>{v}</Tag> },
    { title: '巡检人', dataIndex: 'inspector', width: 80, sorter: true },
    { title: '解决时间', dataIndex: 'resolved_at', width: 110, sorter: true,
      render: v => v ? dayjs(v).format('YYYY-MM-DD') : '-' },
    {
      title: '操作', fixed: 'right', width: 140,
      render: (_, record) => (
        <Space>
          {onViewDetail && (
            <Tooltip title="查看详情">
              <Button size="small" icon={<EyeOutlined />} onClick={() => onViewDetail(record.id)} />
            </Tooltip>
          )}
          <Tooltip title="编辑"><Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} /></Tooltip>
          {(!permissions || permissions.has('inspection:delete')) && (
            <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
              <Tooltip title="删除"><Button size="small" danger icon={<DeleteOutlined />} /></Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 16 }}>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Form layout={isMobile ? 'vertical' : 'inline'} form={searchForm} onFinish={handleSearch}>
          <Form.Item name="keyword"><Input placeholder="全局搜索" prefix={<SearchOutlined />} allowClear style={{ width: isMobile ? '100%' : 150 }} /></Form.Item>
          {(!isMobile || mobileFilterOpen) && (
            <>
              <Form.Item name="datacenter"><Input placeholder="机房" allowClear style={{ width: isMobile ? '100%' : 120 }} /></Form.Item>
              <Form.Item name="inspector"><Input placeholder="巡检人" allowClear style={{ width: isMobile ? '100%' : 100 }} /></Form.Item>
              <Form.Item name="severity">
                <Select placeholder="等级" allowClear style={{ width: isMobile ? '100%' : 90 }}>
                  <Option value="严重">严重</Option>
                  <Option value="一般">一般</Option>
                  <Option value="轻微">轻微</Option>
                </Select>
              </Form.Item>
              <Form.Item name="status">
                <Select placeholder="状态" allowClear style={{ width: isMobile ? '100%' : 100 }}>
                  <Option value="待处理">待处理</Option>
                  <Option value="处理中">处理中</Option>
                  <Option value="已解决">已解决</Option>
                </Select>
              </Form.Item>
              <Form.Item name="time_range"><RangePicker style={{ width: isMobile ? '100%' : 220 }} /></Form.Item>
            </>
          )}
          <Form.Item>
            <Space wrap>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>查询</Button>
              <Button onClick={handleReset}>重置</Button>
              {isMobile && (
                <Button
                  type="link"
                  icon={<FilterOutlined />}
                  onClick={() => setMobileFilterOpen(v => !v)}
                >
                  {mobileFilterOpen ? '收起' : '筛选'}
                </Button>
              )}
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          <span>共 <strong>{total}</strong> 条记录</span>
          {selectedIds.length > 0 && (
            <Popconfirm
              title={`确认删除选中的 ${selectedIds.length} 条记录？`}
              onConfirm={handleBatchDelete}
            >
              <Button danger size="small">批量删除 ({selectedIds.length})</Button>
            </Popconfirm>
          )}
        </Space>
        <Space>
          <Upload accept=".xlsx,.xls" showUploadList={false} beforeUpload={handleImportFile}>
            {isMobile ? (
              <Tooltip title="导入Excel">
                <Button icon={<UploadOutlined />} loading={importLoading} />
              </Tooltip>
            ) : (
              <Button icon={<UploadOutlined />} loading={importLoading}>导入Excel</Button>
            )}
          </Upload>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            {isMobile ? '新增' : '新增巡检记录'}
          </Button>
        </Space>
      </div>

      <ResponsiveTable<Inspection>
        dataSource={data}
        columns={columns}
        rowKey="id"
        size="small"
        loading={loading}
        scroll={{ x: 1400 }}
        rowSelection={{
          selectedRowKeys: selectedIds,
          onChange: keys => setSelectedIds(keys as number[]),
        }}
        onChange={handleTableChange}
        pagination={{
          total, pageSize: query.page_size, current: query.page, showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
          onChange: (page, pageSize) => {
            const q = { ...query, page, page_size: pageSize }
            setQuery(q); fetchData(q)
          },
        }}
        mobileCardRender={(record) => {
          const uText = record.start_u != null && record.end_u != null
            ? (record.start_u === record.end_u ? `${record.start_u}U` : `${record.start_u}-${record.end_u}U`)
            : (record.u_position || '-')
          return (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <Space size={4} wrap>
                  <Tag color={severityColor[record.severity]}>{record.severity}</Tag>
                  <Tag color={statusColor[record.status]}>{record.status}</Tag>
                </Space>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {dayjs(record.found_at).format('MM-DD HH:mm')}
                </Text>
              </div>
              <div style={{ fontSize: 13, marginBottom: 4 }}>
                <Text type="secondary">位置：</Text>
                {record.datacenter} / {record.cabinet || '-'} / {uText}
              </div>
              {record.device && (
                <div style={{ fontSize: 13, marginBottom: 4 }}>
                  <Text type="secondary">设备：</Text>
                  {onGoToDevice && record.device_id ? (
                    <Button
                      type="link"
                      size="small"
                      icon={<LinkOutlined />}
                      style={{ padding: 0, height: 'auto', fontSize: 13 }}
                      onClick={() => onGoToDevice(record.device_id!)}
                    >
                      {record.device.brand} {record.device.model}
                    </Button>
                  ) : (
                    <span>{record.device.brand} {record.device.model}</span>
                  )}
                </div>
              )}
              <div style={{ fontSize: 13, marginBottom: 4 }}>
                <Text type="secondary">巡检人：</Text>{record.inspector || '-'}
              </div>
              <div style={{ fontSize: 13, marginBottom: 8, wordBreak: 'break-word' }}>
                <Text type="secondary">问题：</Text>{record.issue}
              </div>
              <Space size={4} wrap>
                {onViewDetail && (
                  <Button size="small" icon={<EyeOutlined />} onClick={() => onViewDetail(record.id)}>详情</Button>
                )}
                <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>
                {(!permissions || permissions.has('inspection:delete')) && (
                  <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
                    <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
                  </Popconfirm>
                )}
              </Space>
            </div>
          )
        }}
      />

      <Modal
        title={editing ? '编辑巡检记录' : '新增巡检记录'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={isMobile ? '100%' : 720}
        style={isMobile ? { top: 0, maxWidth: '100vw', paddingBottom: 0 } : {}}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item label="机房" name="datacenter" rules={[{ required: true, message: '请选择机房' }]}>
                <Select
                  showSearch
                  allowClear
                  placeholder="选择机房"
                  onChange={(v) => { handleDatacenterChange(v); triggerDeviceLookup() }}
                >
                  {datacenterOptions.map(d => (
                    <Option key={d} value={d}>{d}</Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item label="机柜" name="cabinet">
                <Select
                  showSearch
                  allowClear
                  placeholder="选择机柜"
                  onChange={triggerDeviceLookup}
                  notFoundContent="请先选择机房"
                >
                  {cabinetOptions.map(c => (
                    <Option key={c} value={c}>{c}</Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item label="起始U" name="start_u">
                <Input type="number" min={1} placeholder="如 4" onChange={triggerDeviceLookup} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item label="结束U" name="end_u" extra="单个U位则与起始U相同">
                <Input type="number" min={1} placeholder="如 5（单U可不填）" onChange={triggerDeviceLookup} />
              </Form.Item>
            </Col>
          </Row>

          {/* Auto-found device display */}
          <Form.Item name="device_id" hidden><Input /></Form.Item>
          <div style={{ marginBottom: 16, padding: '8px 12px', background: '#f5f5f5', borderRadius: 6 }}>
            {lookingUp ? (
              <Text type="secondary">正在查找关联设备...</Text>
            ) : foundDevice ? (
              <Space>
                <Text type="success">✓ 已关联设备：</Text>
                {onGoToDevice ? (
                  <Button
                    type="link"
                    size="small"
                    icon={<LinkOutlined />}
                    style={{ padding: 0 }}
                    onClick={() => { setModalOpen(false); onGoToDevice(foundDevice.id) }}
                  >
                    [{foundDevice.datacenter} {foundDevice.cabinet} {foundDevice.u_position}] {foundDevice.brand} {foundDevice.model}
                  </Button>
                ) : (
                  <Text>[{foundDevice.datacenter} {foundDevice.cabinet} {foundDevice.u_position}] {foundDevice.brand} {foundDevice.model}</Text>
                )}
              </Space>
            ) : foundDevice === null ? (
              <Text type="secondary">未找到对应设备（机房/U位无匹配），device_id 将为空</Text>
            ) : (
              <Text type="secondary">填写机房和U位后自动关联设备</Text>
            )}
          </div>

          <Row gutter={16}>
            <Col xs={24} sm={12}>
              <Form.Item label="发现时间" name="found_at" rules={[{ required: true }]}>
                <DatePicker showTime style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col xs={24} sm={12}>
              <Form.Item label="巡检人" name="inspector" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="问题描述" name="issue" rules={[{ required: true }]}>
            <Input.TextArea rows={3} />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} sm={8}>
              <Form.Item label="问题等级" name="severity" rules={[{ required: true }]}>
                <Select>
                  <Option value="严重">严重</Option>
                  <Option value="一般">一般</Option>
                  <Option value="轻微">轻微</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item label="问题状态" name="status" rules={[{ required: true }]}>
                <Select>
                  <Option value="待处理">待处理</Option>
                  <Option value="处理中">处理中</Option>
                  <Option value="已解决">已解决</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col xs={24} sm={8}>
              <Form.Item label="解决时间" name="resolved_at">
                <DatePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="备注" name="remark">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`导入预览（共 ${importCount} 条记录，展示前 ${importPreviewData.length} 条）`}
        open={importPreviewOpen}
        onOk={handleImportConfirm}
        onCancel={() => { setImportPreviewOpen(false); importFileRef.current = null }}
        okText="确认导入"
        cancelText="取消"
        confirmLoading={importLoading}
        width={900}
      >
        <Table
          dataSource={importPreviewData}
          columns={previewColumns}
          rowKey={(_, i) => String(i)}
          size="small"
          pagination={false}
          scroll={{ x: 700 }}
        />
      </Modal>
    </div>
  )
}
