import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { TableOfContents } from './TableOfContents.tsx'

const items = [
  { id: 'intro', label: 'Thông tin chung', depth: 1 as const },
  { id: 'install', label: 'Hợp đồng lao động', depth: 2 as const },
  { id: 'usage', label: 'Lương và phụ cấp', depth: 2 as const },
  { id: 'detail', label: 'Phụ cấp ăn trưa', depth: 3 as const },
]

const meta: Meta<typeof TableOfContents> = {
  title: 'Navigation/TableOfContents',
  component: TableOfContents,
  args: { items, 'aria-label': 'Trên trang này' },
}

export default meta

type Story = StoryObj<typeof TableOfContents>

export const Default: Story = {
  args: { activeId: 'install' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const active = canvas.getByRole('link', { name: 'Hợp đồng lao động' })
    expect(active).toHaveAttribute('aria-current', 'location')
    expect(active).toHaveAttribute('href', '#install')

    const inactive = canvas.getByRole('link', { name: 'Thông tin chung' })
    expect(inactive).not.toHaveAttribute('aria-current')
  },
}
