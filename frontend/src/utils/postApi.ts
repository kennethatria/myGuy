// One post of any kind (gig, marketplace item, request) for the screens
// that treat them alike: Posted, Edit. Gigs live in the backend, items and
// requests in store-service; each service checks the note's rules again.
import config from '@/config'
import { useAuthStore } from '@/stores/auth'
import type { PostKind } from '@/utils/radar'

export type { PostKind }

/** The note colour for each kind (StickyNote `tone`). */
export const TONE = { task: 'gig', item: 'sell', request: 'want' } as const

/** Kinds whose headline and note can be changed after posting. */
export const EDITABLE: readonly PostKind[] = ['task', 'item']

export interface PostNote {
  id: number
  title: string
  description: string
  deadline: string
  status: string
}

const URLS: Record<PostKind, (id: number) => string> = {
  task: (id) => `${config.API_URL}/tasks/${id}`,
  item: (id) => `${config.STORE_API_URL}/items/${id}`,
  request: (id) => `${config.STORE_API_URL}/requests/${id}`
}

/** The page that shows a post. */
export function postRoute(kind: PostKind, id: number) {
  if (kind === 'task') return { name: 'task-detail', params: { id } }
  if (kind === 'item') return { name: 'store-item', params: { id } }
  return { name: 'store-request', params: { id } }
}

async function call(kind: PostKind, id: number, init: RequestInit = {}): Promise<Record<string, unknown>> {
  const headers: Record<string, string> = { Authorization: `Bearer ${useAuthStore().token}` }
  if (init.body) headers['Content-Type'] = 'application/json'
  const response = await fetch(URLS[kind](id), { ...init, headers })
  const data = await response.json().catch(() => ({}))
  // The services word their errors for people (limits, contact details)
  if (!response.ok) throw new Error(typeof data.error === 'string' ? data.error : 'Something went wrong. Please try again.')
  return data
}

export async function fetchPost(kind: PostKind, id: number): Promise<PostNote> {
  const data = await call(kind, id)
  return {
    id: Number(data.id),
    title: String(data.title ?? ''),
    description: String(data.description ?? ''),
    deadline: String(data.deadline ?? ''),
    status: String(data.status ?? '')
  }
}

export async function updatePost(kind: PostKind, id: number, note: { title: string; description: string }): Promise<void> {
  await call(kind, id, { method: 'PUT', body: JSON.stringify(note) })
}

export async function removePost(kind: PostKind, id: number): Promise<void> {
  await call(kind, id, { method: 'DELETE' })
}
