import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { PagingBar, type PagingBarProps } from './PagingBar.tsx'

const labels = { previous: 'Trang trước', next: 'Trang sau', page: (n: number) => `Trang ${n}` }

const meta: Meta<typeof PagingBar> = {
  title: 'Blocks/PagingBar',
  component: PagingBar,
  args: {
    total: 128,
    page: 5,
    pageSize: 10,
    pageSizes: [10, 20, 50],
    totalText: 'Tổng 128 bản ghi',
    pageSizeText: (n: number) => `${n} / trang`,
    pageSizeLabel: 'Số dòng mỗi trang',
    labels,
    onPage: fn(),
    onPageSize: fn(),
  },
}

export default meta

type Story = StoryObj<typeof PagingBar>

function Controlled(props: PagingBarProps) {
  const [page, setPage] = useState(props.page)
  const [pageSize, setPageSize] = useState(props.pageSize)
  return (
    <PagingBar
      {...props}
      page={page}
      pageSize={pageSize}
      onPage={(p) => {
        setPage(p)
        props.onPage(p)
      }}
      onPageSize={(size) => {
        setPageSize(size)
        props.onPageSize(size)
      }}
    />
  )
}

export const Default: Story = {
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const bar = canvasElement.firstElementChild as HTMLElement
    const nav = canvas.getByRole('navigation')
    const barRect = bar.getBoundingClientRect()
    const navRect = nav.getBoundingClientRect()
    const leftGap = navRect.left - barRect.left
    const rightGap = barRect.right - navRect.right
    expect(Math.abs(leftGap - rightGap)).toBeLessThanOrEqual(2)

    await userEvent.click(canvas.getByRole('combobox', { name: 'Số dòng mỗi trang' }))
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(await body.findByRole('option', { name: '20 / trang' }))
    expect(args.onPageSize).toHaveBeenCalledWith(20)
  },
}

// Too narrow for three columns: the page buttons move to their own row and nothing overflows.
export const Narrow: Story = {
  render: (args) => <Controlled {...args} />,
  globals: { viewport: { value: 'mobile' } },
  play: async ({ canvasElement }) => {
    const bar = canvasElement.firstElementChild as HTMLElement
    const nav = within(canvasElement).getByRole('navigation')
    const select = within(canvasElement).getByRole('combobox', { name: 'Số dòng mỗi trang' })
    expect(bar.scrollWidth).toBeLessThanOrEqual(bar.clientWidth)
    expect(select.getBoundingClientRect().right).toBeLessThanOrEqual(bar.getBoundingClientRect().right + 0.5)
    expect(nav.getBoundingClientRect().bottom).toBeLessThanOrEqual(select.getBoundingClientRect().top + 0.5)
  },
}

// No rows, no paging: the bar renders nothing and leaves the space to the table's empty state.
export const Empty: Story = {
  tags: ['!dev', '!autodocs'],
  args: { total: 0 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(canvasElement.firstElementChild).toBeNull()
  },
}

export const NavigationLabel: Story = {
  args: { navigationLabel: 'Phân trang nhân viên' },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('navigation', { name: 'Phân trang nhân viên' })).toBeInTheDocument()
  },
}
