import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Burger } from '../navigation/Burger.tsx'
import { AppShell } from './AppShell.tsx'

const meta: Meta<typeof AppShell> = {
  title: 'Layout/AppShell',
  component: AppShell,
}

export default meta

type Story = StoryObj<typeof AppShell>

function Demo() {
  const [opened, setOpened] = useState(false)
  return (
    <AppShell
      header={<Burger opened={opened} onClick={() => setOpened((o) => !o)} aria-label="Mở/đóng menu" />}
      navbar={<div>Nội dung thanh điều hướng</div>}
      navbarOpened={opened}
    >
      <div>Nội dung trang</div>
    </AppShell>
  )
}

export const Default: Story = {
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    // The header centres and pads its content; it never stretches a lone child to fill its width.
    const burger = within(canvasElement).getByRole('button', { name: 'Mở/đóng menu' })
    expect(burger.getBoundingClientRect().width).toBeLessThan(100)
  },
}

function CollapsedDemo() {
  const [collapsed, setCollapsed] = useState(true)
  return (
    <AppShell
      header={<button onClick={() => setCollapsed((c) => !c)}>Thu gọn/mở rộng thanh bên</button>}
      navbar={<div>Nội dung thanh điều hướng</div>}
      navbarCollapsed={collapsed}
    >
      <div style={{ flex: 1, minHeight: 0 }}>Nội dung trang lấp đầy vùng chính</div>
    </AppShell>
  )
}

export const CollapsedRail: Story = {
  render: () => <CollapsedDemo />,
  globals: { viewport: { value: 'desktop' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    const navbar = canvas.getByText('Nội dung thanh điều hướng').closest('nav')!
    await waitFor(() => expect(navbar.getBoundingClientRect().width).toBe(56))

    await userEvent.click(canvas.getByRole('button', { name: 'Thu gọn/mở rộng thanh bên' }))
    await waitFor(() => expect(navbar.getBoundingClientRect().width).toBe(240))
  },
}

export const HeaderPadding: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => (
    <AppShell header={<div data-testid="header-content">Thanh đầu trang</div>}>
      <div>Nội dung trang</div>
    </AppShell>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const header = canvasElement.querySelector('header')!
    const content = canvas.getByTestId('header-content')
    const headerRect = header.getBoundingClientRect()
    const contentRect = content.getBoundingClientRect()
    // Inline padding keeps content off the header's own edges instead of flush against them.
    expect(contentRect.left).toBeGreaterThan(headerRect.left)
    // align-items: center keeps it vertically centred within the header's fixed height.
    const topGap = contentRect.top - headerRect.top
    const bottomGap = headerRect.bottom - contentRect.bottom
    expect(Math.abs(topGap - bottomGap)).toBeLessThanOrEqual(1.5)
  },
}

function FooterDemo() {
  return (
    <AppShell header={<div>Thanh đầu trang</div>} navbar={<div>Nội dung thanh điều hướng</div>} navbarFooter={<button>Thu gọn</button>} navbarOpened>
      <div>Nội dung trang</div>
    </AppShell>
  )
}

// The breakpoint-driven hide/show of navbarFooter is covered by AppShell.test.tsx, where
// page.viewport() reliably drives the layout; here it's enough to show the slot renders.
export const NavbarFooter: Story = {
  render: () => <FooterDemo />,
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('button')).not.toBeNull()
  },
}

export const OffCanvasBelowBreakpoint: Story = {
  render: () => <Demo />,
  globals: { viewport: { value: 'drawer' } },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    const navbar = canvas.getByText('Nội dung thanh điều hướng').closest('nav')!
    await waitFor(() => expect(navbar.getBoundingClientRect().right).toBeLessThanOrEqual(0.5))
    // Off-screen but still in the tab order would trap keyboard users in an invisible menu.
    await expect(navbar).toHaveAttribute('inert')

    const toggle = canvas.getByRole('button', { name: 'Mở/đóng menu' })
    await userEvent.click(toggle)
    await waitFor(() => expect(navbar.getBoundingClientRect().left).toBeGreaterThanOrEqual(-0.5))
    await expect(navbar).not.toHaveAttribute('inert')

    await userEvent.click(toggle)
    await waitFor(() => expect(navbar.getBoundingClientRect().right).toBeLessThanOrEqual(0.5))
  },
}

// Same closed shell as OffCanvasBelowBreakpoint: at desktop widths the navbar shows without being opened.
export const NavbarShownAtDesktop: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => <Demo />,
  globals: { viewport: { value: 'desktop' } },
  play: async ({ canvasElement }) => {
    const navbar = within(canvasElement).getByText('Nội dung thanh điều hướng').closest('nav')!
    await waitFor(() => expect(navbar.getBoundingClientRect().left).toBeGreaterThanOrEqual(-0.5))
    await expect(navbar).not.toHaveAttribute('inert')
  },
}

// Header content wider than the window must not widen the whole page.
export const WideHeaderOnMobile: Story = {
  tags: ['!dev', '!autodocs'],
  globals: { viewport: { value: 'mobile' } },
  render: () => (
    <AppShell header={<div style={{ flex: 'none', width: 600 }}>Tiêu đề rất rộng</div>}>
      <div>Nội dung trang</div>
    </AppShell>
  ),
  play: async ({ canvasElement }) => {
    const main = canvasElement.querySelector('main')!
    expect(main.getBoundingClientRect().width).toBeLessThanOrEqual(canvasElement.ownerDocument.documentElement.clientWidth)
  },
}
