import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { BulkBar } from './BulkBar.tsx'
import { FilterBar } from './FilterBar.tsx'
import '../../styles/index.css'

describe('FilterBar', () => {
  it('keeps tools at the bar\'s right end from 64em, and lets bulk replace the whole bar', async () => {
    const { container, rerender } = await render(
      <FilterBar
        label="Filters"
        filters={<button type="button">Status</button>}
        tools={<button type="button" data-testid="tools">Columns</button>}
      />,
    )

    await page.viewport(1300, 800)
    const bar = container.querySelector('[role="search"]') as HTMLElement
    // Re-queried on each poll: tools only mounts once the resize reaches the matchMedia listener.
    await expect
      .poll(() => {
        const tools = container.querySelector('[data-testid="tools"]')
        return tools && Math.abs(tools.getBoundingClientRect().right - bar.getBoundingClientRect().right)
      })
      .toBeLessThanOrEqual(2)

    await rerender(
      <FilterBar
        label="Filters"
        filters={<button type="button">Status</button>}
        bulk={
          <BulkBar countText="2 selected" label="Bulk actions" clearLabel="Clear selection" onClear={() => {}}>
            <button type="button">Export</button>
          </BulkBar>
        }
      />,
    )
    await expect.poll(() => container.querySelector('[role="search"]')).toBeNull()
    await expect.poll(() => container.querySelector('[role="toolbar"]')).not.toBeNull()
  })
})
