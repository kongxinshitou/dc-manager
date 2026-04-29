import { useEffect, useState, useCallback } from 'react'
import { Select, Card, Row, Col, Drawer, Grid, Spin, Modal, Form, InputNumber, Button, message, Tag } from 'antd'
import {
  getDatacenters, getDatacenterLayout, getCabinetDevices,
  getDevices, submitApproval,
} from '../api'
import type { Datacenter, DatacenterLayout, Cabinet, CabinetColumn, DeviceWithWarranty } from '../api'

const { Option } = Select

const CABINET_COLORS = {
  low: '#52c41a',    // < 40% usage
  medium: '#1890ff', // 40-70% usage
  high: '#fa8c16',   // 70-90% usage
  full: '#f5222d',   // > 90% usage
  non_cabinet: '#d9d9d9',
}

function getCabinetColor(cabinet: Cabinet): string {
  if (cabinet.cabinet_type && cabinet.cabinet_type !== 'standard' && cabinet.cabinet_type !== 'cabinet') {
    return CABINET_COLORS.non_cabinet
  }
  const usage = cabinet.height > 0 ? (cabinet.used_u || 0) / cabinet.height : 0
  if (usage > 0.9) return CABINET_COLORS.full
  if (usage > 0.7) return CABINET_COLORS.high
  if (usage > 0.4) return CABINET_COLORS.medium
  return CABINET_COLORS.low
}

export default function DatacenterLayout() {
  const [datacenters, setDatacenters] = useState<Datacenter[]>([])
  const [selectedDcId, setSelectedDcId] = useState<number | null>(null)
  const [layout, setLayout] = useState<DatacenterLayout | null>(null)
  const [loading, setLoading] = useState(false)
  const [selectedCabinet, setSelectedCabinet] = useState<Cabinet | null>(null)
  const [hoveredCabinet, setHoveredCabinet] = useState<number | null>(null)
  const [cabinetDevices, setCabinetDevices] = useState<any[]>([])
  const [rackModalOpen, setRackModalOpen] = useState(false)
  const [unrackDevice, setUnrackDevice] = useState<any>(null)
  const [inStockDevices, setInStockDevices] = useState<DeviceWithWarranty[]>([])
  const [rackForm] = Form.useForm()
  const [opLoading, setOpLoading] = useState(false)
  const screens = Grid.useBreakpoint()
  const isMobile = !screens.md

  useEffect(() => { getDatacenters().then(setDatacenters) }, [])

  const fetchLayout = useCallback((dcId: number) => {
    setLoading(true)
    getDatacenterLayout(dcId).then(data => {
      setLayout(data)
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (selectedDcId) fetchLayout(selectedDcId)
    else setLayout(null)
  }, [selectedDcId, fetchLayout])

  // Build grid from columns x rows
  const columnMap = new Map<number, CabinetColumn>()
  const columnOrder: number[] = []
  layout?.columns.forEach(col => { columnMap.set(col.id!, col); columnOrder.push(col.id!) })

  const rowOrder: number[] = []
  layout?.rows.forEach(row => rowOrder.push(row.id!))

  // 1D degradation: use virtual row/column when one is missing
  const hasColumns = columnOrder.length > 0
  const hasRows = rowOrder.length > 0
  const noLayout = layout && !hasColumns && !hasRows

  // For 1D: place all cabinets in a single row or column
  const effectiveColumns = hasColumns ? columnOrder : layout?.cabinets.map((_, i) => i) || []
  const effectiveRows = hasRows ? rowOrder : [0]

  // Index cabinets by (column_id, row_id) or by sequential index for 1D
  const cabinetGrid = new Map<string, Cabinet>()
  if (hasColumns && hasRows) {
    layout?.cabinets.forEach(cab => {
      const key = `${cab.column_id || 0}-${cab.row_id || 0}`
      cabinetGrid.set(key, cab)
    })
  } else if (hasColumns && !hasRows) {
    // Only columns: group cabinets by column_id
    const byCol = new Map<number, Cabinet[]>()
    layout?.cabinets.forEach(cab => {
      const list = byCol.get(cab.column_id || 0) || []
      list.push(cab)
      byCol.set(cab.column_id || 0, list)
    })
    columnOrder.forEach(colId => {
      const list = byCol.get(colId) || []
      list.forEach((cab, i) => {
        cabinetGrid.set(`${colId}-${i}`, cab)
      })
    })
    // Adjust effectiveRows to match max cabinets in any column
    const maxPerCol = Math.max(...Array.from(byCol.values()).map(l => l.length), 1)
    effectiveRows.length = 0
    for (let i = 0; i < maxPerCol; i++) effectiveRows.push(i)
  } else if (!hasColumns && hasRows) {
    // Only rows: all cabinets in sequential order per row
    const byRow = new Map<number, Cabinet[]>()
    layout?.cabinets.forEach(cab => {
      const list = byRow.get(cab.row_id || 0) || []
      list.push(cab)
      byRow.set(cab.row_id || 0, list)
    })
    rowOrder.forEach(rowId => {
      const list = byRow.get(rowId) || []
      list.forEach((cab, i) => {
        cabinetGrid.set(`${i}-${rowId}`, cab)
      })
    })
  } else {
    // Neither: place cabinets sequentially in one row
    layout?.cabinets.forEach((cab, i) => {
      cabinetGrid.set(`${i}-0`, cab)
    })
  }

  // Stats
  const totalCabinets = layout?.cabinets.length || 0
  const usedCabinets = layout?.cabinets.filter(c => (c.used_u || 0) > 0).length || 0
  const totalU = layout?.cabinets.reduce((sum, c) => sum + c.height, 0) || 0
  const usedU = layout?.cabinets.reduce((sum, c) => sum + (c.used_u || 0), 0) || 0
  const usagePercent = totalU > 0 ? ((usedU / totalU) * 100).toFixed(1) : '0'

  const cellWidth = isMobile ? 50 : 117

  const handleSelectCabinet = (cab: Cabinet) => {
    setSelectedCabinet(cab)
    if (cab.id) {
      getCabinetDevices(cab.id).then(res => setCabinetDevices(res.devices || [])).catch(() => setCabinetDevices([]))
    }
  }

  const handleOpenRackModal = () => {
    rackForm.resetFields()
    // Fetch in_stock devices
    getDevices({ device_status: 'in_stock', page: 1, page_size: 200 }).then(res => {
      setInStockDevices(res.data || [])
    })
    setRackModalOpen(true)
  }

  const handleRackSubmit = async () => {
    if (!selectedCabinet || !selectedDcId) return
    const values = await rackForm.validateFields()
    setOpLoading(true)
    try {
      await submitApproval({
        device_id: values.device_id,
        operation_type: 'rack',
        request_data: {
          datacenter: layout?.datacenter.name,
          cabinet: selectedCabinet.name,
          datacenter_id: selectedDcId,
          cabinet_id: selectedCabinet.id,
          start_u: values.start_u,
          u_count: values.u_count,
        },
      })
      message.success('上架申请已提交审批')
      setRackModalOpen(false)
      fetchLayout(selectedDcId)
      // Refresh cabinet devices
      if (selectedCabinet.id) {
        getCabinetDevices(selectedCabinet.id).then(res => setCabinetDevices(res.devices || []))
      }
    } catch (err: any) {
      message.error(err.response?.data?.error || '提交失败')
    } finally {
      setOpLoading(false)
    }
  }

  const handleUnrack = async (device: any) => {
    setOpLoading(true)
    try {
      await submitApproval({
        device_id: device.id,
        operation_type: 'unrack',
        request_data: { storage_location: '仓库待定', custodian: device.custodian || '' },
      })
      message.success('下架申请已提交审批')
      setUnrackDevice(null)
      if (selectedDcId) fetchLayout(selectedDcId)
      if (selectedCabinet?.id) {
        getCabinetDevices(selectedCabinet.id).then(res => setCabinetDevices(res.devices || []))
      }
    } catch (err: any) {
      message.error(err.response?.data?.error || '提交失败')
    } finally {
      setOpLoading(false)
    }
  }
  const cellHeight = isMobile ? 60 : 134
  const gap = 4

  return (
    <div style={{ padding: 16 }}>
      <Card size="small" style={{ marginBottom: 16 }}>
        <Row gutter={[12, 12]} align="middle">
          <Col xs={24} sm={24} md={8}>
            <span style={{ marginRight: 8 }}>选择机房:</span>
            <Select
              value={selectedDcId || undefined}
              onChange={setSelectedDcId}
              placeholder="请选择机房"
              style={{ width: isMobile ? 'calc(100% - 80px)' : 200 }}
              allowClear
            >
              {datacenters.map(dc => <Option key={dc.id} value={dc.id}>{dc.name}</Option>)}
            </Select>
          </Col>
          {layout && (
            <>
              <Col xs={8} sm={8} md={5}><div style={{ padding: isMobile ? '4px 8px' : '8px 16px' }}><div style={{ fontSize: 12, color: '#999' }}>机柜总数</div><div style={{ fontSize: isMobile ? 16 : 20, fontWeight: 600 }}>{totalCabinets}</div></div></Col>
              <Col xs={8} sm={8} md={5}><div style={{ padding: isMobile ? '4px 8px' : '8px 16px' }}><div style={{ fontSize: 12, color: '#999' }}>已用机柜</div><div style={{ fontSize: isMobile ? 16 : 20, fontWeight: 600 }}>{usedCabinets}</div></div></Col>
              <Col xs={8} sm={8} md={6}><div style={{ padding: isMobile ? '4px 8px' : '8px 16px' }}><div style={{ fontSize: 12, color: '#999' }}>U位使用率</div><div style={{ fontSize: isMobile ? 16 : 20, fontWeight: 600 }}>{usagePercent}%</div></div></Col>
            </>
          )}
        </Row>
      </Card>

      {loading && <div style={{ textAlign: 'center', padding: 40 }}><Spin size="large" /></div>}

      {layout && !loading && (
        <>
          {noLayout ? (
            <Card><div style={{ textAlign: 'center', padding: 40, color: '#999' }}>请先配置列和行</div></Card>
          ) : (
          <>
          {/* Legend */}
          <div style={{ marginBottom: 12, display: 'flex', flexWrap: 'wrap', gap: isMobile ? 8 : 16, fontSize: 12 }}>
            <span><span style={{ display: 'inline-block', width: 12, height: 12, background: CABINET_COLORS.low, borderRadius: 2, marginRight: 4 }} />空闲(&lt;40%)</span>
            <span><span style={{ display: 'inline-block', width: 12, height: 12, background: CABINET_COLORS.medium, borderRadius: 2, marginRight: 4 }} />中等(40-70%)</span>
            <span><span style={{ display: 'inline-block', width: 12, height: 12, background: CABINET_COLORS.high, borderRadius: 2, marginRight: 4 }} />较高(70-90%)</span>
            <span><span style={{ display: 'inline-block', width: 12, height: 12, background: CABINET_COLORS.full, borderRadius: 2, marginRight: 4 }} />近满(&gt;90%)</span>
            <span><span style={{ display: 'inline-block', width: 12, height: 12, background: CABINET_COLORS.non_cabinet, borderRadius: 2, marginRight: 4 }} />非机柜</span>
          </div>

          {/* SVG Grid */}
          <div style={{ overflow: 'auto', border: '1px solid #f0f0f0', borderRadius: 4, padding: 8 }}>
            <svg
              width={effectiveColumns.length * (cellWidth + gap) + gap}
              height={effectiveRows.length * (cellHeight + gap) + gap + (hasColumns ? 30 : 0)}
            >
              {/* Column headers */}
              {hasColumns && columnOrder.map((colId, ci) => {
                const col = columnMap.get(colId)
                return col ? (
                  <text key={`col-${colId}`} x={ci * (cellWidth + gap) + gap + cellWidth / 2} y={16} textAnchor="middle" fontSize={11} fontWeight={600}>{col.name}</text>
                ) : null
              })}

              {/* Cabinet cells */}
              {effectiveRows.map((rowId, ri) =>
                effectiveColumns.map((colId, ci) => {
                  const key = `${colId}-${rowId}`
                  const cab = cabinetGrid.get(key)
                  if (!cab) return null
                  const x = ci * (cellWidth + gap) + gap
                  const y = ri * (cellHeight + gap) + gap + (hasColumns ? 24 : 0)
                  const color = getCabinetColor(cab)
                  const usage = cab.height > 0 ? ((cab.used_u || 0) / cab.height * 100).toFixed(0) : '0'
                  const isHovered = hoveredCabinet === cab.id

                  return (
                    <g key={key}
                      onMouseEnter={() => setHoveredCabinet(cab.id)}
                      onMouseLeave={() => setHoveredCabinet(null)}
                      onClick={() => handleSelectCabinet(cab)}
                      style={{ cursor: 'pointer' }}
                    >
                      <rect
                        x={x} y={y} width={cellWidth} height={cellHeight}
                        rx={4} fill={color} fillOpacity={isHovered ? 0.9 : 0.7}
                        stroke={isHovered ? '#333' : '#fff'} strokeWidth={isHovered ? 2 : 1}
                      />
                      <text x={x + cellWidth / 2} y={y + 20} textAnchor="middle" fontSize={10} fontWeight={600} fill="#fff">{cab.name}</text>
                      <text x={x + cellWidth / 2} y={y + 36} textAnchor="middle" fontSize={9} fill="#fff">{usage}%</text>
                      <text x={x + cellWidth / 2} y={y + 50} textAnchor="middle" fontSize={8} fill="#fff">{cab.used_u || 0}/{cab.height}U</text>
                    </g>
                  )
                })
              )}
            </svg>
          </div>

          {/* Cabinet Detail Drawer */}
          <Drawer
            title={selectedCabinet ? `${selectedCabinet.name} (${selectedCabinet.height}U)` : ''}
            open={!!selectedCabinet}
            onClose={() => { setSelectedCabinet(null); setCabinetDevices([]) }}
            width={isMobile ? '100%' : 400}
          >
            {selectedCabinet && (
              <div>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Tag>{selectedCabinet.cabinet_type || 'standard'}</Tag>
                    已用: {selectedCabinet.used_u || 0}U / {selectedCabinet.height}U
                  </div>
                  <Button type="primary" size="small" onClick={handleOpenRackModal}>上架设备</Button>
                </div>

                {/* U-position distribution */}
                <div style={{ border: '1px solid #f0f0f0', borderRadius: 4, padding: 8 }}>
                  <div style={{ fontWeight: 600, marginBottom: 8 }}>U位分布</div>
                  {Array.from({ length: selectedCabinet.height }, (_, i) => {
                    const u = selectedCabinet.height - i
                    const device = cabinetDevices.find((d: any) => {
                      const su = d.start_u; const eu = d.end_u
                      return su != null && eu != null && u >= su && u <= eu
                    })
                    const isOccupied = !!device
                    const isFirstU = device && device.start_u === u
                    return (
                      <div key={u} style={{
                        display: 'flex', alignItems: 'center', height: 20, borderBottom: '1px solid #f5f5f5',
                        background: isOccupied ? '#e6f7ff' : '#fafafa',
                        cursor: isOccupied ? 'pointer' : 'default',
                      }}
                        onClick={() => isOccupied && isFirstU && setUnrackDevice(device)}
                      >
                        <span style={{ width: 30, fontSize: 10, color: '#999', textAlign: 'right', paddingRight: 8 }}>{u}</span>
                        {isOccupied ? (
                          <span style={{ fontSize: 11, color: '#1890ff', flex: 1 }}>
                            {device.brand} {device.model}
                          </span>
                        ) : (
                          <span style={{ fontSize: 10, color: '#d9d9d9' }}>空闲</span>
                        )}
                        {isOccupied && isFirstU && (
                          <Tag color="volcano" style={{ fontSize: 10, cursor: 'pointer', marginLeft: 4 }}>下架</Tag>
                        )}
                      </div>
                    )
                  })}
                </div>

                {/* Device list */}
                {cabinetDevices.length > 0 && (
                  <div style={{ marginTop: 16 }}>
                    <div style={{ fontWeight: 600, marginBottom: 8 }}>设备列表</div>
                    {cabinetDevices.map((d: any, i: number) => (
                      <div key={i} style={{ padding: '8px', borderBottom: '1px solid #f0f0f0', fontSize: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <div><strong>{d.brand} {d.model}</strong></div>
                          <div style={{ color: '#666' }}>U位: {d.start_u}-{d.end_u} | {d.device_type}</div>
                        </div>
                        <Button size="small" danger onClick={() => setUnrackDevice(d)}>下架</Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </Drawer>

          {/* Rack Modal */}
          <Modal
            title={`上架设备到 ${selectedCabinet?.name || ''}`}
            open={rackModalOpen}
            onOk={handleRackSubmit}
            onCancel={() => setRackModalOpen(false)}
            okText="提交审批"
            cancelText="取消"
            confirmLoading={opLoading}
            destroyOnClose
          >
            <Form form={rackForm} layout="vertical">
              <Form.Item label="选择设备" name="device_id" rules={[{ required: true, message: '请选择设备' }]}>
                <Select placeholder="选择入库中的设备" showSearch optionFilterProp="children">
                  {inStockDevices.map(d => (
                    <Select.Option key={d.id} value={d.id}>
                      {d.brand} {d.model} (SN: {d.serial_number || '-'})
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item label="起始U位" name="start_u" rules={[{ required: true, message: '请输入U位' }]}>
                    <InputNumber min={1} max={selectedCabinet?.height || 47} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label="占用U数" name="u_count" rules={[{ required: true, message: '请输入占用U数' }]} initialValue={1}>
                    <InputNumber min={1} max={10} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
            </Form>
          </Modal>

          {/* Unrack Confirm Modal */}
          <Modal
            title="确认下架"
            open={!!unrackDevice}
            onOk={() => unrackDevice && handleUnrack(unrackDevice)}
            onCancel={() => setUnrackDevice(null)}
            okText="提交下架审批"
            cancelText="取消"
            confirmLoading={opLoading}
          >
            {unrackDevice && (
              <div>
                <p>确认下架以下设备？</p>
                <div style={{ padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
                  <strong>{unrackDevice.brand} {unrackDevice.model}</strong>
                  <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>
                    SN: {unrackDevice.serial_number} | U位: {unrackDevice.start_u}-{unrackDevice.end_u}
                  </div>
                </div>
                <div style={{ color: '#faad14', marginTop: 12, fontSize: 12 }}>
                  下架后设备将变为「出库-下架」状态，确认回收后变为「入库-回收」
                </div>
              </div>
            )}
          </Modal>
        </>
          )}
        </>
      )}
    </div>
  )
}
