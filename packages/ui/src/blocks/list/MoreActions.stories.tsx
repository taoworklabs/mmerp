import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Menu } from '../../primitives/overlay/Menu.tsx'
import { MoreActions } from './MoreActions.tsx'

const meta: Meta<typeof MoreActions> = {
  title: 'Blocks/MoreActions',
  component: MoreActions,
  args: { label: 'Thao tác khác' },
}

export default meta

type Story = StoryObj<typeof MoreActions>

export const Row: StoryObj<{ onArchive(): void }> = {
  args: { onArchive: fn() },
  render: (args) => (
    <MoreActions label="Thao tác khác">
      <Menu.Item onClick={args.onArchive}>Lưu trữ</Menu.Item>
      <Menu.Item color="danger">Xóa</Menu.Item>
    </MoreActions>
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Thao tác khác' })
    await userEvent.click(trigger)

    const body = within(canvasElement.ownerDocument.body)
    const archive = await body.findByRole('menuitem', { name: 'Lưu trữ' })
    await userEvent.click(archive)
    expect(args.onArchive).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
  },
}

export const Page: Story = {
  args: { place: 'page' },
  render: (args) => (
    <MoreActions {...args}>
      <Menu.Item>Xuất file</Menu.Item>
    </MoreActions>
  ),
}

// No actions, no trigger: the menu renders nothing rather than an empty dropdown.
export const Empty: Story = {
  tags: ['!dev', '!autodocs'],
  render: (args) => <MoreActions {...args}>{null}</MoreActions>,
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('button')).toBeNull()
  },
}
