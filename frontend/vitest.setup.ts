import { vi } from 'vitest'

// Ensure localStorage is available before any module (e.g. @vue/devtools-kit)
// tries to access it during initialization in the jsdom environment.
if (typeof globalThis.localStorage === 'undefined' || typeof globalThis.localStorage.getItem !== 'function') {
  const storage: Record<string, string> = {}
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: vi.fn((key: string) => storage[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { storage[key] = value }),
      removeItem: vi.fn((key: string) => { delete storage[key] }),
      clear: vi.fn(() => { Object.keys(storage).forEach(k => delete storage[k]) }),
      get length() { return Object.keys(storage).length },
      key: vi.fn((i: number) => Object.keys(storage)[i] ?? null),
    },
    writable: true,
  })
}
