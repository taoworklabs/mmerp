import { render } from 'vitest-browser-react'
import { page, userEvent } from 'vitest/browser'
import { expect, it } from 'vitest'
import { Pagination } from './Pagination.tsx'
import '../../styles/index.css'

it('does not hover-style a disabled edge control', async () => {
  await render(<>
    <style>{'* { transition: none !important }'}</style>
    <Pagination total={3} value={1} onChange={() => {}} labels={{ previous: 'Previous', next: 'Next', page: n => `Page ${n}` }} />
  </>)
  const background = (name: string) => getComputedStyle(page.getByRole('button', { name }).element()).backgroundColor
  const idle = background('Previous')
  await userEvent.hover(page.getByRole('button', { name: 'Previous' }))
  expect(background('Previous')).toBe(idle)
  await userEvent.hover(page.getByRole('button', { name: 'Next' }))
  expect(background('Next')).not.toBe(idle)
})
