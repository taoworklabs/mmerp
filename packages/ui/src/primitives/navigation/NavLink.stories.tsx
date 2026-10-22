import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { NavLink } from './NavLink.tsx'

const meta: Meta<typeof NavLink> = {
  title: 'Navigation/NavLink',
  component: NavLink,
}

export default meta

type Story = StoryObj<typeof NavLink>

export const Default: Story = {
  args: { label: 'Dashboard', href: '#' },
}

export const Active: Story = {
  args: { label: 'Dashboard', href: '#', active: true },
  play: async ({ canvasElement }) => {
    const link = within(canvasElement).getByRole('link', { name: 'Dashboard' })
    const probe = document.createElement('div')
    probe.style.backgroundColor = 'var(--mmerp-color-primary-light)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(link).backgroundColor).toBe(getComputedStyle(probe).backgroundColor)
    probe.remove()
  },
}

export const WithSections: Story = {
  args: { label: 'Settings', description: 'Workspace preferences', href: '#', leftSection: '⚙', rightSection: '3' },
}

export const Long: Story = {
  args: { label: 'A very long navigation label that should truncate instead of widening the navbar', href: '#' },
  render: (args) => (
    <div style={{ width: 200 }}>
      <NavLink {...args} />
    </div>
  ),
  play: async ({ canvasElement }) => {
    const link = within(canvasElement).getByRole('link')
    const label = within(link).getByText(/A very long/)
    expect(label.scrollWidth).toBeGreaterThan(label.clientWidth)
    expect(link.getBoundingClientRect().width).toBe(200)
  },
}

export const Compact: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <NavLink label="Inactive" href="#inactive" leftSection="⚙" compact />
      <NavLink label="Active" href="#active" leftSection="⚙" active compact />
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
    label: 'Products',
    children: (
      <>
        <NavLink label="Items" href="#items" />
        <NavLink label="Categories" href="#categories" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.queryByRole('link', { name: 'Items' })).not.toBeInTheDocument()

    await userEvent.click(canvas.getByRole('button', { name: 'Products' }))
    expect(canvas.getByRole('link', { name: 'Items' })).toBeVisible()
    expect(canvas.getByRole('link', { name: 'Categories' })).toBeVisible()
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
