import { Input } from '@base-ui/react/input'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Field } from './Field.tsx'

const meta: Meta<typeof Field> = {
  title: 'Input/Field',
  component: Field,
  args: { children: <Input type="text" /> },
}

export default meta

type Story = StoryObj<typeof Field>

export const Default: Story = {
  args: { label: 'Name' },
}

export const WithDescription: Story = {
  args: { label: 'Name', description: 'As it appears on your ID' },
}

export const Required: Story = {
  args: { label: 'Name', required: true },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('*')).toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { label: 'Name', error: 'Name is required' },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('Name is required')).toBeInTheDocument()
  },
}

export const Disabled: Story = {
  args: { label: 'Name', disabled: true },
}

export const ReadOnly: Story = {
  args: { label: 'Name', readOnly: true },
}
