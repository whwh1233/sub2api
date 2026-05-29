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
  'community.scanLabel': '扫码加入',
  'community.channels.wechat.title': '微信群',
  'community.channels.wechat.caption': '微信扫码加入 CloseAI 天使用户服务群',
  'community.channels.wechat.hint': '二维码有效期以图片内提示为准。',
  'community.channels.qq.title': 'QQ群',
  'community.channels.qq.caption': 'QQ扫码加入 CloseAI 使用答疑群',
  'community.channels.qq.hint': '群号：1102975321',
  'community.channels.telegram.title': 'Telegram群',
  'community.channels.telegram.caption': 'Telegram扫码加入 CloseAI 交流群',
  'community.channels.telegram.hint': '也可使用邀请链接 https://t.me/+JFjjTyQlBLYyYzRl',
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

  it('renders all community QR codes without tab switching', () => {
    expect(existsSync(viewPath)).toBe(true)

    const viewSource = readFileSync(viewPath, 'utf8')
    const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
    const enLocaleSource = readFileSync(enLocalePath, 'utf8')

    expect(viewSource).toContain('@/assets/community-wechat-qr.png')
    expect(viewSource).toContain('@/assets/community-qq-qr.jpg')
    expect(viewSource).toContain('@/assets/community-telegram-qr.png')
    expect(viewSource).toContain('CloseAI 天使用户服务群二维码')
    expect(viewSource).toContain('data-test="community-notice-intro"')
    expect(viewSource).toContain('data-test="community-notice-tip"')
    expect(viewSource).toContain('data-test="community-qr-grid"')
    expect(viewSource).toContain('data-test="community-qr-card"')
    expect(viewSource).toContain('min-h-[calc(100dvh-7rem)]')
    expect(viewSource).toContain('lg:grid-cols-3')
    expect(viewSource).toContain('max-w-[520px]')
    expect(viewSource).not.toContain('role="tablist"')
    expect(viewSource).not.toContain('role="tab"')
    expect(viewSource).not.toContain('@click="activeChannelKey')
    expect(viewSource).not.toContain('activeChannelKey')
    expect(zhLocaleSource).toContain("title: '加入交流群 / 联系我'")
    expect(zhLocaleSource).toContain('欢迎加入我们的社群')
    expect(zhLocaleSource).toContain('群内禁止发布广告、外链与无关内容')
    expect(zhLocaleSource).toContain("scanLabel: '扫码加入'")
    expect(zhLocaleSource).toContain('群号：1102975321')
    expect(zhLocaleSource).toContain('https://t.me/+JFjjTyQlBLYyYzRl')
    expect(enLocaleSource).toContain("title: 'Join the Community / Contact Me'")
    expect(enLocaleSource).toContain("scanLabel: 'Scan to join'")

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
    expect(wrapper.find('[data-test="community-notice-intro"]').text()).toContain('遇到问题可以在群内反馈')
    expect(wrapper.find('[data-test="community-notice-tip"]').text()).toContain('请尽量附带截图与复现步骤')
    expect(wrapper.findAll('[data-test="community-qr-card"]')).toHaveLength(3)

    expect(wrapper.findAll('[role="tab"]')).toHaveLength(0)
    expect(wrapper.findAll('button')).toHaveLength(0)

    const qrImages = wrapper.findAll('img')
    expect(qrImages).toHaveLength(3)
    expect(qrImages.map((image) => image.attributes('alt'))).toEqual([
      'CloseAI 天使用户服务群二维码',
      'CloseAI 使用答疑群二维码',
      'CloseAI Telegram 交流群二维码',
    ])
    expect(qrImages[0].attributes('src')).toContain('community-wechat-qr')
    expect(qrImages[1].attributes('src')).toContain('community-qq-qr')
    expect(qrImages[2].attributes('src')).toContain('community-telegram-qr')
    expect(wrapper.text()).toContain('微信扫码加入 CloseAI 天使用户服务群')
    expect(wrapper.text()).toContain('群号：1102975321')
    expect(wrapper.text()).toContain('https://t.me/+JFjjTyQlBLYyYzRl')
  })
})
