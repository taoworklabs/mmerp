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
      <Timeline.Item title="Đã tạo đơn" active>
        01/10/2026
      </Timeline.Item>
      <Timeline.Item title="Đã giao cho đơn vị vận chuyển">03/10/2026</Timeline.Item>
      <Timeline.Item title="Khách đã nhận hàng">05/10/2026</Timeline.Item>
    </Timeline>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const activeTitle = canvas.getByText('Đã tạo đơn')
    const activeItem = activeTitle.closest('[data-active]')
    expect(activeItem).not.toBeNull()

    const probe = document.createElement('div')
    probe.style.borderColor = 'var(--mmerp-color-primary-6)'
    canvasElement.appendChild(probe)
    const bullet = activeItem!.querySelector('span')!
    expect(getComputedStyle(bullet).borderColor).toBe(getComputedStyle(probe).borderColor)
    probe.remove()

    expect(canvas.getByText('Đã giao cho đơn vị vận chuyển').closest('[data-active]')).toBeNull()
  },
}

export const Coloured: Story = {
  render: () => (
    <Timeline>
      <Timeline.Item title="Trưởng phòng đã duyệt" active color="success">
        01/10/2026
      </Timeline.Item>
      <Timeline.Item title="Giám đốc đang xem" active color="warning" lineVariant="dashed">
        02/10/2026
      </Timeline.Item>
      <Timeline.Item title="Kế toán" active color="gray" />
    </Timeline>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const probe = document.createElement('div')
    probe.style.borderColor = 'var(--mmerp-color-success-6)'
    canvasElement.appendChild(probe)
    const approved = canvas.getByText('Trưởng phòng đã duyệt').closest('[data-active]')!
    expect(getComputedStyle(approved.querySelector('span')!).borderColor).toBe(getComputedStyle(probe).borderColor)
    probe.remove()

    const open = canvas.getByText('Giám đốc đang xem').closest('[data-active]')!
    expect(getComputedStyle(open, '::before').borderLeftStyle).toBe('dashed')
  },
}
