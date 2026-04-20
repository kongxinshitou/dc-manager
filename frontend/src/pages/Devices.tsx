import { useEffect, useState, useCallback, useRef } from 'react'
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  DatePicker, message, Popconfirm, Row, Col, Card, Tooltip, Upload, Grid,
  Dropdown,
} from 'antd'
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined,
  DownloadOutlined, UploadOutlined, MoreOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import type { ColumnsType, TableProps } from 'antd/es/table'
import type { SorterResult } from 'antd/es/table/interface'
import {
  getDevices, getDevice, createDevice, updateDevice, deleteDevice,
  getDeviceOptions, batchDeleteDevices, exportDevices,
  importDevicesPreview, importDevicesConfirm, operateDevice,
  submitApproval, getDatacenters, getDatacenterCabinets,
} from '../api'
import type { Device, DeviceQuery, DeviceWithWarranty, Datacenter, Cabinet } from '../api'
import ResponsiveTable from '../components/ResponsiveTable'

const { Option } = Select

const statusColor: Record<string, string> = {
  Online: 'green', online: 'green', onlion: 'green', Offline: 'red', offline: 'red',
}

const deviceStatusLabel: Record<string, string> = {
  in_stock: '入库', out_stock: '出库',
}

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

interface DevicesProps {
  focusDeviceId?: number | null
  onFocusHandled?: () => void
}

export default function Devices({ focusDeviceId, onFocusHandled }: DevicesProps) {
  const [data, setData] = useState<DeviceWithWarranty[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState<DeviceQuery>({ page: 1, page_size: 20 })
  const [options, setOptions] = useState<any>({})
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Device | null>(null)
  const [form] = Form.useForm()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  // Operation modal
  const [opModalOpen, setOpModalOpen] = useState(false)
  const [opType, setOpType] = useState<string>('')
  const [opDevice, setOpDevice] = useState<Device | null>(null)
  const [opForm] = Form.useForm()

  // Batch selection
  const [selectedIds, setSelectedIds] = useState<number[]>([])

  // Datacenter & cabinet options for rack form
  const [dcList, setDcList] = useState<Datacenter[]>([])
  const [cabinetList, setCabinetList] = useState<Cabinet[]>([])
  const [selectedDcId, setSelectedDcId] = useState<number | null>(null)

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
    getDatacenters().then(setDcList)
  }, [])

  useEffect(() => {
    if (focusDeviceId) {
      getDevice(focusDeviceId).then(res => {
        openEdit(res.device)
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

  const handleTableChange: TableProps<DeviceWithWarranty>['onChange'] = (pagination, __, sorter) => {
    const s = Array.isArray(sorter) ? sorter[0] : sorter as SorterResult<DeviceWithWarranty>
    const q: DeviceQuery = {
      ...query,
      page: pagination.current || 1,
      page_size: pagination.pageSize || query.page_size,
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
      arrival_date: record.arrival_date ? dayjs(record.arrival_date) : null,
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
      arrival_date: values.arrival_date?.toISOString() ?? null,
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

  // Device operation handlers
  const openOpModal = (device: DeviceWithWarranty, operation: string) => {
    setOpDevice(device)
    setOpType(operation)
    opForm.resetFields()
    setCabinetList([])
    setSelectedDcId(null)
    setOpModalOpen(true)
  }

  const handleRackDcChange = async (dcId: number) => {
    setSelectedDcId(dcId)
    setCabinetList([])
    const dc = dcList.find(d => d.id === dcId)
    if (dc) opForm.setFieldsValue({ datacenter: dc.name })
    try {
      const cabs = await getDatacenterCabinets(dcId)
      setCabinetList(cabs || [])
    } catch { setCabinetList([]) }
  }

  const handleRackCabChange = (cabId: number) => {
    const cab = cabinetList.find(c => c.id === cabId)
    if (cab) opForm.setFieldsValue({ cabinet: cab.name, cabinet_id: cab.id })
  }

  const handleOpSubmit = async () => {
    if (!opDevice) return
    const values = await opForm.validateFields()
    try {
      // Submit via approval flow
      await submitApproval({
        device_id: opDevice.id,
        operation_type: opType,
        request_data: values,
      })
      message.success('已提交审批申请')
      setOpModalOpen(false)
      fetchData(query)
    } catch (err: any) {
      message.error(err.response?.data?.error || '提交失败')
    }
  }

  const getAvailableOps = (record: DeviceWithWarranty) => {
    const ops: { key: string; label: string }[] = []
    if (!record.device_status) {
      ops.push({ key: 'in_stock_new', label: '入库(新购)' })
      ops.push({ key: 'in_stock_recycle', label: '入库(回收)' })
    }
    if (record.device_status === 'in_stock') {
      ops.push({ key: 'rack', label: '上架' })
      ops.push({ key: 'dispatch', label: '外发' })
      ops.push({ key: 'scrap', label: '报废' })
    }
    if (record.device_status === 'out_stock' && record.sub_status === 'racked') {
      ops.push({ key: 'unrack', label: '下架' })
    }
    if (record.device_status === 'out_stock' && record.sub_status === 'unracked') {
      ops.push({ key: 'in_stock_recycle', label: '确认回收入库' })
    }
    return ops
  }

  const opTitle: Record<string, string> = {
    in_stock_new: '入库操作(新购)', in_stock_recycle: '入库操作(回收)',
    rack: '上架操作', dispatch: '外发操作', scrap: '报废操作', unrack: '下架操作',
  }

  const previewColumns: ColumnsType<Device> = [
    { title: '来源', dataIndex: 'source', width: 80 },
    { title: '品牌', dataIndex: 'brand', width: 80 },
    { title: '型号', dataIndex: 'model', width: 100, ellipsis: true },
    { title: '机房', dataIndex: 'datacenter', width: 100 },
    { title: 'IP地址', dataIndex: 'ip_address', width: 120 },
    { title: '序列号', dataIndex: 'serial_number', width: 150, ellipsis: true },
  ]

  const columns: ColumnsType<DeviceWithWarranty> = [
    { title: 'ID', dataIndex: 'id', width: 60, fixed: 'left', sorter: true },
    { title: '来源', dataIndex: 'source', width: 90 },
    { title: '设备状态', width: 100, sorter: true,
      render: (_, r) => {
        const main = deviceStatusLabel[r.device_status] || r.device_status
        const sub = subStatusLabel[r.sub_status] || r.sub_status
        const color = r.device_status === 'in_stock' ? 'blue' : r.device_status === 'out_stock' ? 'orange' : 'default'
        return <Tag color={color}>{main}-{sub}</Tag>
      }
    },
    { title: '在保状态', dataIndex: 'warranty_status', width: 80,
      render: v => <Tag color={warrantyColor[v] || 'default'}>{warrantyLabel[v] || v || '未知'}</Tag>
    },
    { title: '机房', dataIndex: 'datacenter', width: 120, ellipsis: true, sorter: true },
    { title: '机柜', dataIndex: 'cabinet', width: 80 },
    { title: 'U位', dataIndex: 'u_position', width: 80 },
    { title: '品牌', dataIndex: 'brand', width: 70, sorter: true },
    { title: '型号', dataIndex: 'model', width: 100, ellipsis: true },
    { title: '类型', dataIndex: 'device_type', width: 80, sorter: true },
    { title: '序列号', dataIndex: 'serial_number', width: 160, ellipsis: true },
    { title: 'IP地址', dataIndex: 'ip_address', width: 130, sorter: true },
    { title: '管理IP', dataIndex: 'mgmt_ip', width: 130 },
    { title: '合同号', dataIndex: 'contract_no', width: 100 },
    { title: '财务编号(旧)', dataIndex: 'finance_no', width: 100 },
    { title: '责任人', dataIndex: 'custodian', width: 80 },
    { title: '维保截止', dataIndex: 'warranty_end', width: 100,
      render: v => v ? dayjs(v).format('YYYY-MM-DD') : '-' },
    {
      title: '操作', fixed: 'right', width: 200,
      render: (_, record) => {
        const ops = getAvailableOps(record)
        return (
          <Space size={4} wrap>
            <Tooltip title="编辑"><Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)} /></Tooltip>
            <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
              <Tooltip title="删除"><Button size="small" danger icon={<DeleteOutlined />} /></Tooltip>
            </Popconfirm>
            {ops.slice(0, 2).map(o => (
              <Button key={o.key} size="small" type={o.key === 'rack' ? 'primary' : 'default'} onClick={() => openOpModal(record, o.key)}>
                {o.label}
              </Button>
            ))}
            {ops.length > 2 && (
              <Dropdown menu={{ items: ops.slice(2).map(o => ({ key: o.key, label: o.label, onClick: () => openOpModal(record, o.key) })) }}>
                <Button size="small" icon={<MoreOutlined />} />
              </Dropdown>
            )}
          </Space>
        )
      },
    },
  ]

  // Operation form fields based on type
  const renderOpForm = () => {
    switch (opType) {
      case 'rack':
        return (
          <>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item label="机房" name="datacenter_id" rules={[{ required: true, message: '请选择机房' }]}>
                  <Select placeholder="请选择机房" onChange={(v: number) => handleRackDcChange(v)}>
                    {dcList.map(dc => <Option key={dc.id} value={dc.id}>{dc.name}</Option>)}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item label="机柜" name="cabinet_id" rules={[{ required: true, message: '请选择机柜' }]}>
                  <Select placeholder={selectedDcId ? '请选择机柜' : '请先选择机房'} onChange={(v: number) => handleRackCabChange(v)} disabled={!selectedDcId}>
                    {cabinetList.map(cab => <Option key={cab.id} value={cab.id}>{cab.name}</Option>)}
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="datacenter" hidden><Input /></Form.Item>
            <Form.Item name="cabinet" hidden><Input /></Form.Item>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="起始U位" name="start_u" rules={[{ required: true }]}><Input type="number" /></Form.Item></Col>
              <Col span={12}><Form.Item label="占用U数" name="u_count" rules={[{ required: true }]}><Input type="number" /></Form.Item></Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="申请人" name="applicant"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="责任人" name="custodian"><Input /></Form.Item></Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="项目名称" name="project_name"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="所属业务" name="business_unit"><Input /></Form.Item></Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="所属部门" name="department"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="管理IP" name="mgmt_ip"><Input /></Form.Item></Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="业务地址" name="business_address"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="VIP地址" name="vip_address"><Input /></Form.Item></Col>
            </Row>
            <Form.Item label="操作系统" name="os"><Input /></Form.Item>
          </>
        )
      case 'dispatch':
        return (
          <>
            <Form.Item label="外发地址" name="dispatch_address" rules={[{ required: true }]}><Input /></Form.Item>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="外发保管人" name="dispatch_custodian"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="申请人" name="applicant"><Input /></Form.Item></Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}><Form.Item label="项目名称" name="project_name"><Input /></Form.Item></Col>
              <Col span={12}><Form.Item label="所属业务" name="business_unit"><Input /></Form.Item></Col>
            </Row>
            <Form.Item label="所属部门" name="department"><Input /></Form.Item>
          </>
        )
      case 'scrap':
        return (
          <Form.Item label="报废备注" name="scrap_remark"><Input.TextArea rows={3} /></Form.Item>
        )
      case 'in_stock_new':
      case 'in_stock_recycle':
        return (
          <>
            <Form.Item label="存放位置" name="storage_location" rules={[{ required: true }]}>
              <Input placeholder="如：仓库A区3号架" />
            </Form.Item>
            <Form.Item label="责任人" name="custodian">
              <Input placeholder="默认责任人可到系统配置中设置" />
            </Form.Item>
          </>
        )
      case 'unrack':
        return (
          <>
            <Form.Item label="存放位置" name="storage_location" rules={[{ required: true }]}><Input /></Form.Item>
            <Form.Item label="责任人" name="custodian"><Input /></Form.Item>
            <div style={{ color: '#faad14', marginBottom: 16 }}>
              下架后设备将自动变为「入库-回收」状态
            </div>
          </>
        )
      default:
        return null
    }
  }

  return (
    <div style={{ padding: 16 }}>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Form layout="inline" onFinish={handleSearch}>
          <Form.Item name="keyword"><Input placeholder="全局搜索" prefix={<SearchOutlined />} allowClear style={{ width: 160 }} /></Form.Item>
          <Form.Item name="device_status">
            <Select placeholder="设备状态" allowClear style={{ width: 110 }}>
              <Option value="in_stock">入库</Option>
              <Option value="out_stock">出库</Option>
            </Select>
          </Form.Item>
          <Form.Item name="sub_status">
            <Select placeholder="子状态" allowClear style={{ width: 100 }}>
              <Option value="new_purchase">新购</Option>
              <Option value="recycled">回收</Option>
              <Option value="racked">上架</Option>
              <Option value="dispatched">外发</Option>
              <Option value="scrapped">报废</Option>
            </Select>
          </Form.Item>
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
          <Form.Item name="brand">
            <Select placeholder="品牌" allowClear style={{ width: 90 }}>
              {(options.brands || []).map((b: string) => <Option key={b} value={b}>{b}</Option>)}
            </Select>
          </Form.Item>
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
        scroll={{ x: 2000 }}
        rowSelection={!isMobile ? {
          selectedRowKeys: selectedIds,
          onChange: keys => setSelectedIds(keys as number[]),
        } : undefined}
        onChange={handleTableChange}
        pagination={{
          total, pageSize: query.page_size, current: query.page, showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
        }}
        mobileCardRender={(record) => (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <strong>{record.brand} {record.model}</strong>
              <Tag color={record.device_status === 'in_stock' ? 'blue' : 'orange'}>
                {deviceStatusLabel[record.device_status]}-{subStatusLabel[record.sub_status]}
              </Tag>
            </div>
            <div style={{ fontSize: 12, color: '#666', lineHeight: 1.8 }}>
              <div>{record.datacenter} / {record.cabinet} / {record.u_position}</div>
              <div>IP: {record.ip_address} | SN: {record.serial_number}</div>
            </div>
            <div style={{ marginTop: 8 }}>
              <Space size={4} wrap>
                <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>
                <Popconfirm title="确认删除？" onConfirm={() => handleDelete(record.id)}>
                  <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
                </Popconfirm>
                {getAvailableOps(record).slice(0, 1).map(o => (
                  <Button key={o.key} size="small" type="primary" onClick={() => openOpModal(record, o.key)}>{o.label}</Button>
                ))}
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

      {/* Operation Modal (rack/dispatch/scrap/unrack) */}
      <Modal
        title={opTitle[opType] || '操作'}
        open={opModalOpen}
        onOk={handleOpSubmit}
        onCancel={() => setOpModalOpen(false)}
        okText="确认执行"
        cancelText="取消"
        width={600}
        destroyOnClose
      >
        {opDevice && (
          <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
            <strong>{opDevice.brand} {opDevice.model}</strong>
            <span style={{ marginLeft: 8, color: '#666' }}>
              SN: {opDevice.serial_number} | {opDevice.datacenter} / {opDevice.cabinet}
            </span>
          </div>
        )}
        <Form form={opForm} layout="vertical">
          {renderOpForm()}
        </Form>
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
            <Col xs={24} sm={8}><Form.Item label="资产编号" name="asset_number"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="厂商" name="vendor"><Input /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="机房" name="datacenter"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="机柜号" name="cabinet"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}>
              <Form.Item label="U位置" name="u_position" extra="格式如 04U 或 04-05U">
                <Input placeholder="如 04U 或 04-05U" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="品牌" name="brand"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="型号" name="model"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="设备类型" name="device_type"><Select allowClear><Option value="服务器">服务器</Option><Option value="存储">存储</Option><Option value="网络">网络</Option><Option value="其他">其他</Option></Select></Form.Item></Col>
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
            <Col xs={24} sm={8}><Form.Item label="到货日期" name="arrival_date"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="维保年限" name="warranty_years"><Input type="number" placeholder="年" /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="维保起始时间" name="warranty_start"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="维保结束时间" name="warranty_end"><DatePicker style={{ width: '100%' }} /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="合同号" name="contract_no"><Input /></Form.Item></Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} sm={8}><Form.Item label="财务编号(旧)" name="finance_no"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="责任人" name="custodian"><Input /></Form.Item></Col>
            <Col xs={24} sm={8}><Form.Item label="存放位置" name="storage_location"><Input /></Form.Item></Col>
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
