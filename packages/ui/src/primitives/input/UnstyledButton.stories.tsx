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
  args: { children: 'Unstyled' },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByText('Unstyled')
    expect(getComputedStyle(button).borderStyle).toBe('none')
  },
}

export const Disabled: Story = {
  args: { disabled: true, children: 'Unstyled disabled' },
}
