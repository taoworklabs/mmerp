import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { Tabs } from './Tabs.tsx'

const meta: Meta<typeof Tabs> = {
  title: 'Navigation/Tabs',
  component: Tabs,
}

export default meta

type Story = StoryObj<typeof Tabs>

function Demo() {
  return (
    <Tabs defaultValue="profile">
      <Tabs.List>
        <Tabs.Tab value="profile">Profile</Tabs.Tab>
        <Tabs.Tab value="settings">Settings</Tabs.Tab>
        <Tabs.Tab value="billing">Billing</Tabs.Tab>
      </Tabs.List>
      <Tabs.Panel value="profile">Profile content</Tabs.Panel>
      <Tabs.Panel value="settings">Settings content</Tabs.Panel>
      <Tabs.Panel value="billing">Billing content</Tabs.Panel>
    </Tabs>
  )
}

export const Default: Story = {
  render: () => <Demo />,
}

export const ArrowRightSelectsNextTab: Story = {
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const profile = canvas.getByRole('tab', { name: 'Profile' })
    profile.focus()
    await userEvent.keyboard('{ArrowRight}')

    const settings = canvas.getByRole('tab', { name: 'Settings' })
    expect(settings).toHaveFocus()
    expect(settings).toHaveAttribute('data-active', '')

    expect(canvas.getByText('Settings content')).toBeVisible()
    expect(canvas.queryByText('Profile content')).not.toBeInTheDocument()
    expect(canvas.queryByText('Billing content')).not.toBeInTheDocument()
  },
}
