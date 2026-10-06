import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useTasksStore } from '@/stores/tasks'
import { useReviewsStore } from '@/stores/reviews'
import TaskEventMessage from '../TaskEventMessage.vue'
import type { Message, TaskEvent } from '@/stores/messages'

const POSTER = 1
const HELPER = 2

// An event about gig 7, from one person to the other
const event = (name: TaskEvent, from: number, to: number): Message => ({
  id: 1, task_id: 7, sender_id: from, recipient_id: to, content: `event ${name}`,
  message_type: 'system_alert', metadata: { event: name, application_id: 4 },
  is_read: false, is_edited: false, is_deleted: false, created_at: '2026-10-06T10:00:00Z'
} as Message)

const show = (message: Message, viewer: number, latest = true) =>
  mount(TaskEventMessage, { props: { message, currentUserId: viewer, latest } })

const buttons = (wrapper: ReturnType<typeof show>) => wrapper.findAll('button').map(b => b.text())

describe('TaskEventMessage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('asks the poster to accept or decline a new application', async () => {
    const wrapper = show(event('application', HELPER, POSTER), POSTER)
    expect(buttons(wrapper)).toEqual(['Accept', 'Decline'])

    const respond = vi.spyOn(useTasksStore(), 'respondToApplication').mockResolvedValue({} as never)
    await wrapper.findAll('button')[0].trigger('click')
    expect(respond).toHaveBeenCalledWith(7, 4, 'accepted')
  })

  it('shows the applicant their application without actions', () => {
    expect(buttons(show(event('application', HELPER, POSTER), HELPER))).toEqual([])
  })

  it('lets the accepted helper mark the gig done, and again after "not yet"', async () => {
    for (const name of ['accepted', 'not_done'] as const) {
      setActivePinia(createPinia())
      const wrapper = show(event(name, POSTER, HELPER), HELPER)
      expect(buttons(wrapper)).toEqual(['Mark as done'])

      const update = vi.spyOn(useTasksStore(), 'updateTaskStatus').mockResolvedValue({} as never)
      await wrapper.find('button').trigger('click')
      expect(update).toHaveBeenCalledWith(7, 'pending_approval')
    }
  })

  it('asks the poster to approve finished work, or say not yet', async () => {
    const wrapper = show(event('done', HELPER, POSTER), POSTER)
    expect(buttons(wrapper)).toEqual(['Approve', 'Not yet'])

    const update = vi.spyOn(useTasksStore(), 'updateTaskStatus').mockResolvedValue({} as never)
    await wrapper.findAll('button')[1].trigger('click')
    expect(update).toHaveBeenCalledWith(7, 'in_progress')
    expect(buttons(show(event('done', HELPER, POSTER), HELPER))).toEqual([])
  })

  it('offers both people a review once the gig is complete, comment optional', async () => {
    vi.spyOn(useReviewsStore(), 'hasReviewedTask').mockResolvedValue(false)
    const create = vi.spyOn(useReviewsStore(), 'createReview').mockResolvedValue({})

    for (const viewer of [POSTER, HELPER]) {
      const wrapper = show(event('completed', POSTER, HELPER), viewer)
      await flushPromises()
      expect(wrapper.findAll('.star')).toHaveLength(5)
    }

    const wrapper = show(event('completed', POSTER, HELPER), HELPER)
    await flushPromises()
    await wrapper.findAll('.star')[3].trigger('click')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(create).toHaveBeenCalledWith(7, { rating: 4, comment: '' })
    expect(wrapper.text()).toContain('your review is in')
  })

  it('offers nothing on older events: only the newest one is current', () => {
    expect(buttons(show(event('application', HELPER, POSTER), POSTER, false))).toEqual([])
  })

  it('shows why a step failed', async () => {
    const wrapper = show(event('done', HELPER, POSTER), POSTER)
    vi.spyOn(useTasksStore(), 'updateTaskStatus').mockRejectedValue(new Error('Invalid status transition'))
    await wrapper.find('button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').text()).toBe('Invalid status transition')
  })
})
