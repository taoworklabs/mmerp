import type { Meta, StoryObj } from '@storybook/react-vite'
import type * as React from 'react'
import { expect, userEvent, within } from 'storybook/test'
import { TextInput } from './TextInput.tsx'

const meta: Meta<typeof TextInput> = {
  title: 'Input/TextInput',
  component: TextInput,
}

export default meta

type Story = StoryObj<typeof TextInput>

export const Default: Story = {
  args: { label: 'Name' },
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Name' })
    await userEvent.type(input, 'abc')
    expect(input).toHaveValue('abc')
  },
}

export const Filled: Story = {
  args: { label: 'Name', defaultValue: 'Jane Doe' },
}

export const Placeholder: Story = {
  args: { label: 'Name', placeholder: 'Enter your name' },
}

export const WithDescription: Story = {
  args: { label: 'Name', description: 'As it appears on your ID' },
}

export const Required: Story = {
  args: { label: 'Name', required: true },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('textbox', { name: 'Name' })).toHaveAttribute('aria-required', 'true')
  },
}

export const Error: Story = {
  args: { label: 'Name', error: 'Name is required' },
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Name' })
    expect(input).toHaveAttribute('aria-invalid', 'true')
    const describedBy = input.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const errorText = canvasElement.querySelector(`#${describedBy}`)
    expect(errorText).toHaveTextContent('Name is required')
  },
}

export const Disabled: Story = {
  args: { label: 'Name', disabled: true, defaultValue: 'Jane Doe' },
}

export const ReadOnly: Story = {
  args: { label: 'Name', readOnly: true, defaultValue: 'Jane Doe' },
}

export const InModal: Story = {
  args: { label: 'Name' },
  render: (args) => (
    <div style={{ '--mmerp-control-height': 'var(--mmerp-control-height-lg)' } as React.CSSProperties}>
      <TextInput {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const input = within(canvasElement).getByRole('textbox', { name: 'Name' })
    // The border sits on the wrapper around the borderless input; that's the 40px control box.
    expect(input.parentElement?.getBoundingClientRect().height).toBe(40)
  },
}
