import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { Button } from '../primitives/input/Button.tsx'
import { ActionIcon } from '../primitives/input/ActionIcon.tsx'
import { Combobox } from '../primitives/input/Combobox.tsx'
import { FileInput } from '../primitives/input/FileInput.tsx'
import { NumberInput } from '../primitives/input/NumberInput.tsx'
import { Select } from '../primitives/input/Select.tsx'
import { TextInput } from '../primitives/input/TextInput.tsx'
import './index.css'

const sampleData = [{ value: 'a', label: 'A' }]

describe('touch target', () => {
  it('reaches 44x44 below the drawer breakpoint, and keeps its own height at desktop sizes', async () => {
    const { container } = await render(
      <>
        <Button>Default</Button>
        <ActionIcon aria-label="x">x</ActionIcon>
        <Button size="xs">Small</Button>
        <Select label="Pick" data={sampleData} value={null} onChange={() => {}} style={{ width: 120 }} />
        <Combobox label="Search" data={sampleData} value={null} onChange={() => {}} nothingFound="None" style={{ width: 120 }} />
      </>,
    )
    const controls = Array.from(container.querySelectorAll<HTMLElement>('[data-mmerp-control]'))
    expect(controls).toHaveLength(5)

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

  it('grows the decorative box, not the inner control, below the drawer breakpoint', async () => {
    const { container } = await render(
      <>
        <TextInput label="Name" />
        <NumberInput label="Quantity" value={5} onChange={() => {}} />
        <FileInput label="Attachment" value={null} onChange={() => {}} clearLabel="Clear" />
        <Combobox label="Search" data={sampleData} value={null} onChange={() => {}} nothingFound="None" />
      </>,
    )
    const boxes = Array.from(container.querySelectorAll<HTMLElement>('[data-mmerp-control]'))
    expect(boxes).toHaveLength(4)

    await page.viewport(800, 600)
    for (const box of boxes) {
      const boxRect = box.getBoundingClientRect()
      expect(boxRect.height).toBeGreaterThanOrEqual(44)

      const inner = box.querySelector('input, button')
      expect(inner).not.toBeNull()
      const innerRect = inner!.getBoundingClientRect()
      // The regression this guards against: the inner control used to carry its own
      // min-height, growing past the bordered box drawn around it.
      expect(innerRect.top).toBeGreaterThanOrEqual(boxRect.top - 0.5)
      expect(innerRect.bottom).toBeLessThanOrEqual(boxRect.bottom + 0.5)
    }
  })
})
