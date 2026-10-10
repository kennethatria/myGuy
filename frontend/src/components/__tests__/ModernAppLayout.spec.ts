import { describe, it, expect, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ModernAppLayout from '../ModernAppLayout.vue'
import { useAuthStore } from '@/stores/auth'

const Page = { template: '<div />' }

const setUp = async (path: string) => {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: Page },
      { path: '/tasks', name: 'tasks', component: Page },
      { path: '/store', name: 'store', component: Page },
      { path: '/reviews', name: 'reviews', component: Page },
      { path: '/my-gigs/:tab', name: 'my-gigs', component: Page },
      { path: '/profile', name: 'profile', component: Page },
      { path: '/login', name: 'login', component: Page },
      { path: '/tasks/new', name: 'create-task', component: Page },
      { path: '/store/new', name: 'create-listing', component: Page },
      { path: '/store/wanted/new', name: 'create-request', component: Page }
    ]
  })
  router.push(path)
  await router.isReady()
  const wrapper = mount(ModernAppLayout, { global: { plugins: [router] }, attachTo: document.body })
  return { wrapper, router }
}

const isOpen = (wrapper: Awaited<ReturnType<typeof setUp>>['wrapper']) =>
  wrapper.find('#app-drawer').classes().includes('open')

const navItem = (wrapper: Awaited<ReturnType<typeof setUp>>['wrapper'], text: string) =>
  wrapper.findAll('.nav-item').find(a => a.text() === text)!

describe('ModernAppLayout drawer', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('closes when the item for the current page is tapped again', async () => {
    const { wrapper, router } = await setUp('/tasks')
    await wrapper.find('button[aria-label="Menu"]').trigger('click')
    expect(isOpen(wrapper)).toBe(true)

    await navItem(wrapper, 'Gigs').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.name).toBe('tasks')
    expect(isOpen(wrapper)).toBe(false)
    wrapper.unmount()
  })

  it('closes and navigates when another item is tapped', async () => {
    const { wrapper, router } = await setUp('/tasks')
    await wrapper.find('button[aria-label="Menu"]').trigger('click')

    await navItem(wrapper, 'Marketplace').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.name).toBe('store')
    expect(isOpen(wrapper)).toBe(false)
    wrapper.unmount()
  })

  it('goes to sign-in when the session ends (a blocked account is signed out)', async () => {
    const auth = useAuthStore()
    auth.token = 'token'
    const { wrapper, router } = await setUp('/tasks')
    auth.logout()
    await flushPromises()
    expect(router.currentRoute.value.name).toBe('login')
    wrapper.unmount()
  })
})
