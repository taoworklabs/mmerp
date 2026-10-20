import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { PasswordInput } from './PasswordInput.tsx'

const meta: Meta<typeof PasswordInput> = {
  title: 'Input/PasswordInput',
  component: PasswordInput,
  args: { label: 'Password', visibilityLabels: { show: 'Show password', hide: 'Hide password' } },
}

export default meta

type Story = StoryObj<typeof PasswordInput>

export const Default: Story = {
  play: async ({ canvasElement }) => {
    const input = canvasElement.querySelector('input')
    expect(input).toHaveAttribute('type', 'password')
  },
}

export const Filled: Story = {
  args: { defaultValue: 'secret-value' },
}

export const Placeholder: Story = {
  args: { placeholder: 'Enter your password' },
}

export const WithDescription: Story = {
  args: { description: 'At least 8 characters' },
}

export const Required: Story = {
  args: { required: true },
}

export const Error: Story = {
  args: { error: 'Password is too short' },
}

export const Disabled: Story = {
  args: { disabled: true, defaultValue: 'secret-value' },
}

export const ReadOnly: Story = {
  args: { readOnly: true, defaultValue: 'secret-value' },
}

export const ToggleVisibility: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const toggle = canvas.getByRole('button', { name: 'Show password' })
    await userEvent.click(toggle)
    const input = canvasElement.querySelector('input')
    expect(input).toHaveAttribute('type', 'text')
    expect(canvas.getByRole('button', { name: 'Hide password' })).toBeInTheDocument()
  },
}
