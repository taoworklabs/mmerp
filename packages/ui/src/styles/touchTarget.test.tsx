import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { describe, expect, it } from 'vitest'
import { Calendar } from '../primitives/dates/Calendar.tsx'
import { DateInput } from '../primitives/dates/DateInput.tsx'
import { MonthPicker } from '../primitives/dates/MonthPicker.tsx'
import { MonthPickerInput } from '../primitives/dates/MonthPickerInput.tsx'
import { Button } from '../primitives/input/Button.tsx'
import { ActionIcon } from '../primitives/input/ActionIcon.tsx'
import { Combobox } from '../primitives/input/Combobox.tsx'
import { FileInput } from '../primitives/input/FileInput.tsx'
import { NumberInput } from '../primitives/input/NumberInput.tsx'
import { Select } from '../primitives/input/Select.tsx'
import { TextInput } from '../primitives/input/TextInput.tsx'
import './index.css'

const sampleData = [{ value: 'a', label: 'A' }]
const sampleLocale = { weekdays: ['S', 'M', 'T', 'W', 'T', 'F', 'S'], months: Array.from({ length: 12 }, (_, i) => `M${i + 1}`) }

describe('touch target', () => {
  it('reaches 44x44 below the drawer breakpoint, and keeps its own height at desktop sizes', async () => {
    const { container } = await render(
      <>
        <Button>Default</Button>
        <ActionIcon aria-label="x">x</ActionIcon>
        <Button size="xs">Small</Button>
        <Select label="Pick" data={sampleData} value="a" onChange={() => {}} clearable clearLabel="Clear" />
        <Combobox label="Search" data={sampleData} value={null} onChange={() => {}} nothingFound="None" style={{ width: 120 }} />
      </>,
    )
    const controls = Array.from(container.querySelectorAll<HTMLElement>('[data-mmerp-control]'))
    // Select's own box, plus its clear button (an ActionIcon, a control in its own right).
    expect(controls).toHaveLength(6)

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
        <DateInput
          label="Date"
          value={null}
          onChange={() => {}}
          parse={() => null}
          format={(iso) => iso}
          locale={sampleLocale}
          labels={{ previousMonth: 'Previous', nextMonth: 'Next' }}
        />
        <MonthPickerInput
          label="Month"
          value={null}
          onChange={() => {}}
          format={(month) => month}
          locale={sampleLocale}
          labels={{ previousYear: 'Previous', nextYear: 'Next' }}
        />
      </>,
    )
    const boxes = Array.from(container.querySelectorAll<HTMLElement>('[data-mmerp-control]'))
    expect(boxes).toHaveLength(6)

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

  it('grows calendar day and month cells to the touch target below the drawer breakpoint', async () => {
    const { container } = await render(
      <>
        <Calendar value={null} onChange={() => {}} today="2026-10-20" locale={sampleLocale} labels={{ previousMonth: 'Previous', nextMonth: 'Next' }} />
        <MonthPicker value="2026-10" onChange={() => {}} locale={sampleLocale} labels={{ previousYear: 'Previous', nextYear: 'Next' }} />
      </>,
    )
    const [day, month] = Array.from(container.querySelectorAll<HTMLElement>('[role="grid"]')).map((grid) => grid.querySelector<HTMLElement>('button[role="gridcell"]')!)

    await page.viewport(800, 600)
    for (const cell of [day!, month!]) {
      const rect = cell.getBoundingClientRect()
      expect(rect.height).toBeGreaterThanOrEqual(44)
      expect(rect.width).toBeGreaterThanOrEqual(44)
    }

    await page.viewport(1280, 800)
    expect(day!.getBoundingClientRect().height).toBe(32)
    expect(day!.getBoundingClientRect().width).toBe(32)
    expect(month!.getBoundingClientRect().height).toBe(32)
  })
})
