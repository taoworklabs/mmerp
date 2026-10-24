import type { Meta, StoryObj } from '@storybook/react-vite'
import { Paper } from './Paper.tsx'

const meta: Meta<typeof Paper> = {
  title: 'Layout/Paper',
  component: Paper,
}

export default meta

type Story = StoryObj<typeof Paper>

export const Default: Story = {
  args: { children: 'Bề mặt' },
}

export const WithBorder: Story = {
  args: { withBorder: true, children: 'Bề mặt có viền' },
}

export const Padding: Story = {
  args: { padding: 'lg', children: 'Bề mặt có đệm' },
}

export const ShadowXs: Story = {
  args: { shadow: 'xs', children: 'Bề mặt nổi' },
}

export const ShadowSm: Story = {
  args: { shadow: 'sm', children: 'Bề mặt nổi hơn' },
}
