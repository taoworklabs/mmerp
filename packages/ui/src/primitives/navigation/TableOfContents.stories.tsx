import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { TableOfContents } from './TableOfContents.tsx'

const items = [
  { id: 'intro', label: 'Introduction', depth: 1 as const },
  { id: 'install', label: 'Installation', depth: 2 as const },
  { id: 'usage', label: 'Usage', depth: 2 as const },
  { id: 'detail', label: 'Detail', depth: 3 as const },
]

const meta: Meta<typeof TableOfContents> = {
  title: 'Navigation/TableOfContents',
  component: TableOfContents,
  args: { items, 'aria-label': 'On this page' },
}

export default meta

type Story = StoryObj<typeof TableOfContents>

export const Default: Story = {
  args: { activeId: 'install' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const active = canvas.getByRole('link', { name: 'Installation' })
    expect(active).toHaveAttribute('aria-current', 'location')
    expect(active).toHaveAttribute('href', '#install')

    const inactive = canvas.getByRole('link', { name: 'Introduction' })
    expect(inactive).not.toHaveAttribute('aria-current')
  },
}
