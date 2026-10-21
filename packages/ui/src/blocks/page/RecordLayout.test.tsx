import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { ActionBar, RecordColumns } from './RecordLayout.tsx'
import '../../styles/index.css'

describe('RecordColumns', () => {
  it('splits main/side at exactly 1.6:2.6 by default and 62:100 with ratio="document" from 64em, and stacks below it', async () => {
    const { container, rerender } = await render(
      <RecordColumns main={<div data-testid="main">Main</div>} side={<div data-testid="side">Side</div>} />,
    )
    const columns = container.firstElementChild as HTMLElement
    const main = container.querySelector('[data-testid="main"]') as HTMLElement
    const side = container.querySelector('[data-testid="side"]') as HTMLElement

    await page.viewport(1300, 900)
    // The grid's own column-gap, not a hard-coded space token: it still holds if that token changes.
    let gap = parseFloat(getComputedStyle(columns).columnGap)
    let available = columns.getBoundingClientRect().width - gap
    let expectedMain = available * (1.6 / 2.6)
    let expectedSide = available * (1 / 2.6)
    expect(main.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedMain - 2)
    expect(main.getBoundingClientRect().width).toBeLessThanOrEqual(expectedMain + 2)
    expect(side.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedSide - 2)
    expect(side.getBoundingClientRect().width).toBeLessThanOrEqual(expectedSide + 2)

    await rerender(<RecordColumns ratio="document" main={<div data-testid="main">Main</div>} side={<div data-testid="side">Side</div>} />)
    gap = parseFloat(getComputedStyle(columns).columnGap)
    available = columns.getBoundingClientRect().width - gap
    expectedMain = available * (62 / 100)
    expectedSide = available * (38 / 100)
    expect(main.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedMain - 2)
    expect(main.getBoundingClientRect().width).toBeLessThanOrEqual(expectedMain + 2)
    expect(side.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedSide - 2)
    expect(side.getBoundingClientRect().width).toBeLessThanOrEqual(expectedSide + 2)

    await page.viewport(800, 900)
    await expect.poll(() => side.getBoundingClientRect().top).toBeGreaterThanOrEqual(main.getBoundingClientRect().bottom)
  })
})

describe('ActionBar', () => {
  it('renders nothing from 64em, and sits fixed at the viewport bottom below it', async () => {
    const { container } = await render(
      <ActionBar>
        <button type="button">Save</button>
      </ActionBar>,
    )
    const bar = (container.querySelector('button') as HTMLElement).parentElement as HTMLElement

    await page.viewport(1300, 900)
    await expect.poll(() => getComputedStyle(bar).display).toBe('none')

    await page.viewport(800, 900)
    await expect.poll(() => getComputedStyle(bar).display).not.toBe('none')
    await expect.poll(() => Math.abs(bar.getBoundingClientRect().bottom - 900)).toBeLessThanOrEqual(1)
  })
})
