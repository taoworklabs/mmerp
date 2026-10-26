import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Select } from '../../primitives/input/Select.tsx'
import { BulkBar } from './BulkBar.tsx'
import { FilterBar } from './FilterBar.tsx'
import { FilterChip } from './FilterChip.tsx'

const statusData = [
  { value: 'open', label: 'Đang mở' },
  { value: 'closed', label: 'Đã đóng' },
]

const kindData = [
  { value: 'invoice', label: 'Hóa đơn' },
  { value: 'quote', label: 'Báo giá' },
]

const meta: Meta<typeof FilterBar> = {
  title: 'Blocks/FilterBar',
  component: FilterBar,
  // React elements, not values to edit.
  argTypes: { bulk: { control: false }, filters: { control: false }, tools: { control: false } },
}

export default meta

type Story = StoryObj<typeof FilterBar>

export const Default: Story = {
  args: {
    label: 'Bộ lọc',
    filters: (
      <>
        <FilterBar.Search aria-label="Tìm kiếm" placeholder="Tìm theo số chứng từ, khách hàng" value="" onChange={() => {}} />
        <FilterChip label="Trạng thái" icon="●" data={statusData} value={null} onChange={() => {}} valueText={(l, v) => `${l}: ${v}`} clearLabel="Bỏ lọc trạng thái" nothingFound="Không có lựa chọn" />
        <Select aria-label="Loại chứng từ" data={kindData} value={null} onChange={() => {}} placeholder="Loại chứng từ" />
      </>
    ),
    tools: <Button variant="default">Cột hiển thị</Button>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const body = within(canvasElement.ownerDocument.body)

    // A filter reads like the tool buttons beside it, down to its primary edge once open — on the
    // control itself for a FilterChip, on its bordered box for a Select (whose own trigger is a
    // plain child button with no border of its own).
    const probe = document.createElement('div')
    probe.style.borderColor = 'var(--mmerp-color-primary-6)'
    canvasElement.appendChild(probe)
    const primaryEdge = getComputedStyle(probe).borderColor

    const chip = canvas.getByRole('combobox', { name: 'Trạng thái' })
    await userEvent.click(chip)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    // The border colour transitions, so the computed value only settles a beat after the attribute lands.
    await waitFor(() => expect(getComputedStyle(chip).borderColor).toBe(primaryEdge))
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByRole('listbox')).not.toBeInTheDocument())

    const select = canvas.getByRole('combobox', { name: 'Loại chứng từ' })
    const selectBox = select.closest('[data-mmerp-control]') as HTMLElement
    // Shrunk to its content, the select still keeps its chevron clear of the placeholder.
    const placeholderText = within(select).getByText('Loại chứng từ').getBoundingClientRect()
    const chevron = select.querySelector('svg')!.getBoundingClientRect()
    expect(chevron.left - placeholderText.right).toBeGreaterThanOrEqual(4)
    await userEvent.click(select)
    await waitFor(() => expect(body.getByRole('listbox')).toBeInTheDocument())
    await waitFor(() => expect(getComputedStyle(selectBox).borderColor).toBe(primaryEdge))

    probe.remove()
  },
}

export const WithBulk: Story = {
  args: {
    ...Default.args,
    bulk: (
      <BulkBar countText="Đã chọn 2" label="Thao tác hàng loạt" clearLabel="Bỏ chọn" onClear={() => {}}>
        <Button variant="default">Xuất file</Button>
      </BulkBar>
    ),
  },
}
