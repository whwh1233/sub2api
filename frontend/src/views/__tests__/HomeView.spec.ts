import { describe, expect, it, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { checkAuth, fetchPublicSettings, toggleTheme } = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  fetchPublicSettings: vi.fn(),
  toggleTheme: vi.fn(),
}))

const messages: Record<string, string> = {
  'home.viewDocs': '查看文档',
  'home.docs': '文档',
  'home.switchToLight': '切换到浅色模式',
  'home.switchToDark': '切换到深色模式',
  'home.dashboard': '控制台',
  'home.login': '登录',
  'home.getStarted': '立即开始',
  'home.goToDashboard': '进入控制台',
  'home.heroSubtitle': 'ChatGPT 与 Claude 高阶模型池',
  'home.heroDescription': '一个 API 密钥，按需调用 ChatGPT GPT 5.5 / 5.4 与 Claude Opus 4.8 / 4.7，适合代码、写作、推理和团队自动化场景。',
  'home.modelLineup.title': '模型池',
  'home.modelLineup.description': 'ChatGPT · Claude · GPT 5.5 · Opus 4.8',
  'home.modelLineup.soon': '更多高阶模型持续接入',
  'home.features.unifiedGateway': '一键接入',
  'home.features.unifiedGatewayDesc': '获取一个 API 密钥，即可调用已接入的 ChatGPT 与 Claude 模型池，无需分别申请。',
  'home.features.multiAccount': '稳定可靠',
  'home.features.multiAccountDesc': '智能调度多个上游账号，自动切换和负载均衡，减少请求失败。',
  'home.features.balanceQuota': '用多少付多少',
  'home.features.balanceQuotaDesc': '按实际使用量计费，支持设置配额上限，团队用量一目了然。',
  'home.providers.title': 'MODEL LINEUP',
  'home.providers.chatgpt': 'ChatGPT',
  'home.providers.claude': 'Claude',
  'home.providers.gpt55': 'GPT 5.5',
  'home.providers.gpt54': 'GPT 5.4',
  'home.providers.opus48': 'Opus 4.8',
  'home.providers.opus47': 'Opus 4.7',
  'home.providers.soon': '更多模型持续接入',
  'home.footer.allRightsReserved': '保留所有权利。',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
      locale: { value: 'zh' },
    }),
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false,
    user: null,
    checkAuth,
  }),
  useAppStore: () => ({
    cachedPublicSettings: {
      site_name: 'CloseAI',
      site_logo: '',
      site_subtitle: 'ClaudeCode Max 20x、ChatGPT Pro 20x、Gemini、Antigravity',
      doc_url: '',
      home_content: '',
    },
    siteName: 'CloseAI',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings,
    toggleTheme,
  }),
}))

describe('HomeView default marketing copy', () => {
  beforeEach(() => {
    checkAuth.mockReset()
    fetchPublicSettings.mockReset()
    toggleTheme.mockReset()
    localStorage.clear()

    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockReturnValue({ matches: false }),
    })
  })

  it('promotes ChatGPT and Claude model pools without unrelated platform names', () => {
    const wrapper = mount(HomeView, {
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          LocaleSwitcher: true,
          Icon: true,
        },
      },
    })

    const text = wrapper.text()

    expect(text).toContain('CHATGPT + CLAUDE MODEL POOL')
    expect(text).not.toContain('GPT + OPUS MODEL POOL')
    expect(text).toContain('ChatGPT')
    expect(text).toContain('Claude')
    expect(text).toContain('GPT 5.5')
    expect(text).toContain('GPT 5.4')
    expect(text).toContain('Opus 4.8')
    expect(text).toContain('Opus 4.7')
    expect(text).toContain('closeai.space')
    expect(text).toContain('/chat/completions')
    expect(text).toContain('→ routed')
    expect(text).not.toContain('closeai.space/v1')
    expect(text).not.toContain('/v1/chat/completions')
    expect(text).not.toContain('Gemini')
    expect(text).not.toContain('Antigravity')

    wrapper.unmount()
  })
})
