import type { Meta, StoryObj } from '@storybook/react-vite'
import { Divider } from './Divider.tsx'

const meta: Meta<typeof Divider> = {
  title: 'Layout/Divider',
  component: Divider,
}

export default meta

type Story = StoryObj<typeof Divider>

export const Default: Story = {}

export const Vertical: Story = {
  args: { orientation: 'vertical' },
  render: (args) => (
    <div style={{ display: 'flex', height: 40 }}>
      <Divider {...args} />
    </div>
  ),
}
