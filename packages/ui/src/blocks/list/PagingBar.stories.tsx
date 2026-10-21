import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { PagingBar, type PagingBarProps } from './PagingBar.tsx'

const labels = { previous: 'Previous page', next: 'Next page', page: (n: number) => `Page ${n}` }

const meta: Meta<typeof PagingBar> = {
  title: 'Blocks/PagingBar',
  component: PagingBar,
  args: {
    total: 128,
    page: 5,
    pageSize: 10,
    pageSizes: [10, 20, 50],
    totalText: '128 total',
    pageSizeText: (n: number) => `${n} / page`,
    pageSizeLabel: 'Rows per page',
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

    await userEvent.click(canvas.getByRole('combobox', { name: 'Rows per page' }))
    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(await body.findByRole('option', { name: '20 / page' }))
    expect(args.onPageSize).toHaveBeenCalledWith(20)
  },
}

export const Empty: Story = {
  args: { total: 0 },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    expect(canvasElement.firstElementChild).toBeNull()
  },
}
