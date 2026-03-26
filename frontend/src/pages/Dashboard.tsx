import { useEffect, useState } from 'react'
import { Card, Row, Col, Table, Tag, Typography, Spin, Badge, Statistic } from 'antd'
import { AlertOutlined, CheckCircleOutlined, ExclamationCircleOutlined, DatabaseOutlined } from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import dayjs from 'dayjs'
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
  const [issuesPage, setIssuesPage] = useState(1)
  const [issuesPageSize, setIssuesPageSize] = useState(20)

  const fetchData = (page: number, pageSize: number) => {
    setLoading(true)
    getDashboard({ issues_page: page, issues_page_size: pageSize })
      .then(d => { setData(d); setLoading(false) })
      .catch(() => setLoading(false))
  }

  useEffect(() => {
    fetchData(issuesPage, issuesPageSize)
  }, [issuesPage, issuesPageSize])

  if (loading && !data) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
  if (!data) return null

  const severeCount = (data.severity_stats || []).find((s: any) => s.severity === '严重')?.count ?? 0

  // Room stats bar chart
  const roomChartOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: 20, right: 20, bottom: 20, top: 40, containLabel: true },
    xAxis: {
      type: 'category',
      data: (data.room_stats || []).map((s: any) => s.datacenter || '未知'),
      axisLabel: { rotate: 30, fontSize: 11 },
    },
    yAxis: { type: 'value', name: '问题数' },
    series: [{
      name: '未解决问题',
      type: 'bar',
      data: (data.room_stats || []).map((s: any) => s.count),
      itemStyle: { color: '#ff4d4f' },
      label: { show: true, position: 'top' },
    }],
  }

  // Trend line chart
  const trendChartOption = {
    tooltip: { trigger: 'axis' },
    grid: { left: 20, right: 20, bottom: 20, top: 40, containLabel: true },
    xAxis: {
      type: 'category',
      data: (data.trends || []).map((t: any) => t.date),
      axisLabel: { rotate: 30, fontSize: 10 },
    },
    yAxis: { type: 'value', name: '新增问题数' },
    series: [{
      name: '新增问题',
      type: 'line',
      data: (data.trends || []).map((t: any) => t.count),
      smooth: true,
      areaStyle: { opacity: 0.3 },
      itemStyle: { color: '#1677ff' },
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
      data: (data.severity_stats || []).map((s: any) => ({
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
      data: (data.datacenter_device_stats || []).map((s: any) => s.datacenter || '未知'),
      axisLabel: { rotate: 30, fontSize: 11 },
    },
    yAxis: { type: 'value', name: '设备数' },
    series: [{
      name: '设备数量',
      type: 'bar',
      data: (data.datacenter_device_stats || []).map((s: any) => s.count),
      itemStyle: { color: '#1677ff' },
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
      data: (data.device_type_stats || []).map((s: any) => ({
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
    { title: '巡检人', dataIndex: 'inspector', key: 'inspector', width: 80 },
  ]

  return (
    <div style={{ padding: '16px' }}>
      <Title level={4} style={{ marginBottom: 16 }}>数据中心巡检大屏</Title>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={5}>
          <Card>
            <Statistic
              title="设备总数"
              value={data.total_devices ?? 0}
              valueStyle={{ color: '#1677ff' }}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="未解决问题总数"
              value={data.status_stats?.reduce((s: number, i: any) => i.status !== '已解决' ? s + i.count : s, 0) ?? 0}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<AlertOutlined />}
            />
          </Card>
        </Col>
        <Col span={4}>
          <Card>
            <Statistic
              title="严重问题"
              value={severeCount}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="已解决问题"
              value={data.status_stats?.find((s: any) => s.status === '已解决')?.count ?? 0}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title="涉及机房数"
              value={data.room_stats?.length ?? 0}
              valueStyle={{ color: '#722ed1' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 巡检问题图表 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card title="各机房未解决问题数量" size="small">
            <ReactECharts option={roomChartOption} style={{ height: 280 }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card title="问题等级分布" size="small">
            <ReactECharts option={severityChartOption} style={{ height: 280 }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card title="状态统计" size="small" style={{ height: '100%' }}>
            <div style={{ padding: '20px 0' }}>
              {(data.status_stats || []).map((s: any) => (
                <div key={s.status} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 12 }}>
                  <Tag color={statusColor[s.status] || 'default'}>{s.status || '未知'}</Tag>
                  <Badge count={s.count} showZero color={statusColor[s.status] || '#aaa'} overflowCount={9999} />
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 设备统计图表（新增） */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={14}>
          <Card title="机房设备统计（各机房设备数量）" size="small">
            <ReactECharts option={datacenterDeviceChartOption} style={{ height: 280 }} />
          </Card>
        </Col>
        <Col span={10}>
          <Card title="设备类型统计" size="small">
            <ReactECharts option={deviceTypeChartOption} style={{ height: 280 }} />
          </Card>
        </Col>
      </Row>

      {/* 趋势图 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={24}>
          <Card title="近30天问题趋势" size="small">
            <ReactECharts option={trendChartOption} style={{ height: 200 }} />
          </Card>
        </Col>
      </Row>

      {/* 未解决问题列表（支持分页） */}
      <Card title="近期未解决问题（按发现时间排序）" size="small">
        <Table
          dataSource={data.recent_issues || []}
          columns={columns}
          rowKey="id"
          size="small"
          loading={loading}
          scroll={{ x: 800 }}
          pagination={{
            total: data.recent_issues_total ?? 0,
            current: issuesPage,
            pageSize: issuesPageSize,
            showSizeChanger: true,
            pageSizeOptions: ['20', '50'],
            onChange: (page, pageSize) => {
              setIssuesPage(page)
              setIssuesPageSize(pageSize)
            },
          }}
        />
      </Card>
    </div>
  )
}
