import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Timeline } from './Timeline.tsx'

const meta: Meta<typeof Timeline> = {
  title: 'Data display/Timeline',
  component: Timeline,
}

export default meta

type Story = StoryObj<typeof Timeline>

export const Default: Story = {
  render: () => (
    <Timeline>
      <Timeline.Item title="Order placed" active>
        2026-10-01
      </Timeline.Item>
      <Timeline.Item title="Order shipped">2026-10-03</Timeline.Item>
      <Timeline.Item title="Order delivered">2026-10-05</Timeline.Item>
    </Timeline>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const activeTitle = canvas.getByText('Order placed')
    const activeItem = activeTitle.closest('[data-active]')
    expect(activeItem).not.toBeNull()

    const probe = document.createElement('div')
    probe.style.borderColor = 'var(--mmerp-color-primary-6)'
    canvasElement.appendChild(probe)
    const bullet = activeItem!.querySelector('span')!
    expect(getComputedStyle(bullet).borderColor).toBe(getComputedStyle(probe).borderColor)
    probe.remove()

    expect(canvas.getByText('Order shipped').closest('[data-active]')).toBeNull()
  },
}
