import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'

const { createAccountMock, showErrorMock, buildOAuthStub } = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  showErrorMock: vi.fn(),
  buildOAuthStub: () => ({
    authUrl: { value: '' },
    sessionId: { value: '' },
    state: { value: '' },
    loading: { value: false },
    error: { value: '' },
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    exchangeAuthCode: vi.fn(),
    validateRefreshToken: vi.fn(),
    buildCredentials: vi.fn(() => ({})),
    buildExtraInfo: vi.fn(() => ({}))
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isSimpleMode: true })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createAccountMock,
      checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false })
    },
    settings: {
      getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] })
    },
    tlsFingerprintProfiles: {
      list: vi.fn().mockResolvedValue([])
    }
  }
}))

vi.mock('@/composables/useQuotaNotifyState', () => ({
  useQuotaNotifyState: () => ({
    globalEnabled: { value: false },
    state: {
      daily: { enabled: null, threshold: null, thresholdType: null },
      weekly: { enabled: null, threshold: null, thresholdType: null },
      total: { enabled: null, threshold: null, thresholdType: null }
    },
    loadGlobalState: vi.fn(),
    loadFromExtra: vi.fn(),
    writeToExtra: vi.fn(),
    reset: vi.fn()
  })
}))

vi.mock('@/composables/useAccountOAuth', () => ({ useAccountOAuth: buildOAuthStub }))
vi.mock('@/composables/useOpenAIOAuth', () => ({ useOpenAIOAuth: buildOAuthStub }))
vi.mock('@/composables/useGeminiOAuth', () => ({ useGeminiOAuth: buildOAuthStub }))
vi.mock('@/composables/useAntigravityOAuth', () => ({ useAntigravityOAuth: buildOAuthStub }))
vi.mock('@/composables/useGrokOAuth', () => ({ useGrokOAuth: buildOAuthStub }))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const mountModal = () =>
  shallowMount(CreateAccountModal, {
    props: {
      show: true,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        QuotaLimitCard: { template: '<div />' }
      }
    }
  })

describe('CreateAccountModal Grok API Key', () => {
  beforeEach(() => {
    createAccountMock.mockReset()
    createAccountMock.mockResolvedValue({})
    showErrorMock.mockReset()
  })

  it('shows the Base URL and API Key fields after selecting Grok API Key', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.get('[data-testid="create-platform-grok"]').trigger('click')
    await wrapper.get('[data-testid="create-grok-apikey-type"]').trigger('click')

    const baseURLInput = wrapper.get('[data-testid="create-apikey-base-url-input"]')
    const apiKeyInput = wrapper.get('[data-testid="create-apikey-value-input"]')

    expect((baseURLInput.element as HTMLInputElement).value).toBe('https://api.x.ai/v1')
    expect(baseURLInput.attributes('placeholder')).toBe('https://api.x.ai/v1')
    expect(apiKeyInput.attributes('placeholder')).toBe('xai-...')
    expect(wrapper.find('[data-testid="create-grok-oauth-type"]').exists()).toBe(true)
  })

  it('submits a Grok apikey account with the entered Base URL and API Key', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.get('[data-testid="create-account-name-input"]').setValue('Grok API Key')
    await wrapper.get('[data-testid="create-platform-grok"]').trigger('click')
    await wrapper.get('[data-testid="create-grok-apikey-type"]').trigger('click')
    await wrapper.get('[data-testid="create-apikey-base-url-input"]').setValue(' https://xai.example/v1/ ')
    await wrapper.get('[data-testid="create-apikey-value-input"]').setValue(' xai-test-key ')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(showErrorMock).not.toHaveBeenCalled()
    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Grok API Key',
      platform: 'grok',
      type: 'apikey',
      credentials: expect.objectContaining({
        base_url: 'https://xai.example/v1/',
        api_key: 'xai-test-key'
      })
    }))
  })

  it('falls back to the official xAI Base URL when the field is blank', async () => {
    const wrapper = mountModal()
    await flushPromises()

    await wrapper.get('[data-testid="create-account-name-input"]').setValue('Official xAI')
    await wrapper.get('[data-testid="create-platform-grok"]').trigger('click')
    await wrapper.get('[data-testid="create-grok-apikey-type"]').trigger('click')
    await wrapper.get('[data-testid="create-apikey-base-url-input"]').setValue('')
    await wrapper.get('[data-testid="create-apikey-value-input"]').setValue('xai-test-key')
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.credentials).toEqual(expect.objectContaining({
      base_url: 'https://api.x.ai/v1',
      api_key: 'xai-test-key'
    }))
  })
})
