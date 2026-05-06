import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import './index.css'
import App from './App.tsx'
import { zoomlionTheme } from './theme/zoomlion'
import { registerZoomlionEChartsTheme } from './theme/echarts'

dayjs.locale('zh-cn')
registerZoomlionEChartsTheme()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ConfigProvider locale={zhCN} theme={zoomlionTheme}>
      <App />
    </ConfigProvider>
  </StrictMode>,
)
