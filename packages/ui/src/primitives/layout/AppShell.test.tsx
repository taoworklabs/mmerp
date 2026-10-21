import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { AppShell } from './AppShell.tsx'
import '../../styles/index.css'

describe('AppShell', () => {
  it('collapses the navbar to a rail at desktop widths, ignoring it below the drawer breakpoint', async () => {
    const { container } = await render(
      <AppShell header={<div>Header</div>} navbar={<div>Navbar</div>} navbarCollapsed navbarOpened>
        <div>Content</div>
      </AppShell>,
    )
    const navbar = container.querySelector('nav') as HTMLElement

    // The resize reaches the matchMedia listener and the resulting re-render asynchronously.
    await page.viewport(1300, 800)
    await expect.poll(() => navbar.getBoundingClientRect().width).toBe(56)

    // Below 64em the navbar is the drawer: the collapsed rail is ignored, drawn at full width instead.
    await page.viewport(800, 900)
    await expect.poll(() => navbar.getBoundingClientRect().width).toBe(240)
  })

  it('pins navbarFooter at the bottom of the navbar, hidden below the drawer breakpoint', async () => {
    const { container } = await render(
      <AppShell header={<div>Header</div>} navbar={<div>Navbar content</div>} navbarFooter={<button>Collapse</button>} navbarOpened>
        <div>Content</div>
      </AppShell>,
    )
    const footer = container.querySelector('button')!

    // display:none lands on the footer's own wrapping div, not the button itself, so a bounding
    // rect (collapsed by any hidden ancestor) is the reliable check, not the button's own display.
    await page.viewport(1300, 800)
    await expect.poll(() => footer.getBoundingClientRect().width).toBeGreaterThan(0)

    // Below 64em the rail never applies, so the rail-collapse toggle it pins is meaningless there.
    await page.viewport(800, 900)
    await expect.poll(() => footer.getBoundingClientRect().width).toBe(0)
  })

  it('lays main out as a full-height column, so a flex-filling child fills the window below the header', async () => {
    const { container } = await render(
      <AppShell header={<div>Header</div>}>
        <div data-testid="filler" style={{ flex: 1, minHeight: 0 }} />
      </AppShell>,
    )

    await page.viewport(800, 800)
    const filler = container.querySelector('[data-testid="filler"]') as HTMLElement
    await expect.poll(() => filler.getBoundingClientRect().height).toBe(800 - 48)
  })
})
