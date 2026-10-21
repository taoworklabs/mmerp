import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Menu } from '../../primitives/overlay/Menu.tsx'
import { MoreActions } from './MoreActions.tsx'

const meta: Meta<typeof MoreActions> = {
  title: 'Blocks/MoreActions',
  component: MoreActions,
  args: { label: 'More actions' },
}

export default meta

type Story = StoryObj<typeof MoreActions>

export const Row: StoryObj<{ onArchive(): void }> = {
  args: { onArchive: fn() },
  render: (args) => (
    <MoreActions label="More actions">
      <Menu.Item onClick={args.onArchive}>Archive</Menu.Item>
      <Menu.Item color="danger">Delete</Menu.Item>
    </MoreActions>
  ),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'More actions' })
    await userEvent.click(trigger)

    const body = within(canvasElement.ownerDocument.body)
    const archive = await body.findByRole('menuitem', { name: 'Archive' })
    await userEvent.click(archive)
    expect(args.onArchive).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
  },
}

export const Page: Story = {
  args: { place: 'page' },
  render: (args) => (
    <MoreActions {...args}>
      <Menu.Item>Export</Menu.Item>
    </MoreActions>
  ),
}

export const Empty: Story = {
  render: (args) => <MoreActions {...args}>{null}</MoreActions>,
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelector('button')).toBeNull()
  },
}
