import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { UnstyledButton } from './UnstyledButton.tsx'

const meta: Meta<typeof UnstyledButton> = {
  title: 'Input/UnstyledButton',
  component: UnstyledButton,
}

export default meta

type Story = StoryObj<typeof UnstyledButton>

export const Default: Story = {
  args: { children: 'Xem chi tiết' },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByText('Xem chi tiết')
    expect(getComputedStyle(button).borderStyle).toBe('none')
  },
}

export const Disabled: Story = {
  args: { disabled: true, children: 'Xem chi tiết (đã khóa)' },
}
