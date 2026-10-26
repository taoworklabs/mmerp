import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { TextInput } from '../input/TextInput.tsx'
import { Menu } from './Menu.tsx'
import { Dialog, type DialogProps } from './Dialog.tsx'

const meta: Meta<typeof Dialog> = {
  title: 'Overlay/Dialog',
  component: Dialog,
  args: { closeLabel: 'Đóng', onClose: fn() },
  // React elements, not values to edit.
  argTypes: { children: { control: false } },
}

export default meta

type Story = StoryObj<typeof Dialog>

function Demo(props: Omit<DialogProps, 'opened'>) {
  const [opened, setOpened] = useState(false)
  return (
    <>
      <Button onClick={() => setOpened(true)}>Mở hộp thoại</Button>
      <Dialog
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

export const Default: Story = {
  args: { title: 'Gửi duyệt đơn nghỉ phép', children: 'Đơn nghỉ phép sẽ được gửi tới quản lý trực tiếp để phê duyệt.' },
  render: (args) => <Demo {...args} />,
}

export const SizeSm: Story = {
  args: { title: 'Xóa nhân viên?', size: 'sm', children: 'Thao tác này không thể hoàn tác.' },
  render: (args) => <Demo {...args} />,
}

export const SizeLg: Story = {
  args: { title: 'Chi tiết đơn bán hàng', size: 'lg', children: 'Nội dung đơn bán hàng SO-0001.' },
  render: (args) => <Demo {...args} />,
}

export const SizeXl: Story = {
  args: { title: 'Bảng lương tháng 10/2026', size: 'xl', children: 'Nội dung bảng lương của toàn bộ nhân viên.' },
  render: (args) => <Demo {...args} />,
}

export const WithoutCloseButton: Story = {
  args: { title: 'Đang tính lương', withCloseButton: false, children: 'Vui lòng chờ trong giây lát, không đóng cửa sổ này.' },
  render: (args) => <Demo {...args} />,
}

export const ScrollingBody: Story = {
  args: {
    title: 'Điều khoản hợp đồng',
    children: Array.from({ length: 40 }, (_, i) => <p key={i}>Điều {i + 1}</p>),
  },
  render: (args) => <Demo {...args} />,
}

export const WithForm: Story = {
  args: {
    title: 'Sửa hồ sơ nhân viên',
    children: (
      <>
        <TextInput label="Họ và tên" placeholder="Nhập họ và tên" />
        <Dialog.Actions>
          <Button variant="subtle" color="gray">
            Hủy
          </Button>
          <Button type="submit">Lưu</Button>
        </Dialog.Actions>
      </>
    ),
  },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Mở hộp thoại' })
    await userEvent.click(trigger)

    const body = within(canvasElement.ownerDocument.body)
    const dialog = body.getByRole('dialog', { name: 'Sửa hồ sơ nhân viên' })
    expect(dialog.contains(document.activeElement)).toBe(true)

    const input = body.getByRole('textbox', { name: 'Họ và tên' })
    expect(getComputedStyle(input.closest('[data-mmerp-control]') as Element).height).toBe('40px')

    // The actions sit together at the right edge, the primary last, below the fields.
    const save = body.getByRole('button', { name: 'Lưu' }).getBoundingClientRect()
    const cancel = body.getByRole('button', { name: 'Hủy' }).getBoundingClientRect()
    const field = input.closest('[data-mmerp-control]')!.getBoundingClientRect()
    expect(save.right).toBeCloseTo(field.right, 0)
    expect(cancel.right).toBeLessThan(save.left)
    expect(save.left - cancel.right).toBeLessThanOrEqual(16)
    expect(save.top - field.bottom).toBeGreaterThanOrEqual(16)

    await userEvent.tab()
    await userEvent.tab()
    await userEvent.tab()
    await userEvent.tab()
    // Wrapping from the last tabbable element back to the first is redirected by the focus
    // trap asynchronously, so give that a tick before asserting.
    await waitFor(() => expect(dialog.contains(document.activeElement)).toBe(true))

    await userEvent.keyboard('{Escape}')
    expect(args.onClose).toHaveBeenCalledTimes(1)
    // Focus return runs after the popup's own closing transition settles.
    await waitFor(() => expect(document.activeElement).toBe(trigger))
  },
}

export const DialogWithMenu: Story = {
  tags: ['!dev', '!autodocs'],
  args: {
    title: 'Tệp đính kèm',
    children: (
      <Menu.Root>
        <Menu.Trigger render={<Button>Mở menu</Button>} />
        <Menu.Popup>
          <Menu.Item>Đổi tên</Menu.Item>
          <Menu.Item color="danger">Xóa</Menu.Item>
        </Menu.Popup>
      </Menu.Root>
    ),
  },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Mở hộp thoại' }))

    const body = within(canvasElement.ownerDocument.body)
    // The dialog and the menu each mount a beat after their trigger is clicked.
    await userEvent.click(await body.findByRole('button', { name: 'Mở menu' }))
    expect(await body.findByRole('menu')).toBeInTheDocument()

    await userEvent.keyboard('{Escape}')
    // The menu popup unmounts only once its own closing transition finishes.
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
    expect(body.getByRole('dialog', { name: 'Tệp đính kèm' })).toBeInTheDocument()
    expect(args.onClose).not.toHaveBeenCalled()
  },
}
