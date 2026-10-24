import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'
import { AppShell } from '../../primitives/layout/AppShell.tsx'
import { FilterBar } from '../list/FilterBar.tsx'
import { DataTable, type Column } from '../table/DataTable.tsx'
import { Page } from './Page.tsx'
import { PageHeader } from './PageHeader.tsx'

const meta: Meta<typeof Page> = {
  title: 'Blocks/Page',
  component: Page,
  // React elements, not values to edit.
  argTypes: { children: { control: false }, header: { control: false } },
}

export default meta

type Story = StoryObj<typeof Page>

export const Full: Story = {
  args: {
    header: <PageHeader title="Khách hàng" />,
    children: <div>Nội dung trang</div>,
  },
}

export const Centered: Story = {
  args: {
    variant: 'centered',
    header: <PageHeader title="Sửa hồ sơ nhân viên" />,
    children: <div>Nội dung biểu mẫu</div>,
  },
}

export const Detail: Story = {
  args: {
    variant: 'detail',
    header: <PageHeader title="Đơn bán hàng SO-0001" />,
    children: <div>Nội dung chi tiết</div>,
  },
}

function FillDemo() {
  return (
    <AppShell header={<div>Thanh đầu ứng dụng</div>}>
      <Page variant="fill" header={<PageHeader title="Đơn bán hàng" />}>
        <Page.Content data-testid="content">
          {Array.from({ length: 60 }, (_, index) => (
            <div key={index}>Dòng {index + 1}</div>
          ))}
          <div>Dòng cuối trang</div>
        </Page.Content>
      </Page>
    </AppShell>
  )
}

export const Fill: Story = {
  render: () => <FillDemo />,
  globals: { viewport: { value: 'short' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    const title = canvas.getByRole('heading', { name: 'Đơn bán hàng' })
    const topBefore = title.getBoundingClientRect().top

    const content = canvas.getByTestId('content') as HTMLElement
    content.scrollTop = content.scrollHeight

    await waitFor(() => expect(canvas.getByText('Dòng cuối trang').getBoundingClientRect().bottom).toBeLessThanOrEqual(500.5))
    expect(title.getBoundingClientRect().top).toBe(topBefore)
  },
}

type Row = { id: number; name: string; code: string; unit: string }

const listColumns: Column<Row>[] = [
  { key: 'name', header: 'Họ tên', role: 'title', render: (r) => r.name },
  { key: 'code', header: 'Mã', role: 'meta', kind: 'code', render: (r) => r.code },
  { key: 'unit', header: 'Phòng ban', role: 'meta', render: (r) => r.unit },
]

const listRows: Row[] = Array.from({ length: 40 }, (_, index) => ({
  id: index + 1,
  name: `Nhân viên ${index + 1}`,
  code: `NV${String(index + 1).padStart(4, '0')}`,
  unit: ['Phòng Kế toán', 'Phòng Nhân sự', 'Kho Bình Dương'][index % 3]!,
}))

function ListDemo() {
  return (
    <AppShell header={<div>Thanh đầu ứng dụng</div>}>
      <Page
        variant="list"
        header={<PageHeader title="Nhân viên" description="Hồ sơ nhân viên của công ty" />}
        filters={<FilterBar label="Bộ lọc nhân viên" filters={<FilterBar.Search aria-label="Tìm nhân viên" placeholder="Tìm theo tên, mã" value="" onChange={() => {}} />} />}
        footer={<div data-testid="footer">40 nhân viên</div>}
      >
        <DataTable
          columns={listColumns}
          rows={listRows}
          rowKey={(r) => r.id}
          label="Nhân viên"
          renderLink={(row, children) => <a href={`#/employees/${row.id}`}>{children}</a>}
          labels={{ actions: 'Thao tác khác', selectAll: 'Chọn mọi dòng', selectRow: 'Chọn dòng', resize: (c) => `Đổi độ rộng cột ${c}` }}
        />
      </Page>
    </AppShell>
  )
}

// A list page: the header and filters keep the page's margins with no rule of their own; the table
// runs edge to edge, its column-header row's bottom border being the separator, and only it scrolls,
// between the header and a footer pinned at the bottom.
export const List: Story = {
  render: () => <ListDemo />,
  globals: { viewport: { value: 'desktop' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const main = canvasElement.querySelector('main')!.getBoundingClientRect()
    const title = canvas.getByRole('heading', { name: 'Nhân viên' })
    const titleLeft = title.getBoundingClientRect().left
    const table = canvasElement.querySelector('[data-table]') as HTMLElement
    const th = canvas.getByRole('columnheader', { name: 'Họ tên' })
    const footer = canvas.getByTestId('footer')
    const footerBar = footer.parentElement!

    // No rule under the header: the column headers' border is the only separator.
    expect(getComputedStyle(title.closest('[data-page-header]')!).borderBottomWidth).toBe('0px')

    // The table runs edge to edge, unframed, its text lined up with the title.
    const tableRect = table.getBoundingClientRect()
    expect(tableRect.left).toBeCloseTo(main.left, 0)
    expect(tableRect.right).toBeCloseTo(main.right, 0)
    expect(getComputedStyle(table).borderLeftWidth).toBe('0px')
    expect(th.getBoundingClientRect().left + parseFloat(getComputedStyle(th).paddingLeft)).toBeCloseTo(titleLeft, 0)
    // The column headers sit on the page's own surface, part of the header block above them.
    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-body)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(th.closest('thead')!).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()

    // The footer is pinned to the bottom, its rule edge to edge, its content at the page margin.
    const footerRect = footerBar.getBoundingClientRect()
    expect(footerRect.bottom).toBeCloseTo(main.bottom, 0)
    expect(footerRect.left).toBeCloseTo(main.left, 0)
    expect(getComputedStyle(footerBar).borderTopWidth).toBe('1px')
    expect(footer.getBoundingClientRect().left).toBeCloseTo(titleLeft, 0)

    // Only the table scrolls; the header, filters and column headers stay put.
    const scroller = [table, ...table.querySelectorAll<HTMLElement>('*')].find((el) => el.scrollHeight > el.clientHeight + 1)!
    expect(scroller).toBeTruthy()
    const titleTop = title.getBoundingClientRect().top
    const thTop = th.getBoundingClientRect().top
    scroller.scrollTop = 400
    await waitFor(() => expect(scroller.scrollTop).toBeGreaterThan(0))
    expect(title.getBoundingClientRect().top).toBe(titleTop)
    expect(th.getBoundingClientRect().top).toBeCloseTo(thTop, 0)
    expect(canvasElement.querySelector('main')!.scrollTop).toBe(0)

    // The separator rides with the docked column headers: a collapsed table border would scroll
    // away with the rows, so every header cell draws its bottom rule itself.
    for (const cell of th.closest('tr')!.querySelectorAll('th')) {
      expect(getComputedStyle(cell).boxShadow).toMatch(/0px -1px 0px 0px inset/)
    }
  },
}

// Below 64em a list page scrolls as a whole: nothing is pinned, the table shows as cards.
export const ListBelowDrawerBreakpoint: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => <ListDemo />,
  globals: { viewport: { value: 'drawer' } },
  play: async ({ canvasElement }) => {
    const main = canvasElement.querySelector('main')!
    await waitFor(() => expect(main.scrollHeight).toBeGreaterThan(main.clientHeight))
    expect(within(canvasElement).getByTestId('footer').getBoundingClientRect().top).toBeGreaterThan(main.getBoundingClientRect().bottom)
    // Cards keep the page's margins: only a table runs edge to edge.
    const title = within(canvasElement).getByRole('heading', { name: 'Nhân viên' })
    const firstCard = within(canvasElement).getAllByRole('listitem')[0]!
    expect(firstCard.getBoundingClientRect().left).toBeCloseTo(title.getBoundingClientRect().left, 0)
  },
}
