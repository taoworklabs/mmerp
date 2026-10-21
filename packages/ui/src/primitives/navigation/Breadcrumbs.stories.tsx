import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Anchor } from '../typography/Anchor.tsx'
import { Breadcrumbs } from './Breadcrumbs.tsx'

const meta: Meta<typeof Breadcrumbs> = {
  title: 'Navigation/Breadcrumbs',
  component: Breadcrumbs,
  args: { 'aria-label': 'Breadcrumb' },
}

export default meta

type Story = StoryObj<typeof Breadcrumbs>

export const Default: Story = {
  render: (args) => (
    <Breadcrumbs {...args}>
      <Anchor href="#">Home</Anchor>
      <Anchor href="#">Customers</Anchor>
      <span>Nguyễn Văn A</span>
    </Breadcrumbs>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const nav = canvas.getByRole('navigation', { name: 'Breadcrumb' })
    expect(nav.querySelector('ol')).not.toBeNull()

    const current = canvas.getByText('Nguyễn Văn A').closest('li')
    expect(current).toHaveAttribute('aria-current', 'page')
    expect(canvas.getByText('Home').closest('li')).not.toHaveAttribute('aria-current')

    expect(canvas.getAllByText('/')).toHaveLength(2)
  },
}

export const CustomSeparator: Story = {
  render: (args) => (
    <Breadcrumbs {...args} separator=">">
      <Anchor href="#">Home</Anchor>
      <Anchor href="#">Customers</Anchor>
      <span>Nguyễn Văn A</span>
    </Breadcrumbs>
  ),
}
