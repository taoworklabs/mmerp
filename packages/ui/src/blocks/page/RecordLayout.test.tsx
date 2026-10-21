import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { ActionBar, RecordColumns } from './RecordLayout.tsx'
import '../../styles/index.css'

describe('RecordColumns', () => {
  it('holds the 1.6:1 ratio by default and 62:38 with ratio="document" from 64em, and stacks below it', async () => {
    const { container, rerender } = await render(
      <RecordColumns main={<div data-testid="main">Main</div>} side={<div data-testid="side">Side</div>} />,
    )
    const main = container.querySelector('[data-testid="main"]') as HTMLElement
    const side = container.querySelector('[data-testid="side"]') as HTMLElement

    await page.viewport(1300, 900)
    await expect.poll(() => Math.abs(main.getBoundingClientRect().width / side.getBoundingClientRect().width - 1.6)).toBeLessThanOrEqual(0.05)

    await rerender(<RecordColumns ratio="document" main={<div data-testid="main">Main</div>} side={<div data-testid="side">Side</div>} />)
    await expect
      .poll(() => Math.abs(main.getBoundingClientRect().width / side.getBoundingClientRect().width - 62 / 38))
      .toBeLessThanOrEqual(0.05)

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
