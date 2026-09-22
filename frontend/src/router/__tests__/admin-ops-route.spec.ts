import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('admin ops route', () => {
  it('keeps the RPM tab behind the existing admin-only page guard', () => {
    const routerPath = resolve(__dirname, '../index.ts')
    const routerSource = readFileSync(routerPath, 'utf8')

    expect(routerSource).toMatch(
      /path: '\/admin\/ops'[\s\S]*?requiresAuth: true[\s\S]*?requiresAdmin: true/
    )
  })
})
