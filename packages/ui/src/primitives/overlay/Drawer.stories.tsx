import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Drawer, type DrawerProps } from './Drawer.tsx'

const meta: Meta<typeof Drawer> = {
  title: 'Overlay/Drawer',
  component: Drawer,
  args: { closeLabel: 'Đóng', onClose: fn() },
}

export default meta

type Story = StoryObj<typeof Drawer>

function Demo(props: Omit<DrawerProps, 'opened'>) {
  const [opened, setOpened] = useState(false)
  return (
    <>
      <Button onClick={() => setOpened(true)}>Mở bảng bên</Button>
      <Drawer
        {...props}
        opened={opened}
        onClose={() => {
          setOpened(false)
          props.onClose()
        }}
      />
    </>
  )
}

export const PositionLeft: Story = {
  args: { title: 'Bộ lọc nâng cao', position: 'left', children: 'Lọc theo phòng ban, chức danh và ngày vào làm.' },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Mở bảng bên' }))
    const body = within(canvasElement.ownerDocument.body)
    const dialog = body.getByRole('dialog', { name: 'Bộ lọc nâng cao' })
    expect(dialog).toBeInTheDocument()
    await userEvent.keyboard('{Escape}')
    expect(args.onClose).toHaveBeenCalledTimes(1)
  },
}

export const PositionRight: Story = {
  args: { title: 'Chi tiết nhân viên', position: 'right', children: 'Nguyễn Văn An – Phòng Kế toán.' },
  render: (args) => <Demo {...args} />,
}

// A record preview beside a list: the page stays clickable and an outside click leaves it open.
export const NonModal: Story = {
  args: { title: 'Xem nhanh', modal: false, children: 'Nguyễn Văn An – Phòng Kế toán.' },
  render: (args) => {
    function Page() {
      const [clicks, setClicks] = useState(0)
      return (
        <>
          <Demo {...args} />
          <Button variant="default" onClick={() => setClicks((n) => n + 1)}>
            Dòng khác ({clicks})
          </Button>
        </>
      )
    }
    return <Page />
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Mở bảng bên' }))
    const body = within(canvasElement.ownerDocument.body)
    expect(await body.findByRole('dialog', { name: 'Xem nhanh' })).toBeInTheDocument()
    await userEvent.click(canvas.getByRole('button', { name: 'Dòng khác (0)' }))
    expect(canvas.getByRole('button', { name: 'Dòng khác (1)' })).toBeInTheDocument()
    expect(body.getByRole('dialog', { name: 'Xem nhanh' })).toBeInTheDocument()
    expect(args.onClose).not.toHaveBeenCalled()
  },
}

export const PositionBottom: Story = {
  args: { title: 'Bước duyệt', position: 'bottom', size: '85%', children: 'Người duyệt: quản lý trực tiếp.' },
  render: (args) => <Demo {...args} />,
}
