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
        <Tabs.Tab value="profile">Hồ sơ</Tabs.Tab>
        <Tabs.Tab value="settings">Cài đặt</Tabs.Tab>
        <Tabs.Tab value="billing">Thanh toán</Tabs.Tab>
      </Tabs.List>
      <Tabs.Panel value="profile">Nội dung hồ sơ</Tabs.Panel>
      <Tabs.Panel value="settings">Nội dung cài đặt</Tabs.Panel>
      <Tabs.Panel value="billing">Nội dung thanh toán</Tabs.Panel>
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
        <Tabs.List scroll>
          {Array.from({ length: 8 }, (_, index) => (
            <Tabs.Tab key={index} value={`tab-${index}`}>
              {`Nhãn thẻ khá dài số ${index + 1}`}
            </Tabs.Tab>
          ))}
        </Tabs.List>
        {Array.from({ length: 8 }, (_, index) => (
          <Tabs.Panel key={index} value={`tab-${index}`}>
            {`Nội dung thẻ ${index + 1}`}
          </Tabs.Panel>
        ))}
      </Tabs>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const list = canvas.getByRole('tablist')
    expect(list.scrollWidth).toBeGreaterThan(list.clientWidth)
    // No tabIndex was passed here: scroll alone must make the scrollable region a tab stop.
    expect(list.tabIndex).toBe(0)

    const tops = new Set(canvas.getAllByRole('tab').map((tab) => tab.getBoundingClientRect().top))
    expect(tops.size).toBe(1)
  },
}

export const OnRule: Story = {
  render: () => (
    <Tabs defaultValue="overview">
      <Tabs.List onRule>
        <Tabs.Tab value="overview">Tổng quan</Tabs.Tab>
        <Tabs.Tab value="orders">Đơn bán hàng</Tabs.Tab>
      </Tabs.List>
      <Tabs.Panel value="overview">Nội dung tổng quan</Tabs.Panel>
      <Tabs.Panel value="orders">Nội dung đơn bán hàng</Tabs.Panel>
    </Tabs>
  ),
  play: async ({ canvasElement }) => {
    const list = within(canvasElement).getByRole('tablist')
    expect(getComputedStyle(list).borderBottomWidth).toBe('0px')
  },
}

export const ArrowRightSelectsNextTab: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => <Demo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const profile = canvas.getByRole('tab', { name: 'Hồ sơ' })
    profile.focus()
    await userEvent.keyboard('{ArrowRight}')

    const settings = canvas.getByRole('tab', { name: 'Cài đặt' })
    expect(settings).toHaveFocus()
    expect(settings).toHaveAttribute('data-active', '')

    expect(canvas.getByText('Nội dung cài đặt')).toBeVisible()
    // The outgoing panel unmounts only after its exit phase.
    await waitFor(() => expect(canvas.queryByText('Nội dung hồ sơ')).not.toBeInTheDocument())
    expect(canvas.queryByText('Nội dung thanh toán')).not.toBeInTheDocument()
  },
}
