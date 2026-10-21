import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Pagination, type PaginationProps } from './Pagination.tsx'

const labels = { previous: 'Previous page', next: 'Next page', page: (n: number) => `Page ${n}` }

const meta: Meta<typeof Pagination> = {
  title: 'Navigation/Pagination',
  component: Pagination,
  args: { total: 20, labels, onChange: fn() },
}

export default meta

type Story = StoryObj<typeof Pagination>

function Controlled(props: PaginationProps) {
  const [value, setValue] = useState(props.value)
  return (
    <Pagination
      {...props}
      value={value}
      onChange={(page) => {
        setValue(page)
        props.onChange(page)
      }}
    />
  )
}

export const Default: Story = {
  args: { value: 1 },
  render: (args) => <Controlled {...args} />,
}

export const MiddleOfRange: Story = {
  args: { value: 5 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const pageButtons = canvas.getAllByRole('button').filter((button) => /^Page \d+$/.test(button.getAttribute('aria-label') ?? ''))
    expect(pageButtons.map((button) => button.textContent)).toEqual(['1', '4', '5', '6', '20'])

    const dots = canvas.getAllByText('…')
    expect(dots).toHaveLength(2)

    await userEvent.click(canvas.getByRole('button', { name: 'Page 6' }))
    expect(args.onChange).toHaveBeenCalledWith(6)
  },
}

export const AtLastPage: Story = {
  args: { value: 20 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Next page' })).toBeDisabled()
    expect(canvas.getByRole('button', { name: 'Previous page' })).not.toBeDisabled()
  },
}

export const AtFirstPage: Story = {
  args: { value: 1 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Previous page' })).toBeDisabled()
  },
}
