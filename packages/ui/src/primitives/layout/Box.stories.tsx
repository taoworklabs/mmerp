import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Box } from './Box.tsx'

const meta: Meta<typeof Box> = {
  title: 'Layout/Box',
  component: Box,
}

export default meta

type Story = StoryObj<typeof Box>

export const Default: Story = {
  args: { children: 'A box' },
}

export const Render: Story = {
  args: { render: <section />, children: 'Rendered as a section' },
  play: async ({ canvasElement }) => {
    const section = within(canvasElement).getByText('Rendered as a section')
    expect(section.tagName).toBe('SECTION')
  },
}
