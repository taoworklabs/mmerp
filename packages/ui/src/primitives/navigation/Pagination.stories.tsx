import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Pagination, type PaginationProps } from './Pagination.tsx'

const labels = { previous: 'Trang trước', next: 'Trang sau', page: (n: number) => `Trang ${n}` }

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
  globals: { viewport: { value: 'desktop' } },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const pageButtons = canvas.getAllByRole('button').filter((button) => /^Trang \d+$/.test(button.getAttribute('aria-label') ?? ''))
    expect(pageButtons.map((button) => button.textContent)).toEqual(['1', '4', '5', '6', '20'])

    const dots = canvas.getAllByText('…')
    expect(dots).toHaveLength(2)

    await userEvent.click(canvas.getByRole('button', { name: 'Trang 6' }))
    expect(args.onChange).toHaveBeenCalledWith(6)
  },
}

export const AtLastPage: Story = {
  args: { value: 20 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Trang sau' })).toBeDisabled()
    expect(canvas.getByRole('button', { name: 'Trang trước' })).not.toBeDisabled()
  },
}

export const AtFirstPage: Story = {
  args: { value: 1 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('button', { name: 'Trang trước' })).toBeDisabled()
  },
}

// On a phone only the active page sits between the ends, so 44px square buttons still fit the row.
export const Narrow: Story = {
  args: { value: 5 },
  globals: { viewport: { value: 'mobile' } },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const pageButtons = canvas.getAllByRole('button').filter((button) => /^Trang \d+$/.test(button.getAttribute('aria-label') ?? ''))
    expect(pageButtons.map((button) => button.textContent)).toEqual(['1', '5', '20'])
    for (const button of canvas.getAllByRole('button')) {
      const rect = button.getBoundingClientRect()
      expect(rect.width).toBeGreaterThanOrEqual(44)
      expect(rect.height).toBeGreaterThanOrEqual(44)
    }
    const list = canvasElement.querySelector('ul')!
    expect(list.getBoundingClientRect().right).toBeLessThanOrEqual(canvasElement.ownerDocument.documentElement.clientWidth)
  },
}
