import { render } from 'vitest-browser-react'
import { page, userEvent } from 'vitest/browser'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ActionIcon } from '../primitives/input/ActionIcon.tsx'
import { Button } from '../primitives/input/Button.tsx'
import { UnstyledButton } from '../primitives/input/UnstyledButton.tsx'

afterEach(() => vi.restoreAllMocks())

describe('a button rendered as a link', () => {
  const cases = [
    ['Button', <Button render={<a href="#x" />}>x</Button>],
    ['ActionIcon', <ActionIcon render={<a href="#x" />} aria-label="x">x</ActionIcon>],
    ['UnstyledButton', <UnstyledButton render={<a href="#x" />}>x</UnstyledButton>],
  ] as const
  for (const [name, el] of cases) {
    it(`${name} stays a plain anchor and logs nothing`, async () => {
      const error = vi.spyOn(console, 'error').mockImplementation(() => {})
      await render(el)
      const link = page.getByText('x').element().closest('a')!
      expect(link.getAttribute('href')).toBe('#x')
      expect(link.hasAttribute('type')).toBe(false)
      expect(link.hasAttribute('role')).toBe(false)
      expect(error).not.toHaveBeenCalled()
    })
  }

  const blocked = [
    ['disabled', { disabled: true }],
    ['loading', { loading: true }],
  ] as const
  for (const [state, props] of blocked) {
    it(`does not follow or fire onClick while ${state}`, async () => {
      const onClick = vi.fn()
      await render(<Button render={<a href="#blocked" />} onClick={onClick} {...props}>x</Button>)
      const link = page.getByText('x').element().closest('a')!
      const click = new MouseEvent('click', { bubbles: true, cancelable: true })
      link.dispatchEvent(click)
      expect(click.defaultPrevented).toBe(true)
      if (state === 'loading') {
        link.focus()
        await userEvent.keyboard('{Enter}')
      }
      expect(onClick).not.toHaveBeenCalled()
      expect(location.hash).not.toBe('#blocked')
    })
  }
})

describe('a loading button', () => {
  const cases = [
    ['Button', (loading: boolean, onClick: () => void) => <Button loading={loading} onClick={onClick}>Save</Button>],
    ['ActionIcon', (loading: boolean, onClick: () => void) => <ActionIcon loading={loading} onClick={onClick} aria-label="Save">s</ActionIcon>],
  ] as const
  for (const [name, el] of cases) {
    it(`${name} keeps focus, reports aria-disabled and ignores clicks`, async () => {
      const onClick = vi.fn()
      const { rerender } = await render(el(false, onClick))
      const button = page.getByRole('button', { name: 'Save' }).element() as HTMLButtonElement
      button.focus()
      await rerender(el(true, onClick))
      expect(document.activeElement).toBe(button)
      expect(button.disabled).toBe(false)
      expect(button.getAttribute('aria-disabled')).toBe('true')
      button.click()
      expect(onClick).not.toHaveBeenCalled()
    })
  }

  it('leaves a plain disabled Button natively disabled', async () => {
    await render(<Button disabled>Save</Button>)
    expect((page.getByRole('button', { name: 'Save' }).element() as HTMLButtonElement).disabled).toBe(true)
  })
})
