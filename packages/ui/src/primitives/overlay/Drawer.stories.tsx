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
