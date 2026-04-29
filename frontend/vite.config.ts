import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // 手机端兼容：默认 'baseline-widely-available' 大约 Chrome 107+，会用到 ES2023 语法（toSorted/findLast 等），
  // 国内 X5/UC/QQ/华为浏览器跑的 Chromium 80–100 直接解析失败 → 白屏。
  // 拉低目标到 2020 主流移动端基线，让 esbuild 把新语法降到 ES2017 等价形式。
  build: {
    target: ['chrome87', 'edge88', 'firefox78', 'safari14'],
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        // 拆 vendor：react / antd / echarts 各自独立，主包瘦身，单 chunk 失败不再闷死整个应用。
        // Vite 8 走 Rolldown，manualChunks 必须是函数。
        manualChunks(id: string) {
          if (!id.includes('node_modules')) return
          if (id.includes('/echarts/') || id.includes('/echarts-for-react/') || id.includes('/zrender/')) return 'echarts'
          if (id.includes('/antd/') || id.includes('/@ant-design/') || id.includes('/rc-')) return 'antd'
          if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/scheduler/')) return 'react'
        },
      },
    },
  },
  server: {
    port: 9000,
    host: true,
    allowedHosts: true,
    proxy: {
      '/api': 'http://localhost:8080',
      '/mcp': 'http://localhost:8080',
    },
  },
  // 手机端通过内网穿透访问时，请用 `npm run mobile` 走 preview，
  // 走构建产物（一个 HTML + 一个 bundle），避免 dev server 的 ESM 拆包 + HMR ws 在公网隧道下首屏加载几百次小请求导致白屏。
  preview: {
    port: 9000,
    host: true,
    allowedHosts: true,
    proxy: {
      '/api': 'http://localhost:8080',
      '/mcp': 'http://localhost:8080',
    },
  },
})
