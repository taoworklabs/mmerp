import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
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

export const Scroll: Story = {
  render: () => (
    <div style={{ width: 300 }}>
      <Tabs defaultValue="tab-0">
        <Tabs.List scroll tabIndex={0}>
          {Array.from({ length: 8 }, (_, index) => (
            <Tabs.Tab key={index} value={`tab-${index}`}>
              {`Long tab label number ${index + 1}`}
            </Tabs.Tab>
          ))}
        </Tabs.List>
        {Array.from({ length: 8 }, (_, index) => (
          <Tabs.Panel key={index} value={`tab-${index}`}>
            {`Panel ${index + 1}`}
          </Tabs.Panel>
        ))}
      </Tabs>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const list = canvas.getByRole('tablist')
    expect(list.scrollWidth).toBeGreaterThan(list.clientWidth)

    const tops = new Set(canvas.getAllByRole('tab').map((tab) => tab.getBoundingClientRect().top))
    expect(tops.size).toBe(1)
  },
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
    // The outgoing panel unmounts only after its exit phase.
    await waitFor(() => expect(canvas.queryByText('Profile content')).not.toBeInTheDocument())
    expect(canvas.queryByText('Billing content')).not.toBeInTheDocument()
  },
}
