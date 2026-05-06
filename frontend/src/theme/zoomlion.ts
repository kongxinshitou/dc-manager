import type { ThemeConfig } from 'antd'

export const ZL = {
  green:         '#AACE39',
  greenHover:    '#99BA2F',
  greenActive:   '#86A526',
  greenSoft:     'rgba(170, 206, 57, 0.12)',
  greenSofter:   'rgba(170, 206, 57, 0.06)',
  starGray:      '#383842',
  gravelGray:    '#6B6F76',
  text:          '#1F2329',
  textSecondary: '#5F6670',
  muted:         '#8A9199',
  bg:            '#F5F7F4',
  section:       '#F7F8FA',
  card:          '#FFFFFF',
  border:        '#E5E8EB',
  footer:        '#262A31',
  radiusCard:    8,
  radiusButton:  6,
} as const

export const zoomlionTheme: ThemeConfig = {
  token: {
    colorPrimary:         ZL.green,
    colorInfo:            '#1677ff',
    colorSuccess:         '#52c41a',
    colorWarning:         '#fa8c16',
    colorError:           '#ff4d4f',
    colorText:            ZL.text,
    colorTextSecondary:   ZL.textSecondary,
    colorBorderSecondary: ZL.border,
    colorBgLayout:        ZL.bg,
    colorBgContainer:     ZL.card,
    borderRadius:         ZL.radiusButton,
    fontFamily:
      '"PingFang SC","Microsoft YaHei",-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif',
  },
  components: {
    Layout: {
      headerBg: ZL.card,
      siderBg:  ZL.card,
      bodyBg:   ZL.bg,
    },
    Menu: {
      itemSelectedBg:    ZL.greenSoft,
      itemSelectedColor: ZL.green,
      itemHoverColor:    ZL.green,
    },
    Button: {
      primaryShadow:      'none',
      defaultBorderColor: ZL.border,
    },
    Card: {
      borderRadiusLG: ZL.radiusCard,
    },
    Table: {
      headerBg:    ZL.section,
      headerColor: ZL.text,
      rowHoverBg:  ZL.greenSofter,
    },
    Tabs: {
      inkBarColor:       ZL.green,
      itemSelectedColor: ZL.green,
      itemHoverColor:    ZL.greenHover,
    },
  },
}

export const ZL_CHART_PALETTE = [
  ZL.green,
  ZL.starGray,
  '#1677ff',
  '#fa8c16',
  '#722ed1',
  '#13c2c2',
  '#eb2f96',
  '#52c41a',
] as const
