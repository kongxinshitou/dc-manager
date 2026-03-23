import { useEffect, useState, useCallback } from 'react'
import {
  Table, Button, Input, Select, Space, Tag, Modal, Form,
  DatePicker, message, Popconfirm, Card, Tooltip, Row, Col,
} from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { ColumnsType } from 'antd/es/table'
import {
  getInspections, createInspection, updateInspection, deleteInspection,
  getDevices,
} from '../api'
import type { Inspection, InspectionQuery, Device } from '../api'

const { Option } = Select
const { RangePicker } = DatePicker

const severityColor: Record<string, string> = { 严重: 'red', 一般: 'orange', 轻微: 'blue' }
const statusColor: Record<string, string> = { 待处理: 'red', 处理中: 'orange', 已解决: 'green' }

export default function Inspections() {
  const [data, setData] = useState<Inspection[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState<InspectionQuery>({ page: 1, page_size: 20 })
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Inspection | null>(null)
  const [deviceOptions, setDeviceOptions] = useState<Device[]>([])
  const [_deviceSearch, _setDeviceSearch] = useState('')
  const [form] = Form.useForm()
  const [searchForm] = Form.useForm()

  const fetchData = useCallback((q: InspectionQuery) => {
    setLoading(true)
    getInspections(q).then(res => {
      setData(res.data || [])
      setTotal(res.total)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  const searchDevices = (kw: string) => {
    getDevices({ keyword: kw, page_size: 20 }).then(res => setDeviceOptions(res.data || []))
  }

  useEffect(() => { fetchData(query) }, [])

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

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({ found_at: dayjs(), status: '待处理', severity: '一般' })
    setModalOpen(true)
  }

  const openEdit = (record: Inspection) => {
    setEditing(record)
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

  const handleSubmit = async () => {
    const values = await form.validateFields()
    const payload = {
      ...values,
      found_at: values.found_at?.toISOString(),
      resolved_at: values.resolved_at?.toISOString() ?? null,
      device_id: values.device_id ?? null,
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

  const columns: ColumnsType<Inspection> = [
    { title: 'ID', dataIndex: 'id', width: 60, fixed: 'left' },
    { title: '发现时间', dataIndex: 'found_at', width: 150,
      render: v => dayjs(v).format('YYYY-MM-DD HH:mm') },
    { title: '机房', dataIndex: 'datacenter', width: 120, ellipsis: true },
    { title: '机柜', dataIndex: 'cabinet', width: 80 },
    { title: 'U位', dataIndex: 'u_position', width: 80 },
    { title: '关联设备', key: 'device', width: 120,
      render: (_, r) => r.device ? `${r.device.brand} ${r.device.model}` : '-' },
    { title: '问题描述', dataIndex: 'issue', ellipsis: true },
    { title: '等级', dataIndex: 'severity', width: 70,
      render: v => <Tag color={severityColor[v]}>{v}</Tag> },
    { title: '状态', dataIndex: 'status', width: 80,
      render: v => <Tag color={statusColor[v]}>{v}</Tag> },
    { title: '巡检人', dataIndex: 'inspector', width: 80 },
    { title: '解决时间', dataIndex: 'resolved_at', width: 110,
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
        <Form layout="inline" form={searchForm} onFinish={handleSearch}>
          <Form.Item name="keyword"><Input placeholder="全局搜索" prefix={<SearchOutlined />} allowClear style={{ width: 150 }} /></Form.Item>
          <Form.Item name="datacenter"><Input placeholder="机房" allowClear style={{ width: 120 }} /></Form.Item>
          <Form.Item name="inspector"><Input placeholder="巡检人" allowClear style={{ width: 100 }} /></Form.Item>
          <Form.Item name="severity">
            <Select placeholder="等级" allowClear style={{ width: 90 }}>
              <Option value="严重">严重</Option>
              <Option value="一般">一般</Option>
              <Option value="轻微">轻微</Option>
            </Select>
          </Form.Item>
          <Form.Item name="status">
            <Select placeholder="状态" allowClear style={{ width: 100 }}>
              <Option value="待处理">待处理</Option>
              <Option value="处理中">处理中</Option>
              <Option value="已解决">已解决</Option>
            </Select>
          </Form.Item>
          <Form.Item name="time_range"><RangePicker style={{ width: 220 }} /></Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>查询</Button>
              <Button onClick={handleReset}>重置</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <div style={{ marginBottom: 8, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span>共 <strong>{total}</strong> 条记录</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增巡检记录</Button>
      </div>

      <Table
        dataSource={data}
        columns={columns}
        rowKey="id"
        size="small"
        loading={loading}
        scroll={{ x: 1400 }}
        pagination={{
          total, pageSize: query.page_size, current: query.page, showSizeChanger: true,
          pageSizeOptions: ['20', '50', '100'],
          onChange: (page, pageSize) => {
            const q = { ...query, page, page_size: pageSize }
            setQuery(q); fetchData(q)
          },
        }}
      />

      <Modal
        title={editing ? '编辑巡检记录' : '新增巡检记录'}
        open={modalOpen}
        onOk={handleSubmit}
        onCancel={() => setModalOpen(false)}
        width={720}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="机房" name="datacenter" rules={[{ required: true, message: '请填写机房' }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="机柜" name="cabinet"><Input /></Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="U位" name="u_position"><Input /></Form.Item>
            </Col>
          </Row>

          <Form.Item label="关联设备（可选，输入关键字搜索）" name="device_id">
            <Select
              showSearch
              allowClear
              placeholder="搜索设备..."
              filterOption={false}
              onSearch={searchDevices}
              notFoundContent={null}
            >
              {deviceOptions.map(d => (
                <Option key={d.id} value={d.id}>
                  [{d.datacenter} {d.cabinet}] {d.brand} {d.model} - {d.ip_address || d.serial_number}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="发现时间" name="found_at" rules={[{ required: true }]}>
                <DatePicker showTime style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="巡检人" name="inspector" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="问题描述" name="issue" rules={[{ required: true }]}>
            <Input.TextArea rows={3} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="问题等级" name="severity" rules={[{ required: true }]}>
                <Select>
                  <Option value="严重">严重</Option>
                  <Option value="一般">一般</Option>
                  <Option value="轻微">轻微</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="问题状态" name="status" rules={[{ required: true }]}>
                <Select>
                  <Option value="待处理">待处理</Option>
                  <Option value="处理中">处理中</Option>
                  <Option value="已解决">已解决</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
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
    </div>
  )
}
