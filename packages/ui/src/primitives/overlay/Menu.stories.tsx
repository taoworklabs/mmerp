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
      <Menu.Trigger render={<Button>Open menu</Button>} />
      <Menu.Popup>
        <Menu.Item onClick={args.onRename}>Rename</Menu.Item>
        <Menu.Item>Archive</Menu.Item>
        <Menu.Item color="danger" onClick={args.onDelete}>
          Delete
        </Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Open menu' })
    trigger.focus()
    await userEvent.keyboard('{Enter}')

    const body = within(canvasElement.ownerDocument.body)
    const rename = body.getByRole('menuitem', { name: 'Rename' })
    expect(rename).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{ArrowDown}')
    const archive = body.getByRole('menuitem', { name: 'Archive' })
    expect(archive).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{ArrowDown}')
    const deleteItem = body.getByRole('menuitem', { name: 'Delete' })
    expect(deleteItem).toHaveAttribute('data-highlighted', '')

    await userEvent.keyboard('{Enter}')
    expect(args.onDelete).toHaveBeenCalledTimes(1)
    // The popup unmounts only once its own closing transition finishes.
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
  },
}

export const Typeahead: Story = {
  render: () => (
    <Menu.Root>
      <Menu.Trigger render={<Button>Open menu</Button>} />
      <Menu.Popup>
        <Menu.Item>Rename</Menu.Item>
        <Menu.Item>Archive</Menu.Item>
        <Menu.Item color="danger">Delete</Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Open menu' })
    trigger.focus()
    await userEvent.keyboard('{Enter}')
    await userEvent.keyboard('d')

    const body = within(canvasElement.ownerDocument.body)
    const deleteItem = body.getByRole('menuitem', { name: 'Delete' })
    expect(deleteItem).toHaveAttribute('data-highlighted', '')
  },
}

export const WithLabelAndSeparator: Story = {
  render: () => (
    <Menu.Root defaultOpen>
      <Menu.Trigger render={<Button>Open menu</Button>} />
      <Menu.Popup>
        <Menu.Label>Actions</Menu.Label>
        <Menu.Item leftSection={<span aria-hidden="true">R</span>}>Rename</Menu.Item>
        <Menu.Separator />
        <Menu.Item color="danger" disabled>
          Delete
        </Menu.Item>
      </Menu.Popup>
    </Menu.Root>
  ),
}
