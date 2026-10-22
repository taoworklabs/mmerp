import type { Meta, StoryObj } from '@storybook/react-vite'
import { Center } from './Center.tsx'

const meta: Meta<typeof Center> = {
  title: 'Layout/Center',
  component: Center,
}

export default meta

type Story = StoryObj<typeof Center>

export const Default: Story = {
  args: { children: 'Centred', style: { height: 120, border: '1px dashed currentColor' } },
}

export const Inline: Story = {
  args: { inline: true, children: 'Inline centred' },
}
