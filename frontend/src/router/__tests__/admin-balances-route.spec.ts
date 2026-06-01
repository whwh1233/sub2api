import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('admin balances route', () => {
  it('registers an admin-only balance overview page', () => {
    expect(routerSource).toContain("path: '/admin/balances'")
    expect(routerSource).toContain("name: 'AdminBalances'")
    expect(routerSource).toContain("component: () => import('@/views/admin/BalancesView.vue')")
    expect(routerSource).toContain("titleKey: 'admin.balances.title'")
    expect(routerSource).toMatch(/path: '\/admin\/balances'[\s\S]*?requiresAdmin: true/)
  })
})
