import * as echarts from 'echarts'
import { ZL, ZL_CHART_PALETTE } from './zoomlion'

let registered = false

export function registerZoomlionEChartsTheme() {
  if (registered) return
  echarts.registerTheme('zoomlion', {
    color: [...ZL_CHART_PALETTE],
    backgroundColor: 'transparent',
    textStyle: { color: ZL.text },
    title: {
      textStyle:    { color: ZL.text, fontWeight: 600 },
      subtextStyle: { color: ZL.textSecondary },
    },
    line:    { itemStyle: { borderWidth: 2 }, lineStyle: { width: 2 }, symbol: 'circle', symbolSize: 6, smooth: true },
    bar:     { itemStyle: { borderRadius: [4, 4, 0, 0] } },
    pie:     { itemStyle: { borderColor: '#fff', borderWidth: 1 } },
    categoryAxis: {
      axisLine:  { lineStyle: { color: ZL.border } },
      axisTick:  { lineStyle: { color: ZL.border } },
      axisLabel: { color: ZL.textSecondary },
      splitLine: { show: false, lineStyle: { color: ZL.border } },
    },
    valueAxis: {
      axisLine:  { show: false, lineStyle: { color: ZL.border } },
      axisTick:  { show: false },
      axisLabel: { color: ZL.textSecondary },
      splitLine: { lineStyle: { color: ZL.border, type: 'dashed' } },
    },
    legend: { textStyle: { color: ZL.textSecondary } },
    tooltip: {
      backgroundColor: '#fff',
      borderColor: ZL.border,
      borderWidth: 1,
      textStyle: { color: ZL.text },
    },
  })
  registered = true
}
