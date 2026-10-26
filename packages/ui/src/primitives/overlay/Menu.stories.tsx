import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Menu } from './Menu.tsx'

const meta: Meta = {
  title: 'Overlay/Menu',
}

export default meta

type Story = StoryObj

export const Default: StoryObj<{ onRename(): void; onDelete(): void }> = {
  args: { onRename: fn(), onDelete: fn() },
  render: (args) => (
    <Menu.Root>
      <Menu.Trigger render={<Button>Mở menu</Button>} />
      <Menu.Popup>
        <Menu.Item onClick={args.onRename}>Đổi tên</Menu.Item>
        <Menu.Item>Lưu trữ</Menu.Item>
        <Menu.Item color="danger" onClick={args.onDelete}>
          Xóa
        </Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Mở menu' })
    trigger.focus()
    await userEvent.keyboard('{Enter}')

    const body = within(canvasElement.ownerDocument.body)
    const rename = body.getByRole('menuitem', { name: 'Đổi tên' })
    expect(rename).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{ArrowDown}')
    const archive = body.getByRole('menuitem', { name: 'Lưu trữ' })
    expect(archive).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{ArrowDown}')
    const deleteItem = body.getByRole('menuitem', { name: 'Xóa' })
    expect(deleteItem).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{Enter}')
    expect(args.onDelete).toHaveBeenCalledTimes(1)
    // The popup unmounts only once its own closing transition finishes.
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
  },
}

export const Typeahead: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => (
    <Menu.Root>
      <Menu.Trigger render={<Button>Mở menu</Button>} />
      <Menu.Popup>
        <Menu.Item>Đổi tên</Menu.Item>
        <Menu.Item>Lưu trữ</Menu.Item>
        <Menu.Item color="danger">Xóa</Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Mở menu' })
    trigger.focus()
    await userEvent.keyboard('{Enter}')
    await userEvent.keyboard('x')

    const body = within(canvasElement.ownerDocument.body)
    const deleteItem = body.getByRole('menuitem', { name: 'Xóa' })
    expect(deleteItem).toHaveAttribute('data-highlighted', '')
  },
}

export const WithLabelAndSeparator: Story = {
  render: () => (
    <Menu.Root defaultOpen>
      <Menu.Trigger render={<Button>Mở menu</Button>} />
      <Menu.Popup>
        <Menu.Label>Thao tác</Menu.Label>
        <Menu.Item leftSection={<span aria-hidden="true">Đ</span>}>Đổi tên</Menu.Item>
        <Menu.Separator />
        <Menu.Item color="danger" disabled>
          Xóa
        </Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
}
