import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import CommunityView from '@/views/user/CommunityView.vue'

const messages: Record<string, string> = {
  'community.title': '加入交流群 / 联系我',
  'community.description': '欢迎加入我们的社群，遇到问题可以在群内反馈，也会同步最新更新与活动。',
  'community.introTitle': '欢迎加入我们的社群',
  'community.introText': '遇到问题可以在群内反馈，也会同步最新更新与活动。',
  'community.tipTitle': '温馨提示',
  'community.tipText': '群内禁止发布广告、外链与无关内容。如需反馈 Bug 或建议，请尽量附带截图与复现步骤，我们会尽快响应。',
  'community.qrCaption': '微信扫码加入 CloseAI 天使用户服务群',
  'community.scanHint': '二维码 7 天内有效，过期后请重新进入页面获取更新。',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const testDir = dirname(fileURLToPath(import.meta.url))
const viewPath = resolve(testDir, '../CommunityView.vue')
const routerPath = resolve(testDir, '../../../router/index.ts')
const zhLocalePath = resolve(testDir, '../../../i18n/locales/zh.ts')
const enLocalePath = resolve(testDir, '../../../i18n/locales/en.ts')

describe('CommunityView', () => {
  it('is routed as an authenticated user page', () => {
    const routerSource = readFileSync(routerPath, 'utf8')

    expect(routerSource).toContain("path: '/community'")
    expect(routerSource).toContain("name: 'Community'")
    expect(routerSource).toContain("component: () => import('@/views/user/CommunityView.vue')")
    expect(routerSource).toContain("titleKey: 'community.title'")
    expect(routerSource).toContain("descriptionKey: 'community.description'")
  })

  it('renders the WeChat group QR code and community guidance copy', () => {
    expect(existsSync(viewPath)).toBe(true)

    const viewSource = readFileSync(viewPath, 'utf8')
    const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
    const enLocaleSource = readFileSync(enLocalePath, 'utf8')

    expect(viewSource).toContain('@/assets/community-wechat-qr.png')
    expect(viewSource).toContain("alt=\"CloseAI 天使用户服务群二维码\"")
    expect(zhLocaleSource).toContain("title: '加入交流群 / 联系我'")
    expect(zhLocaleSource).toContain('欢迎加入我们的社群')
    expect(zhLocaleSource).toContain('群内禁止发布广告、外链与无关内容')
    expect(enLocaleSource).toContain("title: 'Join the Community / Contact Me'")

    const wrapper = mount(CommunityView, {
      global: {
        stubs: {
          AppLayout: { template: '<main><slot /></main>' },
          Icon: true,
        },
      },
    })

    expect(wrapper.text()).toContain('加入交流群 / 联系我')
    expect(wrapper.text()).toContain('欢迎加入我们的社群')
    expect(wrapper.text()).toContain('温馨提示')
    expect(wrapper.text()).toContain('群内禁止发布广告、外链与无关内容')
    const qrImage = wrapper.get('img[alt="CloseAI 天使用户服务群二维码"]')
    expect(qrImage.attributes('src')).toContain('community-wechat-qr')
  })
})
