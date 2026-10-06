import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import CreateReviewView from '../CreateReviewView.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { taskId: '7' } }),
  useRouter: () => ({ push: vi.fn() })
}))

// The gig as the backend sends it: snake_case ids
const gig = {
  id: 7,
  title: 'Fix sink',
  status: 'completed',
  created_by: 1,
  assigned_to: 2,
  creator: { id: 1, username: 'ann' },
  assignee: { id: 2, username: 'bob' }
}

function respond(url: string) {
  const body = url.endsWith('/reviews/mine') ? { reviewed: false } : gig
  return Promise.resolve(new Response(JSON.stringify(body), { status: 200 }))
}

async function mountAs(userId: number) {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore().user = { id: userId, username: 'me' } as never
  const wrapper = mount(CreateReviewView, {
    global: { plugins: [pinia], stubs: { ReviewForm: true, RouterLink: true } }
  })
  await flushPromises()
  return wrapper
}

describe('CreateReviewView', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn((url: string) => respond(url)))
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('lets the poster review the person who did the gig', async () => {
    const wrapper = await mountAs(1)
    expect(wrapper.text()).toContain('You are reviewing: bob')
  })

  it('lets the person who did the gig review the poster', async () => {
    const wrapper = await mountAs(2)
    expect(wrapper.text()).toContain('You are reviewing: ann')
  })

  it('turns away someone who was not part of the gig', async () => {
    const wrapper = await mountAs(3)
    expect(wrapper.text()).toContain('You can only review gigs you posted or were assigned to.')
  })
})
