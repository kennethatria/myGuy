import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import TaskDetailView from '@/views/tasks/TaskDetailView.vue'

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({ user: { id: 1, username: 'helper' }, token: 'mock-token' }))
}))

const openChat = vi.fn()
vi.mock('@/stores/chat', () => ({
  useChatStore: vi.fn(() => ({ openChat, connected: true }))
}))

const gig = {
  id: 3,
  title: 'Carry boxes upstairs',
  description: 'Ten boxes to the third floor',
  status: 'open',
  created_by: 2,
  creator: { id: 2, username: 'ann' },
  deadline: new Date(Date.now() + 3_600_000).toISOString(),
  created_at: new Date().toISOString()
}

const tasksStore = {
  getTask: vi.fn(),
  getTaskApplications: vi.fn(),
  applyForTask: vi.fn()
}
vi.mock('@/stores/tasks', () => ({ useTasksStore: vi.fn(() => tasksStore) }))
vi.mock('@/stores/users', () => ({ useUsersStore: vi.fn(() => ({ getUserById: vi.fn() })) }))

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/tasks/:id', name: 'task-detail', component: TaskDetailView }]
})

describe('Applying for a gig', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    tasksStore.getTask.mockResolvedValue({ ...gig })
    tasksStore.getTaskApplications.mockResolvedValue([])
    tasksStore.applyForTask.mockResolvedValue({})
    await router.push('/tasks/3')
  })

  it('opens the conversation with the poster, like booking an item', async () => {
    const wrapper = mount(TaskDetailView, {
      global: { plugins: [router, createPinia()], stubs: { 'router-link': true } }
    })
    await flushPromises()

    // After applying, the poster lists this person's application
    tasksStore.getTaskApplications.mockResolvedValue([{ id: 9, applicant: { id: 1, username: 'helper' }, status: 'pending' }])
    const apply = wrapper.findAll('button').find((b) => b.text() === 'Apply')
    expect(apply).toBeTruthy()
    await apply.trigger('click')
    await flushPromises()

    expect(tasksStore.applyForTask).toHaveBeenCalledWith(3, { message: '' })
    expect(openChat).toHaveBeenCalledWith({ taskId: 3, otherUserId: 2, otherUserName: 'ann' })
  })

  it('stays on the page when applying fails', async () => {
    tasksStore.applyForTask.mockRejectedValue(new Error('This gig has expired'))
    const wrapper = mount(TaskDetailView, {
      global: { plugins: [router, createPinia()], stubs: { 'router-link': true } }
    })
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Apply').trigger('click')
    await flushPromises()

    expect(openChat).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('This gig has expired')
  })
})
