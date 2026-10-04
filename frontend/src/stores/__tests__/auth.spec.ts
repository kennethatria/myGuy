import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/stores/user', () => ({ useUserStore: vi.fn(() => ({ cacheCurrentUser: vi.fn(), clearCache: vi.fn() })) }))

import { useAuthStore } from '@/stores/auth'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.setItem('token', 'token')
  })

  it('maps the API user (snake_case) to the fields the app reads', async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ id: 1, username: 'sam', email: 's@example.com', full_name: 'Sam Okello', average_rating: 4.5 })
    })) as unknown as typeof fetch
    const store = useAuthStore()

    await store.checkAuth()

    expect(store.user?.fullName).toBe('Sam Okello')
    expect(store.user?.averageRating).toBe(4.5)
  })
})
