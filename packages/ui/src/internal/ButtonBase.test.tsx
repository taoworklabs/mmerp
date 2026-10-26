import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
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
})
