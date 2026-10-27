import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { NavLink } from './NavLink.tsx'

const meta: Meta<typeof NavLink> = {
  title: 'Navigation/NavLink',
  component: NavLink,
  // React elements, not values to edit.
  argTypes: { children: { control: false } },
}

export default meta

type Story = StoryObj<typeof NavLink>

export const Default: Story = {
  args: { label: 'Tổng quan', href: '#' },
}

export const Active: Story = {
  args: { label: 'Tổng quan', href: '#', active: true },
  play: async ({ canvasElement }) => {
    const link = within(canvasElement).getByRole('link', { name: 'Tổng quan' })
    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-primary-light)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(link).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()
  },
}

export const WithSections: Story = {
  args: { label: 'Cài đặt', description: 'Tùy chọn của doanh nghiệp', href: '#', leftSection: '⚙', rightSection: '3' },
}

export const Long: Story = {
  args: { label: 'Báo cáo tổng hợp công nợ phải thu theo khách hàng và nhân viên kinh doanh', href: '#' },
  render: (args) => (
    <div style={{ width: 200 }}>
      <NavLink {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const link = within(canvasElement).getByRole('link')
    const label = within(link).getByText(/Báo cáo tổng hợp/)
    expect(label.scrollWidth).toBeGreaterThan(label.clientWidth)
    expect(link.getBoundingClientRect().width).toBe(200)
  },
}

export const LongDescription: Story = {
  args: { label: 'Chờ duyệt', description: 'Bảng công BC-2026-000012 của Nguyễn Văn An, phòng Kinh doanh (HCM)', href: '#' },
  render: (args) => (
    <div style={{ width: 240 }}>
      <NavLink {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const description = within(canvasElement).getByText(/Bảng công/)
    // Wrapped, not cut: every word stays readable on a narrow row.
    expect(description.scrollWidth).toBeLessThanOrEqual(description.clientWidth)
    expect(description.getBoundingClientRect().height).toBeGreaterThan(parseFloat(getComputedStyle(description).lineHeight) * 1.5)
  },
}

export const Compact: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <NavLink label="Báo giá" href="#inactive" leftSection="⚙" compact />
      <NavLink label="Đơn bán hàng" href="#active" leftSection="⚙" active compact />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const [inactive, active] = canvas.getAllByRole('link')

    expect(inactive!.getBoundingClientRect().height).toBe(32)
    expect(active!.getBoundingClientRect().height).toBe(32)

    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    const dimmed = getComputedStyle(probe).color
    probe.remove()

    expect(getComputedStyle(inactive!).color).toBe(dimmed)
    expect(getComputedStyle(active!).color).not.toBe(dimmed)
  },
}

export const Nested: Story = {
  args: {
    label: 'Hàng hóa',
    children: (
      <>
        <NavLink label="Mặt hàng" href="#items" />
        <NavLink label="Nhóm hàng" href="#categories" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.queryByRole('link', { name: 'Mặt hàng' })).not.toBeInTheDocument()

    await userEvent.click(canvas.getByRole('button', { name: 'Hàng hóa' }))
    expect(canvas.getByRole('link', { name: 'Mặt hàng' })).toBeVisible()
    expect(canvas.getByRole('link', { name: 'Nhóm hàng' })).toBeVisible()
  },
}

export const GroupLabel: Story = {
  render: () => (
    <div>
      <NavLink.GroupLabel>Hồ sơ</NavLink.GroupLabel>
      <NavLink label="Nhân viên" href="#" />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const label = within(canvasElement).getByText('Hồ sơ')
    // Indented like the links below it, not flush with the nav's own edge.
    expect(getComputedStyle(label).paddingInlineStart).toBe('8px')

    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(label).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}
