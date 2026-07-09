import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('admin raw exchange logs route', () => {
  it('registers an admin-only raw exchange log viewer', () => {
    expect(routerSource).toContain("path: '/admin/raw-exchange-logs'")
    expect(routerSource).toContain("name: 'AdminRawExchangeLogs'")
    expect(routerSource).toContain("component: () => import('@/views/admin/RawExchangeLogsView.vue')")
    expect(routerSource).toMatch(/path: '\/admin\/raw-exchange-logs'[\s\S]*?requiresAdmin: true/)
  })
})
