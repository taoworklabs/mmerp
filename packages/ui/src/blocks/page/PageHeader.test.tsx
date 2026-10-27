import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { PageHeader } from './PageHeader.tsx'
import '../../styles/index.css'

describe('PageHeader', () => {
  it('stacks actions under the title below 48em, and rows them with the title from 48em', async () => {
    const { container } = await render(<PageHeader title="Orders" actions={<button type="button">New order</button>} />)
    const title = container.querySelector('h1') as HTMLElement
    const actions = container.querySelector('button') as HTMLElement

    await page.viewport(600, 800)
    await expect.poll(() => actions.getBoundingClientRect().top).toBeGreaterThanOrEqual(title.getBoundingClientRect().bottom)

    await page.viewport(1300, 800)
    await expect.poll(() => Math.abs(actions.getBoundingClientRect().top - title.getBoundingClientRect().top)).toBeLessThanOrEqual(4)
  })

  it('makes the action as wide as the header below 48em, and leaves it its own width from 48em', async () => {
    const { container } = await render(<PageHeader title="Orders" actions={<button type="button">New order</button>} />)
    const header = container.querySelector('[data-page-header]') as HTMLElement
    const action = container.querySelector('button') as HTMLElement

    await page.viewport(375, 800)
    await expect.poll(() => Math.round(action.getBoundingClientRect().width)).toBe(Math.round(header.getBoundingClientRect().width))

    await page.viewport(1300, 800)
    await expect.poll(() => action.getBoundingClientRect().width).toBeLessThan(header.getBoundingClientRect().width / 2)
  })
})
