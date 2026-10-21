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
