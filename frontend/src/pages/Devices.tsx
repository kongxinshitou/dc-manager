import { useEffect, useState, useCallback, useRef } from 'react'
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  DatePicker, message, Popconfirm, Row, Col, Card, Tooltip, Upload, Grid
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined,
  DownloadOutlined, UploadOutlined
} from '@ant-design/icons'
import dayjs from 'dayjs'
import type { ColumnsType, TableProps } from 'antd/es/table'
import type { SorterResult } from 'antd/es/table/interface'
import {
  getDevices, getDevice, createDevice, updateDevice, deleteDevice,
  getDeviceOptions, batchDeleteDevices, exportDevices,
  importDevicesPreview, importDevicesConfirm,
} from '../api'
import type { Device, DeviceQuery } from '../api'
import ResponsiveTable from '../components/ResponsiveTable'

const { Option } = Select

const statusColor: Record<string, string> = {
  Online: 'green', online: 'green', onlion: 'green', Offline: 'red', offline: 'red',
}

interface DevicesProps {
  focusDeviceId?: number | null
  onFocusHandled?: () => void
}

export default function Devices({ focusDeviceId, onFocusHandled }: DevicesProps) {
  const [data, setData] = useState<Device[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState<DeviceQuery>({ page: 1, page_size: 20 })
  const [options, setOptions] = useState<any>({})
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Device | null>(null)
  const [form] = Form.useForm()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  // Batch selection
  const [selectedIds, setSelectedIds] = useState<number[]>([])

  // Import state
  const [importLoading, setImportLoading] = useState(false)
  const [importPreviewOpen, setImportPreviewOpen] = useState(false)
  const [importPreviewData, setImportPreviewData] = useState<Device[]>([])
  const [importCount, setImportCount] = useState(0)
  const importFileRef = useRef<File | null>(null)

  const fetchOptions = useCallback(() => {
    getDeviceOptions().then(setOptions)
  }, [])

  const fetchData = useCallback((q: DeviceQuery) => {
    setLoading(true)
    getDevices(q).then(res => {
      setData(res.data || [])
      setTotal(res.total)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  useEffect(() => {
    fetchOptions()
    fetchData(query)
  }, [])

  // When a device ID is passed from outside (e.g., navigation from Inspections), open its edit modal
  useEffect(() => {
    if (focusDeviceId) {
      getDevice(focusDeviceId).then(device => {
        openEdit(device)
        onFocusHandled?.()
      }).catch(() => onFocusHandled?.())
    }
  }, [focusDeviceId])

  const handleSearch = (values: any) => {
    const q = { ...query, ...values, page: 1 }
    setQuery(q)
    fetchData(q)
  }

  const handleReset = () => {
    const q: DeviceQuery = { page: 1, page_size: 20 }
    setQuery(q)
    fetchData(q)
  }

  const handleTableChange: TableProps<Device>['onChange'] = (_, __, sorter) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter as SorterResult<Device>
    const q: DeviceQuery = {
      ...query,
      page: 1,
      order_by: s.order ? (s.field as string) : undefined,
      sort: s.order === 'ascend' ? 'asc' : s.order === 'descend' ? 'desc' : undefined,
    }
    setQuery(q)
    fetchData(q)
  }

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (record: Device) => {
    setEditing(record)
    form.setFieldsValue({
      ...record,
      warranty_start: record.warranty_start ? dayjs(record.warranty_start) : null,
      warranty_end: record.warranty_end ? dayjs(record.warranty_end) : null,
      manufacture_date: record.manufacture_date ? dayjs(record.manufacture_date) : null,
    })
    setModalOpen(true)
  }

  const handleDelete = async (id: number) => {
    await deleteDevice(id)
    message.success('已删除')
    fetchData(query)
  }

  const handleBatchDelete = async () => {
    await batchDeleteDevices(selectedIds)
    message.success(`已删除 ${selectedIds.length} 条记录`)
    setSelectedIds([])
    fetchData(query)
  }

  const handleSubmit = async () => {
    const values = await form.validateFields()
    const payload = {
      ...values,
      warranty_start: values.warranty_start?.toISOString() ?? null,
      warranty_end: values.warranty_end?.toISOString() ?? null,
      manufacture_date: values.manufacture_date?.toISOString() ?? null,
    }
    if (editing) {
      await updateDevice(editing.id, payload)
      message.success('更新成功')
    } else {
      await createDevice(payload)
      message.success('创建成功')
    }
    setModalOpen(false)
    fetchData(query)
    fetchOptions()
  }

  const handleExport = async () => {
    try {
      const blob = await exportDevices(query)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'devices.xlsx'
      a.click()
      URL.revokeObjectURL(url)
    } catch {
      message.error('导出失败')
    }
  }

  const handleImportFile = async (file: File) => {
    if (!file.name.endsWith('.xlsx') && !file.name.endsWith('.xls')) {
      message.error('请上传 .xlsx 格式文件')
      return false
    }
    importFileRef.current = file
    setImportLoading(true)
    try {
      const res = await importDevicesPreview(file)
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
      const res = await importDevicesConfirm(importFileRef.current)
      message.success(res.message || `导入成功，新增${res.inserted}条，跳过${res.skipped}条重复`)
      setImportPreviewOpen(false)
      importFileRef.current = null
      fetchData(query)
      fetchOptions()
    } catch (err: any) {
      message.error(err.response?.data?.error || '导入失败')
    } finally {
      setImportLoading(false)
    }
  }

  const previewColumns: ColumnsType<Device> = [
    { title: '来源', dataIndex: 'source', width: 80 },
    { title: '品牌', dataIndex: 'brand', width: 80 },
    { title: '型号', dataIndex: 'model', width: 100, ellipsis: true },
    { title: '机房', dataIndex: 'datacenter', width: 100 },
    { title: 'IP地址', dataIndex: 'ip_address', width: 120 },
    { title: '序列号', dataIndex: 'serial_number', width: 150, ellipsis: true },
  ]

  const columns: ColumnsType<Device> = [
    { title: 'ID', dataIndex: 'id', width: 60, fixed: 'left', sorter: true },
    { title: '来源', dataIndex: 'source', width: 90 },
    { title: '状态', dataIndex: 'status', width: 70, sorter: true,
      render: v => v ? <Tag color={statusColor[v] || 'default'}>{v}</Tag> : '-' },
    { title: '机房', dataIndex: 'datacenter', width: 120, ellipsis: true, sorter: true },
    { title: '机柜', dataIndex: 'cabinet', width: 80 },
    { title: 'U位', dataIndex: 'u_position', width: 80 },
    { title: '起始U', dataIndex: 'start_u', width: 60,
      render: (v: number | null) => v != null ? v : '-' },
    { title: '结束U', dataIndex: 'end_u', width: 60,
      render: (v: number | null) => v != null ? v : '-' },
    { title: '品牌', dataIndex: 'brand', width: 70, sorter: true },
    { title: '型号', dataIndex: 'model', width: 100, ellipsis: true },
    { title: '类型', dataIndex: 'device_type', width: 80, sorter: true },
    { title: '序列号', dataIndex: 'serial_number', width: 160, ellipsis: true },
    { title: 'IP地址', dataIndex: 'ip_address', width: 130, sorter: true },
    { title: '管理IP', dataIndex: 'mgmt_ip', width: 130 },
    { title: '操作系统', dataIndex: 'os', width: 90, ellipsis: true },
    { title: '用途', dataIndex: 'purpose', width: 120, ellipsis: true },
    { title: '责任人', dataIndex: 'owner', width: 80, sorter: true },
    { title: '维保截止', dataIndex: 'warranty_end', width: 100,
      render: v => v ? dayjs(v).format('YYYY-MM-DD') : '-' },
    {
      title: '操作', fixed: 'right', width: 90,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑"><Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} /></Tooltip>
          <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
            <Tooltip title="删除"><Button size="small" danger icon={<DeleteOutlined />} /></Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: 16 }}>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Form layout="inline" onFinish={handleSearch}>
          <Form.Item name="keyword"><Input placeholder="全局搜索" prefix={<SearchOutlined />} allowClear style={{ width: 160 }} /></Form.Item>
          <Form.Item name="datacenter">
            <Select placeholder="机房" allowClear style={{ width: 130 }}>
              {(options.datacenters || []).map((d: string) => <Option key={d} value={d}>{d}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="source">
            <Select placeholder="来源区域" allowClear style={{ width: 110 }}>
              {(options.sources || []).map((s: string) => <Option key={s} value={s}>{s}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="device_type">
            <Select placeholder="设备类型" allowClear style={{ width: 100 }}>
              {(options.device_types || []).map((t: string) => <Option key={t} value={t}>{t}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="brand">
            <Select placeholder="品牌" allowClear style={{ width: 90 }}>
              {(options.brands || []).map((b: string) => <Option key={b} value={b}>{b}</Option>)}
            </Select>
          </Form.Item>
          <Form.Item name="ip_address"><Input placeholder="IP地址" allowClear style={{ width: 130 }} /></Form.Item>
          <Form.Item name="owner"><Input placeholder="责任人" allowClear style={{ width: 100 }} /></Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>查询</Button>
              <Button onClick={handleReset}>重置</Button>
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
          <Button icon={<DownloadOutlined />} onClick={handleExport}>导出Excel</Button>
          <Upload
            accept=".xlsx,.xls"
            showUploadList={false}
            beforeUpload={handleImportFile}
          >
            <Button icon={<UploadOutlined />} loading={importLoading}>导入Excel</Button>
          </Upload>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增设备</Button>
        </Space>
      </div>

      <ResponsiveTable
        dataSource={data}
        columns={columns}
        rowKey="id"
        size="small"
        loading={loading}
        scroll={{ x: 1800 }}
        rowSelection={!isMobile ? {
          selectedRowKeys: selectedIds,
          onChange: keys => setSelectedIds(keys as number[]),
        } : undefined}
        onChange={handleTableChange}
        pagination={{
          total, pageSize: query.page_size, current: query.page, showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
          onChange: (page, pageSize) => {
            const q = { ...query, page, page_size: pageSize }
            setQuery(q); fetchData(q)
          },
        }}
        mobileCardRender={(record) => (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <strong>{record.brand} {record.model}</strong>
              <Tag color={statusColor[record.status] || 'default'}>{record.status}</Tag>
            </div>
            <div style={{ fontSize: 12, color: '#666', lineHeight: 1.8 }}>
              <div>{record.datacenter} / {record.cabinet} / {record.u_position}</div>
              <div>IP: {record.ip_address} | SN: {record.serial_number}</div>
              <div>类型: {record.device_type} | 责任人: {record.owner}</div>
            </div>
            <div style={{ marginTop: 8 }}>
              <Space>
                <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>
                <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
                  <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
                </Popconfirm>
              </Space>
            </div>
          </div>
        )}
      />

      {/* Import Preview Modal */}
      <Modal
        title={`导入预览（共 ${importCount} 条记录，展示前 ${importPreviewData.length} 条）`}
        open={importPreviewOpen}
        onOk={handleImportConfirm}
        onCancel={() => { setImportPreviewOpen(false); importFileRef.current = null }}
        okText="确认导入"
        cancelText="取消"
        confirmLoading={importLoading}
        width={800}
      >
        <Table
          dataSource={importPreviewData}
          columns={previewColumns}
          rowKey={(_, i) => String(i)}
          size="small"
          pagination={false}
          scroll={{ x: 600 }}
        />
      </Modal>

      {/* Create / Edit Modal */}
      <Modal
        title={editing ? '编辑设备' : '新增设备'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={isMobile ? '100%' : 800}
        style={isMobile ? { top: 0, maxWidth: '100vw', paddingBottom: 0 } : {}}
        destroyOnClose
      >
        <Form form={form} layout="vertical" style={{ maxHeight: '60vh', overflowY: 'auto' }}>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="来源区域" name="source"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="状态" name="status"><Select allowClear><Option value="Online">Online</Option><Option value="Offline">Offline</Option></Select></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="资产编号" name="asset_number"><Input /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="机房" name="datacenter" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="机柜号" name="cabinet"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}>
              <Form.Item label="U位置" name="u_position" extra="格式如 04U 或 04-05U">
                <Input placeholder="如 04U 或 04-05U" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="品牌" name="brand" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="型号" name="model" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="设备类型" name="device_type" rules={[{ required: true }]}><Select><Option value="服务器">服务器</Option><Option value="存储">存储</Option><Option value="网络">网络</Option><Option value="其他">其他</Option></Select></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="序列号" name="serial_number"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="操作系统" name="os"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="IP地址" name="ip_address"><Input /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="远程管理IP" name="mgmt_ip"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="管理口账号" name="mgmt_account"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="系统账号密码" name="system_account"><Input /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="设备出厂时间" name="manufacture_date"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="维保起始时间" name="warranty_start"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="维保结束时间" name="warranty_end"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={12}><Form.Item label="设备用途" name="purpose"><Input /></Form.Item></Col>
            <Col xs={24} sm={12}><Form.Item label="责任人" name="owner"><Input /></Form.Item></Col>
          </Row>
          <Form.Item label="备注" name="remark"><Input.TextArea rows={2} /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
