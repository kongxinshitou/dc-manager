import { useEffect, useState, useCallback } from 'react'
import {
  Card, Button, Input, Space, Modal, Form, List, Select, message, Row, Col,
  InputNumber, Popconfirm, Grid, Tag,
} from 'antd'
import {
  PlusOutlined, DeleteOutlined, EditOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import {
  getDatacenters, createDatacenter, updateDatacenter, deleteDatacenter,
  getDatacenterColumns, setDatacenterColumns, getDatacenterRows, setDatacenterRows,
  getDatacenterCabinets, generateCabinets,
} from '../api'
import type { Datacenter, CabinetColumn, CabinetRow, Cabinet } from '../api'

const { Option } = Select

export default function DatacenterManage() {
  const [datacenters, setDatacenters] = useState<Datacenter[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [columns, setColumns] = useState<CabinetColumn[]>([])
  const [rows, setRows] = useState<CabinetRow[]>([])
  const [cabinets, setCabinets] = useState<Cabinet[]>([])
  const [dcModalOpen, setDcModalOpen] = useState(false)
  const [editingDc, setEditingDc] = useState<Datacenter | null>(null)
  const [dcForm] = Form.useForm()
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  const selected = datacenters.find(d => d.id === selectedId)

  const fetchDatacenters = useCallback(() => {
    getDatacenters().then(setDatacenters)
  }, [])

  useEffect(() => { fetchDatacenters() }, [fetchDatacenters])

  useEffect(() => {
    if (selectedId) {
      getDatacenterColumns(selectedId).then(setColumns)
      getDatacenterRows(selectedId).then(setRows)
      getDatacenterCabinets(selectedId).then(setCabinets)
    } else {
      setColumns([]); setRows([]); setCabinets([])
    }
  }, [selectedId])

  const handleSaveDc = async () => {
    const values = await dcForm.validateFields()
    if (editingDc) {
      await updateDatacenter(editingDc.id, values)
      message.success('更新成功')
    } else {
      await createDatacenter(values)
      message.success('创建成功')
    }
    setDcModalOpen(false)
    fetchDatacenters()
  }

  const handleDeleteDc = async (id: number) => {
    try {
      await deleteDatacenter(id)
      message.success('已删除')
      if (selectedId === id) setSelectedId(null)
      fetchDatacenters()
    } catch (err: any) {
      message.error(err.response?.data?.error || '删除失败')
    }
  }

  const handleSaveColumns = async () => {
    if (!selectedId) return
    await setDatacenterColumns(selectedId, columns)
    message.success('列配置已保存')
  }

  const handleSaveRows = async () => {
    if (!selectedId) return
    await setDatacenterRows(selectedId, rows)
    message.success('行配置已保存')
  }

  const handleGenerate = async () => {
    if (!selectedId) return
    try {
      const res = await generateCabinets(selectedId, selected?.max_u || 47)
      message.success(`已生成 ${res.generated} 个机柜`)
      getDatacenterCabinets(selectedId).then(setCabinets)
    } catch (err: any) {
      message.error(err.response?.data?.error || '生成失败')
    }
  }

  const addColumn = () => {
    setColumns([...columns, { name: `列${columns.length + 1}`, sort_order: columns.length, column_type: 'cabinet' }])
  }

  const removeColumn = (index: number) => {
    setColumns(columns.filter((_, i) => i !== index))
  }

  const updateColumn = (index: number, field: keyof CabinetColumn, value: any) => {
    const newCols = [...columns]
    newCols[index] = { ...newCols[index], [field]: value }
    setColumns(newCols)
  }

  const addRow = () => {
    setRows([...rows, { name: `${rows.length + 1}`, sort_order: rows.length }])
  }

  const removeRow = (index: number) => {
    setRows(rows.filter((_, i) => i !== index))
  }

  const updateRow = (index: number, field: keyof CabinetRow, value: any) => {
    const newRows = [...rows]
    newRows[index] = { ...newRows[index], [field]: value }
    setRows(newRows)
  }

  return (
    <div style={{ padding: 16, display: 'flex', gap: 16, flexDirection: isMobile ? 'column' : 'row' }}>
      {/* Left: Datacenter List */}
      <Card title="机房列表" extra={<Button icon={<PlusOutlined />} onClick={() => { setEditingDc(null); dcForm.resetFields(); setDcModalOpen(true) }} />} style={{ width: isMobile ? '100%' : 280, flexShrink: 0 }}>
        <List
          dataSource={datacenters}
          size="small"
          renderItem={dc => (
            <List.Item
              style={{ cursor: 'pointer', background: selectedId === dc.id ? '#e6f7ff' : undefined, padding: '8px 12px', borderRadius: 4 }}
              onClick={() => setSelectedId(dc.id)}
              actions={[
                <EditOutlined onClick={(e) => { e.stopPropagation(); setEditingDc(dc); dcForm.setFieldsValue(dc); setDcModalOpen(true) }} />,
                <Popconfirm title="确认删除？" onConfirm={(e) => { e?.stopPropagation(); handleDeleteDc(dc.id) }}>
                  <DeleteOutlined onClick={(e) => e.stopPropagation()} style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta title={<>{dc.name} {dc.current_status && <Tag color={dc.current_status === '运行中' ? 'green' : dc.current_status === '建设中' ? 'blue' : 'default'} style={{ marginLeft: 4, fontSize: 11 }}>{dc.current_status}</Tag>}</>} description={dc.campus || dc.remark || '无备注'} />
            </List.Item>
          )}
        />
      </Card>

      {/* Right: Configuration */}
      <div style={{ flex: 1 }}>
        {!selected ? (
          <Card><div style={{ textAlign: 'center', padding: 40, color: '#999' }}>请选择或创建一个机房</div></Card>
        ) : (
          <>
            {/* Column Configuration */}
            <Card title="列定义" size="small" style={{ marginBottom: 16 }}
              extra={<Space><Button size="small" onClick={addColumn}>添加列</Button><Button size="small" type="primary" onClick={handleSaveColumns}>保存列</Button></Space>}
            >
              {columns.map((col, i) => (
                <Row key={i} gutter={8} style={{ marginBottom: 8 }} align="middle">
                  <Col span={8}><Input value={col.name} placeholder="列名" size="small" onChange={e => updateColumn(i, 'name', e.target.value)} /></Col>
                  <Col span={6}>
                    <Select value={col.column_type || 'cabinet'} size="small" style={{ width: '100%' }} onChange={v => updateColumn(i, 'column_type', v)}>
                      <Option value="cabinet">机柜</Option>
                      <Option value="hda">HDA</Option>
                      <Option value="pdu">PDU</Option>
                      <Option value="aircon">空调</Option>
                      <Option value="other">其他</Option>
                    </Select>
                  </Col>
                  <Col span={6}><InputNumber value={col.sort_order} size="small" style={{ width: '100%' }} onChange={v => updateColumn(i, 'sort_order', v)} /></Col>
                  <Col span={4}><Button size="small" danger icon={<DeleteOutlined />} onClick={() => removeColumn(i)} /></Col>
                </Row>
              ))}
            </Card>

            {/* Row Configuration */}
            <Card title="行定义" size="small" style={{ marginBottom: 16 }}
              extra={<Space><Button size="small" onClick={addRow}>添加行</Button><Button size="small" type="primary" onClick={handleSaveRows}>保存行</Button></Space>}
            >
              {rows.map((row, i) => (
                <Row key={i} gutter={8} style={{ marginBottom: 8 }} align="middle">
                  <Col span={10}><Input value={row.name} placeholder="行名" size="small" onChange={e => updateRow(i, 'name', e.target.value)} /></Col>
                  <Col span={8}><InputNumber value={row.sort_order} size="small" style={{ width: '100%' }} onChange={v => updateRow(i, 'sort_order', v)} /></Col>
                  <Col span={4}><Button size="small" danger icon={<DeleteOutlined />} onClick={() => removeRow(i)} /></Col>
                </Row>
              ))}
            </Card>

            {/* Generate Cabinets */}
            <Card title="机柜管理" size="small" extra={<Button type="primary" icon={<AppstoreOutlined />} onClick={handleGenerate}>生成机柜</Button>}>
              {cabinets.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 20, color: '#999' }}>配置列和行后点击"生成机柜"</div>
              ) : (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                  {cabinets.map(c => (
                    <div key={c.id} style={{ padding: '4px 12px', background: '#f0f0f0', borderRadius: 4, fontSize: 12 }}>
                      {c.name} <span style={{ color: '#999' }}>({c.height}U)</span>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </>
        )}
      </div>

      {/* Datacenter Create/Edit Modal */}
      <Modal title={editingDc ? '编辑机房' : '新建机房'} open={dcModalOpen} onOk={handleSaveDc} onCancel={() => setDcModalOpen(false)} width={600}>
        <Form form={dcForm} layout="vertical">
          <Form.Item label="机房名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="所属数据中心/园区" name="campus">
                <Input placeholder="如：中联重科总部" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="地理位置" name="location">
                <Input placeholder="如：湖南省长沙市岳麓区" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="楼层" name="floor">
                <Input placeholder="如：1F" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="房间号" name="room">
                <Input placeholder="如：101" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="联系人" name="contact">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item label="运营方式" name="operation_mode">
                <Select placeholder="请选择" allowClear>
                  <Option value="自建">自建</Option>
                  <Option value="托管">托管</Option>
                  <Option value="租赁">租赁</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="当前状态" name="current_status">
                <Select placeholder="请选择">
                  <Option value="运行中">运行中</Option>
                  <Option value="建设中">建设中</Option>
                  <Option value="停用">停用</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="机柜最大U数" name="max_u" initialValue={47}>
                <Input type="number" min={1} max={100} />
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
