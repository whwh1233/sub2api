import { existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import en from '../locales/en/index'
import zh from '../locales/zh/index'

const localeRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../locales')

describe('custom domain locale migration', () => {
  it('exposes custom user and admin domains through the modular locale indexes', () => {
    expect(en.home.heroSubtitle).toBe('ChatGPT and Claude advanced model pools')
    expect(zh.home.heroSubtitle).toBe('ChatGPT 与 Claude 高阶模型池')
    expect(en.nav.community).toBe('Community')
    expect(zh.nav.dailyLeaderboard).toBe('每日榜单')
    expect(en.community.title).toBe('Join the Community / Contact Me')
    expect(zh.leaderboard.title).toBe('每日榜单')
    expect(en.admin.balances.title).toBe('Balance Overview')
    expect(zh.admin.balances.title).toBe('余额总览')
  })

  it('does not restore the deleted monolithic locale files', () => {
    expect(existsSync(resolve(localeRoot, 'en.ts'))).toBe(false)
    expect(existsSync(resolve(localeRoot, 'zh.ts'))).toBe(false)
  })
})
