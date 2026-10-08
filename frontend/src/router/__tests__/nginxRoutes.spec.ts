import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import router from '@/router'

// nginx answers 404 for paths that aren't app routes ($spa_route in
// configuration_management/templates/nginx-common.conf.j2), so every route
// added here must be added there too.
// Tests run from frontend/ (vitest root).
const template = readFileSync(
  resolve(process.cwd(), '../configuration_management/templates/nginx-common.conf.j2'),
  'utf8',
)
const pattern = template.match(/map \$uri \$spa_route \{[^}]*?"~(\^[^"]+)" 1;/)?.[1]
const spaRoute = new RegExp(pattern ?? '$^')

// A concrete URL for a route path: a constrained param takes its first
// option (or a number), a free one any segment.
function sample(path: string): string {
  return path
    .replace(/:\w+\((\\\\d\+|\\d\+)\)/g, '7')
    .replace(/:\w+\(([^|)]+)[^)]*\)/g, '$1')
    .replace(/:\w+/g, 'x1')
}

describe('nginx app routes', () => {
  it('reads the route pattern from the nginx template', () => {
    expect(pattern).toBeTruthy()
  })

  const paths = router
    .getRoutes()
    .map((route) => route.path)
    .filter((path) => !path.includes(':pathMatch'))

  it.each(paths)('serves %s as an app page', (path) => {
    expect(sample(path)).toMatch(spaRoute)
    expect(`${sample(path)}/`).toMatch(spaRoute)
  })

  it.each(['/.env', '/wp-login.php', '/containers/json', '/SDK/webLanguage', '/geoserver/web/', '/cgi-bin/index2.asp', '/tasks/1/extra'])(
    'answers %s with a 404',
    (path) => {
      expect(path).not.toMatch(spaRoute)
    },
  )
})
