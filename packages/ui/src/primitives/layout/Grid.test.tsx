import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { Grid } from './Grid.tsx'
import '../../styles/index.css'

describe('Grid.Col responsive span', () => {
  it('follows base below the md breakpoint and md from it up', async () => {
    const { container } = await render(
      <Grid>
        <Grid.Col span={{ base: 12, md: 4 }} data-testid="c" />
      </Grid>,
    )
    const grid = container.querySelector('div') as HTMLElement
    const col = container.querySelector('[data-testid="c"]') as HTMLElement

    await page.viewport(800, 600)
    const gridWidth = grid.getBoundingClientRect().width
    expect(col.getBoundingClientRect().width).toBeCloseTo(gridWidth, 0)

    await page.viewport(1300, 800)
    const gridWidthWide = grid.getBoundingClientRect().width
    const gap = 16 // tokens.space.md, the Grid's default gap
    const trackWidth = (gridWidthWide - gap * 11) / 12
    const expected = trackWidth * 4 + gap * 3 // a 4-track span also covers the 3 gaps between those tracks
    expect(col.getBoundingClientRect().width).toBeGreaterThanOrEqual(expected - 2)
    expect(col.getBoundingClientRect().width).toBeLessThanOrEqual(expected + 2)
  })
})
