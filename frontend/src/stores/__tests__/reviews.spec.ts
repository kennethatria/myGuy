import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({ user: { id: 1 }, token: 'token' }))
}))
vi.mock('@/stores/user', () => ({
  useUserStore: vi.fn(() => ({
    fetchUsers: vi.fn(),
    getUserById: (id: number) => ({ id, username: `user${id}` })
  }))
}))

import { useReviewsStore } from '@/stores/reviews'

const taskReviews = [
  { id: 1, task_id: 4, rating: 5, comment: 'fast work', created_at: '2026-03-01T10:00:00Z', reviewer: { id: 2, username: 'ann' }, task: { id: 4, title: 'Paint fence' } }
]
const storeRatings = [
  { booking_id: 9, item_id: 3, item_title: 'Bike', rater_id: 6, rated_as: 'seller', rating: 3, review: 'ok', rated_at: '2026-04-01T10:00:00Z' }
]

function mockFetch(storeOk = true) {
  globalThis.fetch = vi.fn((url: string) => {
    if (url.includes('/ratings')) {
      return Promise.resolve({ ok: storeOk, json: () => Promise.resolve(storeRatings) })
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve(taskReviews) })
  }) as unknown as typeof fetch
}

describe('reviews store: combined rating', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('merges task reviews and store ratings, newest first', async () => {
    mockFetch()
    const store = useReviewsStore()

    const all = await store.fetchAllRatings(1)

    expect(all.map(r => r.id)).toEqual(['store-9', 1])
    expect(all[0]).toMatchObject({
      rating: 3,
      comment: 'ok',
      reviewer: { id: 6, username: 'user6' },
      item: { id: 3, title: 'Bike', ratedAs: 'seller' }
    })
    expect(store.calculateAverageRating(all)).toBe(4) // (5 + 3) / 2
  })

  it('still shows task reviews when store ratings are unavailable', async () => {
    mockFetch(false)
    const store = useReviewsStore()

    const all = await store.fetchAllRatings(1)

    expect(all.map(r => r.id)).toEqual([1])
  })
})
