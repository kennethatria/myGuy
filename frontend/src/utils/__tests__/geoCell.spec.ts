import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { cell, roughLocation, CELL_STEP } from '../geoCell'

const cases = JSON.parse(
  // tests run from frontend/
  readFileSync(resolve(process.cwd(), '../shared/geo-cell-cases.json'), 'utf8')
) as { step: number; cells: { degrees: number; cell: number }[] }

describe('geoCell', () => {
  it('matches the shared cell cases', () => {
    expect(cases.step).toBe(CELL_STEP)
    for (const c of cases.cells) {
      expect(cell(c.degrees), `cell of ${c.degrees}`).toBe(c.cell)
    }
  })

  it('sends only the cell centre', () => {
    expect(roughLocation(0.3476, 32.5842)).toEqual({ lat: 0.35, lng: 32.585 })
    expect(roughLocation(-0.3376, 31.7321)).toEqual({ lat: -0.34, lng: 31.73 })
  })
})
