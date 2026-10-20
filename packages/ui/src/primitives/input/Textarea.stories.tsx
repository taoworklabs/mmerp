import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Textarea } from './Textarea.tsx'

const meta: Meta<typeof Textarea> = {
  title: 'Input/Textarea',
  component: Textarea,
  args: { label: 'Notes' },
}

export default meta

type Story = StoryObj<typeof Textarea>

export const Default: Story = {}

export const Filled: Story = {
  args: { defaultValue: 'Some notes already written here.' },
}

export const Placeholder: Story = {
  args: { placeholder: 'Write your notes' },
}

export const WithDescription: Story = {
  args: { description: 'Visible only to reviewers' },
}

export const Required: Story = {
  args: { required: true },
}

export const Error: Story = {
  args: { error: 'Notes cannot be empty' },
}

export const Disabled: Story = {
  args: { disabled: true, defaultValue: 'Some notes already written here.' },
}

export const ReadOnly: Story = {
  args: { readOnly: true, defaultValue: 'Some notes already written here.' },
}

export const Autosize: Story = {
  args: { autosize: true, minRows: 2, maxRows: 4, defaultValue: 'Grows with its content.' },
  play: async ({ canvasElement }) => {
    const textarea = within(canvasElement).getByRole('textbox', { name: 'Notes' })
    expect(textarea).toHaveAttribute('data-autosize', '')
  },
}
