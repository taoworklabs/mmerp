import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { Button } from '../primitives/input/Button.tsx'
import { ActionIcon } from '../primitives/input/ActionIcon.tsx'
import './index.css'

describe('touch target', () => {
  it('reaches 44x44 below the drawer breakpoint, and keeps its own height at desktop sizes', async () => {
    const { container } = await render(
      <>
        <Button>Default</Button>
        <ActionIcon aria-label="x">x</ActionIcon>
        <Button size="xs">Small</Button>
      </>,
    )
    const controls = Array.from(container.querySelectorAll<HTMLElement>('[data-mmerp-control]'))
    expect(controls).toHaveLength(3)

    await page.viewport(800, 600)
    for (const control of controls) {
      const rect = control.getBoundingClientRect()
      expect(rect.height).toBeGreaterThanOrEqual(44)
      expect(rect.width).toBeGreaterThanOrEqual(44)
    }

    await page.viewport(1280, 800)
    const [defaultButton, , xsButton] = controls
    expect(defaultButton?.getBoundingClientRect().height).toBe(30)
    expect(xsButton?.getBoundingClientRect().height).toBe(24)
  })
})
