import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AppSidebar from '@/components/layout/AppSidebar.vue'

const pushMock = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/usage' }),
  useRouter: () => ({ push: pushMock }),
}))

vi.mock('@/composables/useBatchImageAccess', () => ({
  useBatchImageAccess: () => ({
    canUseBatchImage: { value: false },
    refreshBatchImageAccess: vi.fn().mockResolvedValue(false),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'nav.dashboard': '仪表盘',
    'nav.apiKeys': 'API 密钥',
    'nav.usage': '使用记录',
    'nav.dailyLeaderboard': '每日榜单',
    'nav.availableChannels': '可用渠道',
    'nav.channelStatus': '渠道状态',
    'nav.mySubscriptions': '我的订阅',
    'nav.buySubscription': '充值/订阅',
    'nav.myOrders': '我的订单',
    'nav.redeem': '兑换',
    'nav.affiliate': '邀请返利',
    'nav.community': '交流群',
    'nav.profile': '个人资料',
    'nav.lightMode': '浅色模式',
    'nav.darkMode': '深色模式',
    'nav.expand': '展开',
    'nav.collapse': '收起',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    sidebarCollapsed: false,
    mobileOpen: false,
    backendModeEnabled: false,
    siteName: 'Sub2API',
    siteLogo: '',
    siteVersion: '2026',
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      custom_menu_items: [
        { id: 'chat-test', label: '对话测试', icon_svg: '<svg></svg>', url: '', visibility: 'user', sort_order: 10 },
        { id: 'recharge', label: '充值', icon_svg: '<svg></svg>', url: '', visibility: 'user', sort_order: 20 },
        { id: 'ai-image', label: 'AI生图', icon_svg: '<svg></svg>', url: '', visibility: 'user', sort_order: 30 },
        { id: 'model-pricing', label: '模型价格', icon_svg: '<svg></svg>', url: '', visibility: 'user', sort_order: 40 },
      ],
    },
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn(),
  }),
  useAuthStore: () => ({
    isAdmin: false,
    isSimpleMode: false,
  }),
  useOnboardingStore: () => ({
    isCurrentStep: () => false,
    nextStep: vi.fn(),
  }),
  useAdminSettingsStore: () => ({
    customMenuItems: [],
    opsMonitoringEnabled: true,
    paymentEnabled: true,
    fetch: vi.fn(),
  }),
}))

vi.mock('@/utils/featureFlags', () => ({
  FeatureFlags: {
    channelMonitor: 'channelMonitor',
    payment: 'payment',
    availableChannels: 'availableChannels',
    affiliate: 'affiliate',
    riskControl: 'riskControl',
  },
  makeSidebarFlag: (flag: string) => () => flag !== 'payment',
}))

vi.mock('@/utils/sanitize', () => ({
  sanitizeSvg: (svg: string) => svg,
}))

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

beforeEach(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('lets the user collapse a group even while a child route is active', () => {
    // The expand state must come from the user's override first, falling back
    // to the active-route heuristic only when the user has not clicked yet.
    expect(componentSource).toContain('const groupExpandOverrides = ref<Map<string, boolean>>(new Map())')
    expect(componentSource).not.toContain('expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar community navigation', () => {
  it('links users to the community page', () => {
    expect(componentSource).toContain("path: '/community'")
    expect(componentSource).toContain("label: t('nav.community')")
  })
})

describe('AppSidebar daily leaderboard navigation', () => {
  it('links users to the daily leaderboard page', () => {
    expect(componentSource).toContain("path: '/leaderboard'")
    expect(componentSource).toContain("label: t('nav.dailyLeaderboard')")
    expect(componentSource).toContain('RacingLeaderboardIcon')
  })
})

describe('AppSidebar admin balance overview navigation', () => {
  it('links admins to the dedicated balance overview page', () => {
    expect(componentSource).toContain("path: '/admin/balances'")
    expect(componentSource).toContain("label: t('nav.balanceOverview')")
    expect(componentSource).toContain('CreditCardIcon')
  })
})

describe('AppSidebar user navigation order', () => {
  it('places recharge with redeem and keeps pricing/status/profile items at the end', () => {
    const wrapper = mount(AppSidebar, {
      global: {
        stubs: {
          RouterLink: RouterLinkStub,
          VersionBadge: true,
        },
      },
    })

    const labels = wrapper
      .findAll('a.sidebar-link')
      .map((link) => link.text().trim())
      .filter(Boolean)

    expect(labels).toEqual([
      '仪表盘',
      'API 密钥',
      '使用记录',
      '每日榜单',
      '交流群',
      '充值',
      '兑换',
      '我的订阅',
      '对话测试',
      'AI生图',
      '模型价格',
      '可用渠道',
      '渠道状态',
      '邀请返利',
      '个人资料',
    ])
  })
})
