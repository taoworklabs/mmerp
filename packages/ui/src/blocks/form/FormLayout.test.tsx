import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { FormFields, FormSection } from './FormSection.tsx'
import '../../styles/index.css'

describe('FormSection', () => {
  it('splits title and fields two columns from 80em (normal half, Short a third, Row the whole), and stacks the title above the fields below 80em', async () => {
    const { container } = await render(
      <FormSection title="Section title">
        <div data-testid="normal">Normal field</div>
        <FormSection.Short>
          <div data-testid="short">Short field</div>
        </FormSection.Short>
        <FormSection.Row>
          <div data-testid="row">Row field</div>
        </FormSection.Row>
      </FormSection>,
    )
    const legend = container.querySelector('legend') as HTMLElement
    const normal = container.querySelector('[data-testid="normal"]') as HTMLElement
    const short = container.querySelector('[data-testid="short"]') as HTMLElement
    const row = container.querySelector('[data-testid="row"]') as HTMLElement
    const fields = normal.parentElement as HTMLElement

    await page.viewport(1400, 900)
    expect(legend.getBoundingClientRect().left).toBeLessThan(normal.getBoundingClientRect().left)

    // The grid's own column gap, not a hard-coded space token: it still holds if that token changes.
    const gap = parseFloat(getComputedStyle(fields).columnGap)
    const colWidth = (fields.getBoundingClientRect().width - 5 * gap) / 6
    const expectedNormal = colWidth * 3 + gap * 2
    const expectedShort = colWidth * 2 + gap
    const expectedRow = fields.getBoundingClientRect().width

    expect(normal.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedNormal - 2)
    expect(normal.getBoundingClientRect().width).toBeLessThanOrEqual(expectedNormal + 2)
    expect(short.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedShort - 2)
    expect(short.getBoundingClientRect().width).toBeLessThanOrEqual(expectedShort + 2)
    expect(row.getBoundingClientRect().width).toBeGreaterThanOrEqual(expectedRow - 2)
    expect(row.getBoundingClientRect().width).toBeLessThanOrEqual(expectedRow + 2)

    await page.viewport(1000, 900)
    await expect.poll(() => normal.getBoundingClientRect().top).toBeGreaterThanOrEqual(legend.getBoundingClientRect().bottom)
  })

  it('spans the whole row for a [data-full-row] wrapper, in a section grid and in FormFields', async () => {
    const { container } = await render(
      <>
        <FormSection title="Section title">
          <div>Normal field</div>
          <div data-full-row="" data-testid="section-row">Long text field</div>
        </FormSection>
        <FormFields>
          <div>Normal field</div>
          <div data-full-row="" data-testid="pair-row">Long text field</div>
        </FormFields>
      </>,
    )
    await page.viewport(1400, 900)
    for (const id of ['section-row', 'pair-row']) {
      const row = container.querySelector(`[data-testid="${id}"]`) as HTMLElement
      const grid = row.parentElement as HTMLElement
      await expect.poll(() => Math.round(row.getBoundingClientRect().width)).toBe(Math.round(grid.getBoundingClientRect().width))
    }
  })
})
