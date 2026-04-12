import { Grid, Table, Card, Empty } from 'antd'
import type { TableProps } from 'antd'

const { useBreakpoint } = Grid

interface ResponsiveTableProps<T> extends TableProps<T> {
  /** 在移动端展示的卡片内容渲染函数 */
  mobileCardRender?: (item: T) => React.ReactNode
}

/**
 * 响应式表格组件：桌面端显示 Table，移动端显示 Card 列表
 */
export default function ResponsiveTable<T extends object>({
  mobileCardRender,
  ...tableProps
}: ResponsiveTableProps<T>) {
  const screens = useBreakpoint()
  const isMobile = !screens.md

  if (!isMobile || !mobileCardRender) {
    return <Table<T> {...tableProps} />
  }

  const dataSource = tableProps.dataSource || []

  if (dataSource.length === 0) {
    return <Empty description="暂无数据" style={{ padding: 32 }} />
  }

  return (
    <div>
      {dataSource.map((item, index) => {
        const key = tableProps.rowKey
          ? typeof tableProps.rowKey === 'function'
            ? tableProps.rowKey(item, index)
            : (item as any)[tableProps.rowKey as string]
          : index
        return (
          <Card
            key={key}
            size="small"
            style={{ marginBottom: 8 }}
          >
            {mobileCardRender(item)}
          </Card>
        )
      })}
    </div>
  )
}
