import { render } from 'vitest-browser-react'
import { page, userEvent } from 'vitest/browser'
import { describe, expect, it, vi } from 'vitest'
import { Checkbox } from '../primitives/input/Checkbox.tsx'
import { FileInput } from '../primitives/input/FileInput.tsx'
import { TextInput } from '../primitives/input/TextInput.tsx'
import { NumberInput } from '../primitives/input/NumberInput.tsx'
import { MultiSelect } from '../primitives/input/MultiSelect.tsx'
import { FormFields } from '../blocks/form/FormSection.tsx'
import { Group } from '../primitives/layout/Group.tsx'
import { Button } from '../primitives/input/Button.tsx'
import { Dialog } from '../primitives/overlay/Dialog.tsx'
import './index.css'

describe('production interactions', () => {
  it('keeps spacing between wrapped rows of actions', async () => {
    await page.viewport(375, 740)
    await render(<Group gap="xs" style={{ width: 100 }}><Button fullWidth>First</Button><Button fullWidth>Second</Button></Group>)
    const first = page.getByRole('button', { name: 'First' }).element().getBoundingClientRect()
    const second = page.getByRole('button', { name: 'Second' }).element().getBoundingClientRect()
    expect(second.top - first.bottom).toBe(8)
  })

  it('aligns multi-select with form fields and contains long selected labels on a phone', async () => {
    const onChange = vi.fn()
    const fields = (label: string) => (
      <FormFields>
        <TextInput label="Name" />
        <MultiSelect label="People" data={[{ value: 'a', label }]} value={['a']} onChange={onChange} nothingFound="None" removeLabel={() => 'Remove'} />
      </FormFields>
    )
    const { container, rerender } = await render(fields('An'))
    await page.viewport(1280, 800)
    const controls = container.querySelectorAll('[data-mmerp-control]')
    expect(controls[0]!.getBoundingClientRect().height).toBe(40)
    expect(controls[1]!.getBoundingClientRect().height).toBe(40)
    await page.viewport(375, 740)
    await rerender(fields('A very long selection '.repeat(15)))
    expect(document.documentElement.scrollWidth).toBeLessThanOrEqual(375)
    await userEvent.click(page.getByRole('button', { name: 'Remove' }))
    expect(onChange.mock.calls[0]?.[0]).toEqual([])
  })

  it('gives text and number fields a visible keyboard focus ring, including errors', async () => {
    await render(<><TextInput label="Name" error="Required" /><NumberInput label="Quantity" value={1} onChange={() => {}} /></>)
    for (const name of ['Name', 'Quantity']) {
      const input = page.getByRole('textbox', { name }).element() as HTMLElement
      input.focus()
      const wrapper = input.closest('[data-mmerp-control]')!
      expect(getComputedStyle(wrapper).outlineStyle).toBe('solid')
      expect(getComputedStyle(wrapper).outlineWidth).toBe('2px')
    }
  })

  it('gives a checkbox a usable label target on desktop and phone', async () => {
    await render(<Checkbox label="Accept terms" />)
    const label = page.getByRole('checkbox').element().closest('label')!
    await page.viewport(1280, 800)
    expect(label.getBoundingClientRect().height).toBeGreaterThanOrEqual(30)
    await page.viewport(375, 740)
    expect(label.getBoundingClientRect().height).toBeGreaterThanOrEqual(44)
  })

  it('names a standalone checkbox on the control itself', async () => {
    await render(<Checkbox aria-label="Select row" />)
    await expect.element(page.getByRole('checkbox', { name: 'Select row' })).toBeVisible()
  })

  it('keeps a file field named after its label when a file or placeholder is visible', async () => {
    await render(<FileInput label="Attachment" placeholder="Choose a file" value={null} onChange={() => {}} clearLabel="Clear" />)
    await expect.element(page.getByRole('button', { name: 'Attachment Choose a file' })).toBeVisible()
    await userEvent.click(page.getByText('Attachment'))
    await expect.element(page.getByRole('button', { name: 'Attachment Choose a file' })).toHaveFocus()
  })

  it('preserves caller descriptions alongside file field guidance', async () => {
    await render(<><p id="upload-help">Maximum 10 MB</p><FileInput label="Attachment" description="PDF only" aria-describedby="upload-help" value={null} onChange={() => {}} clearLabel="Clear" /></>)
    const control = page.getByRole('button', { name: 'Attachment' }).element()
    const descriptions = control.getAttribute('aria-describedby')!.split(' ').map(id => document.getElementById(id)?.textContent).join(' ')
    expect(descriptions).toContain('Maximum 10 MB')
    expect(descriptions).toContain('PDF only')
  })

  it('fills the phone viewport with a dialog while keeping its close button available', async () => {
    await page.viewport(375, 740)
    await render(<Dialog opened onClose={vi.fn()} title="Edit record" closeLabel="Close">{Array.from({ length: 30 }, (_, i) => <p key={i}>Field {i}</p>)}</Dialog>)
    const dialog = page.getByRole('dialog').element()
    const bounds = dialog.getBoundingClientRect()
    expect(bounds.width).toBe(375)
    expect(bounds.height).toBe(740)
    expect(bounds.x).toBe(0)
    expect(bounds.y).toBe(0)
    await expect.element(page.getByRole('button', { name: 'Close' })).toBeVisible()
  })
})
