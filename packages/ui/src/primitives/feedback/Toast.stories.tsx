import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { toast, Toaster } from './Toast.tsx'

const meta: Meta<typeof Toaster> = {
  title: 'Feedback/Toast',
  component: Toaster,
  args: { closeLabel: 'Đóng' },
  decorators: [
    (Story, { args }) => (
      <>
        <Toaster closeLabel={args.closeLabel} />
        <Story />
      </>
    ),
  ],
}

export default meta

type Story = StoryObj<typeof Toaster>

const undoSpy = fn()

const colourMessages = {
  success: 'Đã ghi sổ phiếu lương',
  danger: 'Không gửi được đơn nghỉ phép',
  info: 'Báo giá đang chờ duyệt',
  warning: 'Hợp đồng sắp hết hạn',
} as const

export const Success: Story = {
  render: () => <Button onClick={() => toast.show({ message: 'Đã lưu khách hàng', timeout: 300 })}>Hiện thông báo</Button>,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Hiện thông báo' }))

    const body = within(canvasElement.ownerDocument.body)
    expect(await body.findByRole('status')).toHaveTextContent('Đã lưu khách hàng')

    await waitFor(() => expect(body.queryByRole('status')).toBeNull(), { timeout: 1000 })
  },
}

export const Sticky: Story = {
  render: () => <Button onClick={() => toast.show({ message: 'Mất kết nối máy chủ, dữ liệu chưa được đồng bộ', timeout: null })}>Hiện thông báo không tự ẩn</Button>,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Hiện thông báo không tự ẩn' }))

    const body = within(canvasElement.ownerDocument.body)
    const item = await body.findByRole('status')
    expect(item).toHaveTextContent('Mất kết nối máy chủ, dữ liệu chưa được đồng bộ')

    await new Promise((resolve) => setTimeout(resolve, 1000))
    expect(body.getByRole('status')).toHaveTextContent('Mất kết nối máy chủ, dữ liệu chưa được đồng bộ')

    await userEvent.click(body.getByRole('button', { name: 'Đóng' }))
    await waitFor(() => expect(body.queryByRole('status')).not.toBeInTheDocument())
  },
}

export const Undo: Story = {
  render: () => (
    <Button
      onClick={() => toast.show({ message: 'Đã xóa đơn bán hàng SO-0012', timeout: null, action: { label: 'Hoàn tác', onClick: undoSpy } })}
    >
      Hiện thông báo có hoàn tác
    </Button>
  ),
  play: async ({ canvasElement }) => {
    undoSpy.mockClear()
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Hiện thông báo có hoàn tác' }))

    const body = within(canvasElement.ownerDocument.body)
    await body.findByRole('status')
    await userEvent.click(body.getByRole('button', { name: 'Hoàn tác' }))

    expect(undoSpy).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(body.queryByRole('status')).not.toBeInTheDocument())
  },
}

export const Colours: Story = {
  render: () => (
    <Button
      onClick={() => {
        for (const color of ['success', 'danger', 'info', 'warning'] as const) {
          toast.show({ message: colourMessages[color], color, timeout: null })
        }
      }}
    >
      Hiện mỗi màu một thông báo
    </Button>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Hiện mỗi màu một thông báo' }))

    const body = within(canvasElement.ownerDocument.body)
    const items = await body.findAllByRole('status')
    expect(items).toHaveLength(4)

    for (const color of ['success', 'danger', 'info', 'warning'] as const) {
      const probe = document.createElement('div')
      probe.style.borderColor = `var(--mmerp-color-${color}-6)`
      canvasElement.appendChild(probe)

      const item = items.find((el) => el.textContent?.includes(colourMessages[color]))
      expect(getComputedStyle(item as Element).borderLeftColor).toBe(getComputedStyle(probe).borderColor)
      probe.remove()
    }
  },
}
