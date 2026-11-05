import { render } from 'vitest-browser-react'
import { describe, expect, it } from 'vitest'
import { Checkbox } from './Checkbox.tsx'
import '../../styles/index.css'

describe('Checkbox', () => {
  it('submits its value with a native form when checked', async () => {
    const screen = await render(
      <form data-testid="form">
        <Checkbox name="agree" value="yes" label="Agree" data-testid="root" />
      </form>,
    )
    const form = screen.getByTestId('form').element() as HTMLFormElement
    expect(new FormData(form).get('agree')).toBeNull()
    await screen.getByRole('checkbox').click()
    expect(new FormData(form).get('agree')).toBe('yes')
    expect(screen.getByTestId('root').element().getAttribute('name')).toBeNull()
  })

  it('passes required and readOnly to the control', async () => {
    const screen = await render(<Checkbox name="agree" label="Agree" required readOnly />)
    const box = screen.getByRole('checkbox')
    await expect.element(box).toHaveAttribute('aria-required', 'true')
    await expect.element(box).toHaveAttribute('aria-readonly', 'true')
    const input = document.querySelector<HTMLInputElement>('input[name="agree"]')
    expect(input?.required).toBe(true)
  })
})
