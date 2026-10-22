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
  args: { label: 'Họ tên' },
}

export const WithDescription: Story = {
  args: { label: 'Họ tên', description: 'Ghi đúng như trên căn cước công dân' },
}

export const Required: Story = {
  args: { label: 'Họ tên', required: true },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('*')).toBeInTheDocument()
  },
}

export const Error: Story = {
  args: { label: 'Họ tên', error: 'Vui lòng nhập họ tên' },
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('Vui lòng nhập họ tên')).toBeInTheDocument()
  },
}

export const Disabled: Story = {
  args: { label: 'Họ tên', disabled: true },
}

export const ReadOnly: Story = {
  args: { label: 'Họ tên', readOnly: true },
}
