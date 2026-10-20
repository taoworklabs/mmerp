import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { FileInput, type FileInputProps } from './FileInput.tsx'

const meta: Meta<typeof FileInput> = {
  title: 'Input/FileInput',
  component: FileInput,
  args: { label: 'Attachment', clearLabel: 'Clear', onChange: fn() },
}

export default meta

type Story = StoryObj<typeof FileInput>

function Controlled(props: FileInputProps) {
  const [value, setValue] = useState(props.value)
  return (
    <FileInput
      {...props}
      value={value}
      onChange={(file) => {
        setValue(file)
        props.onChange(file)
      }}
    />
  )
}

export const Default: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
}

export const Placeholder: Story = {
  args: { value: null, placeholder: 'No file selected' },
  render: (args) => <Controlled {...args} />,
}

export const WithDescription: Story = {
  args: { value: null, description: 'PDF up to 10 MB' },
  render: (args) => <Controlled {...args} />,
}

export const Required: Story = {
  args: { value: null, required: true },
  render: (args) => <Controlled {...args} />,
}

export const Error: Story = {
  args: { value: null, error: 'A file is required' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const trigger = within(canvasElement).getByRole('button', { name: 'Attachment' })
    expect(trigger).toHaveAttribute('aria-invalid', 'true')
    const describedBy = trigger.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const errorText = canvasElement.querySelector(`#${describedBy}`)
    expect(errorText).toHaveTextContent('A file is required')
  },
}

export const WithAriaLabel: Story = {
  args: { value: null, label: undefined, 'aria-label': 'Attachment file' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByRole('button', { name: 'Attachment file' })).toBeInTheDocument()
  },
}

export const Disabled: Story = {
  args: { value: null, disabled: true },
  render: (args) => <Controlled {...args} />,
}

export const ReadOnly: Story = {
  args: { value: null, readOnly: true },
  render: (args) => <Controlled {...args} />,
}

export const Filled: Story = {
  args: { value: new File(['content'], 'report.pdf', { type: 'application/pdf' }) },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(within(canvasElement).getByText('report.pdf')).toBeInTheDocument()
  },
}

export const UploadAndClear: Story = {
  args: { value: null },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const file = new File(['content'], 'photo.png', { type: 'image/png' })
    const input = canvasElement.querySelector<HTMLInputElement>('input[type="file"]')
    if (!input) throw new TypeError('file input not found')
    await userEvent.upload(input, file)
    expect(within(canvasElement).getByText('photo.png')).toBeInTheDocument()

    const clear = within(canvasElement).getByRole('button', { name: 'Clear' })
    await userEvent.click(clear)
    expect(within(canvasElement).queryByText('photo.png')).not.toBeInTheDocument()
  },
}
