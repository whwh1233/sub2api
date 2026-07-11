import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { getRawExchangeLog, listRawExchangeLogs, type RawExchangeLogListResponse } from '@/api/admin/ops'

describe('admin ops raw exchange logs api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads raw exchange logs with filter params from the dedicated endpoint', async () => {
    const response: RawExchangeLogListResponse = {
      items: [
        {
          line: 4,
          offset: 1024,
          stage: 'client_exchange',
          operation: '',
          attempt: 0,
          completed_at: '2026-07-10T12:00:00Z',
          started_at: '',
          request_id: 'req-1',
          client_request_id: 'client-1',
          method: 'POST',
          path: '/v1/chat/completions',
          request_uri: '/v1/chat/completions?debug=1',
          url: '',
          raw_query: 'debug=1',
          status_code: 200,
          latency_ms: 42,
          client_ip: '127.0.0.1',
          protocol: 'HTTP/1.1',
          platform: 'openai',
          model: 'gpt-test',
          account_id: 7,
          user_id: 8,
          request_body_bytes: 11,
          request_body_truncated: false,
          response_body_bytes: 13,
          response_body_truncated: false,
          raw: {
            request_headers: { Authorization: ['Bearer secret'] },
            request_body_base64: 'eyJmb28iOjF9',
          },
        },
      ],
      total: 1,
      path: '/data/raw-exchange/raw-exchange.jsonl',
    }
    get.mockResolvedValue({ data: response })

    const result = await listRawExchangeLogs({
      limit: 20,
      q: 'secret',
      request_id: 'req-1',
      path: '/v1/chat',
      method: 'POST',
      status_code: 200,
    })

    expect(get).toHaveBeenCalledWith('/admin/ops/raw-exchange-logs', {
      params: {
        limit: 20,
        q: 'secret',
        request_id: 'req-1',
        path: '/v1/chat',
        method: 'POST',
        status_code: 200,
      },
    })
    expect(result).toEqual(response)
  })

  it('loads one complete raw record by byte offset', async () => {
    const detail = { offset: 1024, raw: { request_body: 'secret' } }
    get.mockResolvedValue({ data: detail })

    await expect(getRawExchangeLog(1024)).resolves.toEqual(detail)
    expect(get).toHaveBeenCalledWith('/admin/ops/raw-exchange-logs', { params: { offset: 1024 } })
  })
})
