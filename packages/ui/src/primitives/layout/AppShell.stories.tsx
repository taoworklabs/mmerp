import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { page } from 'vitest/browser'
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
      header={<Burger opened={opened} onClick={() => setOpened((o) => !o)} aria-label="Toggle navigation" />}
      navbar={<div>Navbar content</div>}
      navbarOpened={opened}
    >
      <div>Page content</div>
    </AppShell>
  )
}

export const Default: Story = {
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    // The header centres and pads its content; it never stretches a lone child to fill its width.
    const burger = within(canvasElement).getByRole('button', { name: 'Toggle navigation' })
    expect(burger.getBoundingClientRect().width).toBeLessThan(100)
  },
}

function CollapsedDemo() {
  const [collapsed, setCollapsed] = useState(true)
  return (
    <AppShell
      header={<button onClick={() => setCollapsed((c) => !c)}>Toggle rail</button>}
      navbar={<div>Navbar content</div>}
      navbarCollapsed={collapsed}
    >
      <div style={{ flex: 1, minHeight: 0 }}>Page content filling main</div>
    </AppShell>
  )
}

export const CollapsedRail: Story = {
  render: () => <CollapsedDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await page.viewport(1280, 800)
    const navbar = canvas.getByText('Navbar content').closest('nav')!
    await waitFor(() => expect(navbar.getBoundingClientRect().width).toBe(56))

    await userEvent.click(canvas.getByRole('button', { name: 'Toggle rail' }))
    await waitFor(() => expect(navbar.getBoundingClientRect().width).toBe(240))
  },
}

export const HeaderPadding: Story = {
  render: () => (
    <AppShell header={<div data-testid="header-content">Header</div>}>
      <div>Page content</div>
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
    <AppShell header={<div>Header</div>} navbar={<div>Navbar content</div>} navbarFooter={<button>Collapse</button>} navbarOpened>
      <div>Page content</div>
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
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await page.viewport(800, 600)
    const navbar = canvas.getByText('Navbar content').closest('nav')!
    await waitFor(() => expect(navbar.getBoundingClientRect().right).toBeLessThanOrEqual(0.5))
    // Off-screen but still in the tab order would trap keyboard users in an invisible menu.
    await expect(navbar).toHaveAttribute('inert')

    const toggle = canvas.getByRole('button', { name: 'Toggle navigation' })
    await userEvent.click(toggle)
    await waitFor(() => expect(navbar.getBoundingClientRect().left).toBeGreaterThanOrEqual(-0.5))
    await expect(navbar).not.toHaveAttribute('inert')

    // Close it again so the desktop assertion below checks the breakpoint itself, not a flag left over from mobile.
    await userEvent.click(toggle)
    await waitFor(() => expect(navbar.getBoundingClientRect().right).toBeLessThanOrEqual(0.5))

    await page.viewport(1280, 800)
    await waitFor(() => expect(navbar.getBoundingClientRect().left).toBeGreaterThanOrEqual(-0.5))
    await expect(navbar).not.toHaveAttribute('inert')
  },
}
