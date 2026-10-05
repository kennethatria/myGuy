import { describe, it, expect, vi } from 'vitest'
import { isStaleBuildError, reloadOnStaleBuild, clearStaleBuildReload } from '../staleBuild'

function memoryStorage() {
  const data = new Map<string, string>()
  return {
    getItem: (k: string) => data.get(k) ?? null,
    setItem: (k: string, v: string) => void data.set(k, v),
    removeItem: (k: string) => void data.delete(k)
  }
}

describe('staleBuild', () => {
  it('recognises missing-chunk errors from each browser and Vite', () => {
    for (const message of [
      'Failed to fetch dynamically imported module: https://x/assets/CreateTaskView-C_aMDSxu.js',
      'error loading dynamically imported module',
      'Importing a module script failed.',
      'Unable to preload CSS for /assets/CreateTaskView-BTo-05Ma.css'
    ]) {
      expect(isStaleBuildError(new TypeError(message))).toBe(true)
    }
    expect(isStaleBuildError(new Error('Failed to load the gig'))).toBe(false)
  })

  it('reloads into the page the user clicked', () => {
    const load = vi.fn()
    const reloaded = reloadOnStaleBuild(new TypeError('Failed to fetch dynamically imported module'), '/tasks/create', memoryStorage(), load)
    expect(reloaded).toBe(true)
    expect(load).toHaveBeenCalledWith('/tasks/create')
  })

  it('ignores unrelated errors', () => {
    const load = vi.fn()
    expect(reloadOnStaleBuild(new Error('boom'), '/tasks/create', memoryStorage(), load)).toBe(false)
    expect(load).not.toHaveBeenCalled()
  })

  it('reloads only once per page until a navigation succeeds', () => {
    const storage = memoryStorage()
    const load = vi.fn()
    const error = new TypeError('Failed to fetch dynamically imported module')

    reloadOnStaleBuild(error, '/tasks/create', storage, load)
    expect(reloadOnStaleBuild(error, '/tasks/create', storage, load)).toBe(false)
    expect(load).toHaveBeenCalledTimes(1)

    clearStaleBuildReload(storage)
    expect(reloadOnStaleBuild(error, '/tasks/create', storage, load)).toBe(true)
  })

  it('never reloads without session storage, so it cannot loop', () => {
    const load = vi.fn()
    expect(reloadOnStaleBuild(new TypeError('Importing a module script failed.'), '/store', null, load)).toBe(false)
    expect(load).not.toHaveBeenCalled()
  })
})
