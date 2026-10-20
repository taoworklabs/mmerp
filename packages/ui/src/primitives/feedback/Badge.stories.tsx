import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Badge } from './Badge.tsx'

const meta: Meta<typeof Badge> = {
  title: 'Feedback/Badge',
  component: Badge,
}

export default meta

type Story = StoryObj<typeof Badge>

export const Primary: Story = {
  args: { color: 'primary', children: 'Primary' },
}

export const Success: Story = {
  args: { color: 'success', children: 'Success' },
}

export const Warning: Story = {
  args: { color: 'warning', children: 'Warning' },
}

export const Danger: Story = {
  args: { color: 'danger', children: 'Danger' },
}

export const Info: Story = {
  args: { color: 'info', children: 'Info' },
}

export const Discovery: Story = {
  args: { color: 'discovery', children: 'Discovery' },
}

export const Gray: Story = {
  args: { color: 'gray', children: 'Gray' },
}

export const Filled: Story = {
  args: { variant: 'filled', children: 'Filled' },
}

export const Outline: Story = {
  args: { variant: 'outline', children: 'Outline' },
}

export const Long: Story = {
  args: { children: 'A'.repeat(300) },
  render: (args) => (
    <div style={{ width: 120 }}>
      <Badge {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const badge = within(canvasElement).getByText(/^A+$/)
    expect(getComputedStyle(badge).textOverflow).toBe('ellipsis')
    expect(badge.scrollWidth).toBeGreaterThan(badge.clientWidth)
  },
}
