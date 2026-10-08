import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({ token: 'token' }))
}))
vi.mock('@/config', () => ({
  default: { API_URL: 'http://api/api/v1', STORE_API_URL: 'http://store/api/v1' }
}))

import { fetchPost, updatePost, removePost, postRoute } from '../postApi'

let fetchMock: ReturnType<typeof vi.fn>

function respond(ok: boolean, body: unknown) {
  fetchMock = vi.fn(() => Promise.resolve({ ok, json: () => Promise.resolve(body) }))
  globalThis.fetch = fetchMock as unknown as typeof fetch
}

describe('postApi', () => {
  beforeEach(() => respond(true, {}))

  it('reads each kind from the service that owns it', async () => {
    respond(true, { id: 4, title: 'Paint fence', description: 'Saturday', deadline: '2026-10-09T10:00:00Z', status: 'open' })
    expect(await fetchPost('task', 4)).toEqual({ id: 4, title: 'Paint fence', description: 'Saturday', deadline: '2026-10-09T10:00:00Z', status: 'open' })
    expect(fetchMock.mock.calls[0][0]).toBe('http://api/api/v1/tasks/4')

    await fetchPost('item', 5)
    expect(fetchMock.mock.calls[1][0]).toBe('http://store/api/v1/items/5')
    await fetchPost('request', 6)
    expect(fetchMock.mock.calls[2][0]).toBe('http://store/api/v1/requests/6')
  })

  it('sends edits as JSON and removals as DELETE, signed in', async () => {
    await updatePost('item', 5, { title: 'Bike', description: 'Red' })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://store/api/v1/items/5')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body)).toEqual({ title: 'Bike', description: 'Red' })
    expect(init.headers).toEqual({ Authorization: 'Bearer token', 'Content-Type': 'application/json' })

    await removePost('task', 4)
    expect(fetchMock.mock.calls[1][1]).toMatchObject({ method: 'DELETE', headers: { Authorization: 'Bearer token' } })
  })

  it("passes on the service's message when it refuses", async () => {
    respond(false, { error: 'Keep contact details for the chat' })
    await expect(updatePost('task', 4, { title: 'Call 0700 000000', description: 'x' })).rejects.toThrow('Keep contact details for the chat')
    respond(false, {})
    await expect(removePost('request', 6)).rejects.toThrow('Something went wrong')
  })

  it('refuses a kind it does not know, without calling anything', async () => {
    await expect(fetchPost('constructor' as never, 1)).rejects.toThrow('That post does not exist.')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('links each kind to its page', () => {
    expect(postRoute('task', 1)).toEqual({ name: 'task-detail', params: { id: 1 } })
    expect(postRoute('item', 2)).toEqual({ name: 'store-item', params: { id: 2 } })
    expect(postRoute('request', 3)).toEqual({ name: 'store-request', params: { id: 3 } })
  })
})
