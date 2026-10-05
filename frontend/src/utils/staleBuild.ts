// After a deploy, a tab opened on the previous build still asks for that
// build's page chunks. They are gone, so the lazy route import fails and the
// router cancels the navigation: the click appears to do nothing. Loading the
// target page fresh fetches the new build.

const RELOADED_KEY = 'myguy:stale-build-reload'

// Chrome, Firefox and Safari word a failed dynamic import differently; Vite
// throws "Unable to preload CSS" when a route's stylesheet is missing.
const CHUNK_ERROR = /Failed to fetch dynamically imported module|error loading dynamically imported module|Importing a module script failed|Unable to preload CSS/i

export function isStaleBuildError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error)
  return CHUNK_ERROR.test(message)
}

/**
 * Reload into href if error means this tab runs an outdated build. Returns
 * whether it reloaded. Only once per href, so a genuine outage (the chunk is
 * missing even after reloading) can't loop; without session storage to
 * remember that, it doesn't reload at all.
 */
export function reloadOnStaleBuild(
  error: unknown,
  href: string,
  storage: Pick<Storage, 'getItem' | 'setItem'> | null = safeSessionStorage(),
  load: (href: string) => void = (url) => window.location.assign(url)
): boolean {
  if (!storage || !isStaleBuildError(error)) return false
  if (storage.getItem(RELOADED_KEY) === href) return false
  storage.setItem(RELOADED_KEY, href)
  load(href)
  return true
}

/** Call after a successful navigation so a later deploy can reload again. */
export function clearStaleBuildReload(storage: Pick<Storage, 'removeItem'> | null = safeSessionStorage()) {
  storage?.removeItem(RELOADED_KEY)
}

function safeSessionStorage(): Storage | null {
  try {
    return window.sessionStorage
  } catch {
    return null
  }
}
