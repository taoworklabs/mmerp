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
  args: { closeLabel: 'Close', onClose: fn() },
}

export default meta

type Story = StoryObj<typeof Dialog>

function Demo(props: Omit<DialogProps, 'opened'>) {
  const [opened, setOpened] = useState(false)
  return (
    <>
      <Button onClick={() => setOpened(true)}>Open dialog</Button>
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
  args: { title: 'Dialog title', children: 'Dialog content' },
  render: (args) => <Demo {...args} />,
}

export const SizeSm: Story = {
  args: { title: 'Small dialog', size: 'sm', children: 'Small content' },
  render: (args) => <Demo {...args} />,
}

export const SizeLg: Story = {
  args: { title: 'Large dialog', size: 'lg', children: 'Large content' },
  render: (args) => <Demo {...args} />,
}

export const SizeXl: Story = {
  args: { title: 'Extra large dialog', size: 'xl', children: 'Extra large content' },
  render: (args) => <Demo {...args} />,
}

export const WithoutCloseButton: Story = {
  args: { title: 'No close button', withCloseButton: false, children: 'Content without a close button' },
  render: (args) => <Demo {...args} />,
}

export const ScrollingBody: Story = {
  args: {
    title: 'Scrolling body',
    children: Array.from({ length: 40 }, (_, i) => <p key={i}>Line {i + 1}</p>),
  },
  render: (args) => <Demo {...args} />,
}

export const WithForm: Story = {
  args: {
    title: 'Edit profile',
    children: (
      <>
        <TextInput label="Name" placeholder="Your name" />
        <Button type="submit" style={{ marginTop: 16 }}>
          Save
        </Button>
      </>
    ),
  },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Open dialog' })
    await userEvent.click(trigger)

    const body = within(canvasElement.ownerDocument.body)
    const dialog = body.getByRole('dialog', { name: 'Edit profile' })
    expect(dialog.contains(document.activeElement)).toBe(true)

    const input = body.getByRole('textbox', { name: 'Name' })
    expect(getComputedStyle(input.closest('[data-mmerp-control]') as Element).height).toBe('40px')

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
  args: {
    title: 'Dialog with a menu',
    children: (
      <Menu.Root>
        <Menu.Trigger render={<Button>Open menu</Button>} />
        <Menu.Popup>
          <Menu.Item>Rename</Menu.Item>
          <Menu.Item color="danger">Delete</Menu.Item>
        </Menu.Popup>
      </Menu.Root>
    ),
  },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Open dialog' }))

    const body = within(canvasElement.ownerDocument.body)
    await userEvent.click(body.getByRole('button', { name: 'Open menu' }))
    expect(body.getByRole('menu')).toBeInTheDocument()

    await userEvent.keyboard('{Escape}')
    // The menu popup unmounts only once its own closing transition finishes.
    await waitFor(() => expect(body.queryByRole('menu')).not.toBeInTheDocument())
    expect(body.getByRole('dialog', { name: 'Dialog with a menu' })).toBeInTheDocument()
    expect(args.onClose).not.toHaveBeenCalled()
  },
}
