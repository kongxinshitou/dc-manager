import { useEffect, useState } from 'react'
import { Card, Row, Col, Tag, Typography, Spin, Badge, Statistic, Space, Alert, Button } from 'antd'
import { AlertOutlined, CheckCircleOutlined, ExclamationCircleOutlined, DatabaseOutlined, ReloadOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
import ResponsiveTable from '../components/ResponsiveTable'
import { getDashboard } from '../api'

const { Title } = Typography

const severityColor: Record<string, string> = {
  严重: 'red',
  一般: 'orange',
  轻微: 'blue',
}

const statusColor: Record<string, string> = {
  待处理: 'red',
  处理中: 'orange',
  已解决: 'green',
}


export default function Dashboard() {
  const [data, setData] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [issuesPage, setIssuesPage] = useState(1)
  const [issuesPageSize, setIssuesPageSize] = useState(20)

  const fetchData = (page: number, pageSize: number) => {
    setLoading(true)
    setError(null)
    getDashboard({ issues_page: page, issues_page_size: pageSize })
      .then(d => { setData(d); setLoading(false) })
      .catch((err: any) => {
        setError(err?.response?.data?.error || err?.message || '加载失败，请检查网络或后端服务')
        setLoading(false)
      })
  }

  useEffect(() => {
    fetchData(issuesPage, issuesPageSize)
  }, [issuesPage, issuesPageSize])

  const safeData = data || {}
  const severeCount = (safeData.severity_stats || []).find((s: any) => s.severity === '严重')?.count ?? 0

  // Room stats bar chart
  const roomChartOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: 20, right: 20, bottom: 20, top: 40, containLabel: true },
    xAxis: {
      type: 'category',
      data: (safeData.room_stats || []).map((s: any) => s.datacenter || '未知'),
      axisLabel: { rotate: 30, fontSize: 11 },
    },
    yAxis: { type: 'value', name: '问题数' },
    series: [{
      name: '未解决问题',
      type: 'bar',
      data: (safeData.room_stats || []).map((s: any) => s.count),
      label: { show: true, position: 'top' },
    }],
  }

  // Trend line chart
  const trendChartOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: 20, right: 20, bottom: 20, top: 40, containLabel: true },
    xAxis: {
      type: 'category',
      data: (safeData.trends || []).map((t: any) => t.date),
      axisLabel: { rotate: 30, fontSize: 10 },
    },
    yAxis: { type: 'value', name: '新增问题数' },
    series: [{
      name: '新增问题',
      type: 'line',
      data: (safeData.trends || []).map((t: any) => t.count),
      smooth: true,
      areaStyle: { opacity: 0.18 },
    }],
  }

  // Severity pie chart
  const severityChartOption = {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [{
      name: '问题等级',
      type: 'pie',
      radius: ['40%', '70%'],
      data: (safeData.severity_stats || []).map((s: any) => ({
        name: s.severity || '未知',
        value: s.count,
        itemStyle: { color: severityColor[s.severity] || '#aaa' },
      })),
      label: { formatter: '{b}: {c}' },
    }],
  }

  // Datacenter device count bar chart
  const datacenterDeviceChartOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: 20, right: 20, bottom: 20, top: 40, containLabel: true },
    xAxis: {
      type: 'category',
      data: (safeData.datacenter_device_stats || []).map((s: any) => s.datacenter || '未知'),
      axisLabel: { rotate: 30, fontSize: 11 },
    },
    yAxis: { type: 'value', name: '设备数' },
    series: [{
      name: '设备数量',
      type: 'bar',
      data: (safeData.datacenter_device_stats || []).map((s: any) => s.count),
      label: { show: true, position: 'top' },
    }],
  }

  // Device type pie chart
  const deviceTypeChartOption = {
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0, type: 'scroll' },
    series: [{
      name: '设备类型',
      type: 'pie',
      radius: ['35%', '65%'],
      data: (safeData.device_type_stats || []).map((s: any) => ({
        name: s.device_type || '未知',
        value: s.count,
      })),
      label: { formatter: '{b}: {c}' },
    }],
  }

  const columns = [
    { title: '发现时间', dataIndex: 'found_at', key: 'found_at', width: 160,
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm') },
    { title: '机房', dataIndex: 'datacenter', key: 'datacenter', width: 120 },
    { title: '机柜', dataIndex: 'cabinet', key: 'cabinet', width: 80 },
    { title: '问题描述', dataIndex: 'issue', key: 'issue', ellipsis: true },
    { title: '等级', dataIndex: 'severity', key: 'severity', width: 70,
      render: (v: string) => <Tag color={severityColor[v]}>{v}</Tag> },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80,
      render: (v: string) => <Tag color={statusColor[v]}>{v}</Tag> },
    { title: '升级', dataIndex: 'escalation_level', key: 'escalation_level', width: 70,
      render: (v: number) => <Tag color={v > 0 ? 'volcano' : 'default'}>{v > 0 ? `L${v}` : '无'}</Tag> },
    { title: '巡检人', dataIndex: 'inspector', key: 'inspector', width: 80 },
    { title: '责任人', dataIndex: 'assignee_name', key: 'assignee_name', width: 90,
      render: (v: string) => v || '-' },
  ]

  return (
    <Spin spinning={loading && !data} size="large">
    <div style={{ padding: '16px', minHeight: 200 }}>
      <Title level={4} style={{ marginBottom: 16 }}>数据中心巡检大屏</Title>

      {error && (
        <Alert
          type="error"
          showIcon
          closable
          style={{ marginBottom: 16 }}
          message={error}
          action={
            <Button size="small" icon={<ReloadOutlined />} onClick={() => fetchData(issuesPage, issuesPageSize)}>
              重试
            </Button>
          }
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} md={5}>
          <Card>
            <Statistic
              title="设备总数"
              value={safeData.total_devices ?? 0}
              valueStyle={{ color: '#1677ff' }}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={5}>
          <Card>
            <Statistic
              title="未解决问题总数"
              value={safeData.status_stats?.reduce((s: number, i: any) => i.status !== '已解决' ? s + i.count : s, 0) ?? 0}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={8} md={4}>
          <Card>
            <Statistic
              title="严重问题"
              value={severeCount}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={5}>
          <Card>
            <Statistic
              title="已解决问题"
              value={safeData.status_stats?.find((s: any) => s.status === '已解决')?.count ?? 0}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={12} sm={12} md={5}>
          <Card>
            <Statistic
              title="涉及机房数"
              value={safeData.room_stats?.length ?? 0}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 巡检问题图表 */}
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={24} md={12}>
          <Card title="各机房未解决问题数量" size="small" className="zl-card-accent">
            <ReactECharts option={roomChartOption} theme="zoomlion" style={{ height: 280, width: '100%' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card title="问题等级分布" size="small" className="zl-card-accent">
            <ReactECharts option={severityChartOption} theme="zoomlion" style={{ height: 280, width: '100%' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card title="状态统计" size="small" className="zl-card-accent" style={{ height: '100%' }}>
            <div style={{ padding: '20px 0' }}>
              {(safeData.status_stats || []).map((s: any) => (
                <div key={s.status} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                  <Tag color={statusColor[s.status] || 'default'}>{s.status || '未知'}</Tag>
                  <Badge count={s.count} showZero color={statusColor[s.status] || '#aaa'} overflowCount={9999} />
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 设备统计图表 */}
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col xs={24} md={14}>
          <Card title="机房设备统计" size="small" className="zl-card-accent">
            <ReactECharts option={datacenterDeviceChartOption} theme="zoomlion" style={{ height: 280, width: '100%' }} />
          </Card>
        </Col>
        <Col xs={24} md={10}>
          <Card title="设备类型统计" size="small" className="zl-card-accent">
            <ReactECharts option={deviceTypeChartOption} theme="zoomlion" style={{ height: 280, width: '100%' }} />
          </Card>
        </Col>
      </Row>

      {/* 趋势图 */}
      <Row gutter={[12, 12]} style={{ marginBottom: 16 }}>
        <Col span={24}>
          <Card title="近30天问题趋势" size="small" className="zl-card-accent">
            <ReactECharts option={trendChartOption} theme="zoomlion" style={{ height: 200, width: '100%' }} />
          </Card>
        </Col>
      </Row>

      {/* 未解决问题列表（支持分页） */}
      <Card title="近期未解决问题（按发现时间排序）" size="small">
        <ResponsiveTable<any>
          dataSource={safeData.recent_issues || []}
          columns={columns}
          rowKey="id"
          size="small"
          loading={loading}
          scroll={{ x: 800 }}
          pagination={{
            total: safeData.recent_issues_total ?? 0,
            current: issuesPage,
            pageSize: issuesPageSize,
            showSizeChanger: true,
            pageSizeOptions: ['20', '50'],
            onChange: (page, pageSize) => {
              setIssuesPage(page)
              setIssuesPageSize(pageSize)
            },
          }}
          mobileCardRender={(record: any) => (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
                <Space size={4} wrap>
                  <Tag color={severityColor[record.severity]}>{record.severity}</Tag>
                  <Tag color={statusColor[record.status]}>{record.status}</Tag>
                  {record.escalation_level > 0 && <Tag color="volcano">L{record.escalation_level}</Tag>}
                </Space>
                <span style={{ fontSize: 12, color: '#999' }}>
                  {dayjs(record.found_at).format('MM-DD HH:mm')}
                </span>
              </div>
              <div style={{ fontSize: 13, marginBottom: 4 }}>
                <span style={{ color: '#999' }}>位置：</span>
                {record.datacenter} / {record.cabinet || '-'}
              </div>
              <div style={{ fontSize: 13, marginBottom: 4, wordBreak: 'break-word' }}>
                <span style={{ color: '#999' }}>问题：</span>{record.issue}
              </div>
              <div style={{ fontSize: 13 }}>
                <span style={{ color: '#999' }}>巡检人：</span>{record.inspector || '-'}
              </div>
              <div style={{ fontSize: 13 }}>
                <span style={{ color: '#999' }}>责任人：</span>{record.assignee_name || '-'}
              </div>
            </div>
          )}
        />
      </Card>
    </div>
    </Spin>
  )
}
