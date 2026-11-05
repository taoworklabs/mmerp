import { render } from 'vitest-browser-react'
import { createRef } from 'react'
import { describe, expect, it } from 'vitest'
import { Checkbox } from './Checkbox.tsx'
import '../../styles/index.css'

describe('Checkbox', () => {
  it('submits its value with a native form when checked', async () => {
    const screen = await render(
      <form data-testid="form">
        <Checkbox name="agree" value="yes" label="Agree" data-testid="box" />
      </form>,
    )
    const form = screen.getByTestId('form').element() as HTMLFormElement
    expect(new FormData(form).get('agree')).toBeNull()
    await screen.getByRole('checkbox').click()
    expect(new FormData(form).get('agree')).toBe('yes')
    expect(screen.getByTestId('box').element().getAttribute('name')).toBeNull()
  })

  it('passes required and readOnly to the control', async () => {
    const screen = await render(<Checkbox name="agree" label="Agree" required readOnly />)
    const box = screen.getByRole('checkbox')
    await expect.element(box).toHaveAttribute('aria-required', 'true')
    await expect.element(box).toHaveAttribute('aria-readonly', 'true')
    const input = document.querySelector<HTMLInputElement>('input[name="agree"]')
    expect(input?.required).toBe(true)
  })

  it('routes id and ref to the control, so a form can focus an invalid box', async () => {
    const ref = createRef<HTMLSpanElement>()
    const screen = await render(<Checkbox id="agree" ref={ref} label="Agree" error="Required" />)
    const box = screen.getByRole('checkbox').element()
    expect(ref.current).toBe(box)
    // Base UI puts the id on its hidden input, which hands focus on to the box.
    document.getElementById('agree')?.focus()
    expect(document.activeElement).toBe(box)
  })
})
