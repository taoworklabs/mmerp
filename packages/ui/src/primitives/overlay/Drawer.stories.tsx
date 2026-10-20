import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Drawer, type DrawerProps } from './Drawer.tsx'

const meta: Meta<typeof Drawer> = {
  title: 'Overlay/Drawer',
  component: Drawer,
  args: { closeLabel: 'Close', onClose: fn() },
}

export default meta

type Story = StoryObj<typeof Drawer>

function Demo(props: Omit<DrawerProps, 'opened'>) {
  const [opened, setOpened] = useState(false)
  return (
    <>
      <Button onClick={() => setOpened(true)}>Open drawer</Button>
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
  args: { title: 'Left drawer', position: 'left', children: 'Left drawer content' },
  render: (args) => <Demo {...args} />,
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Open drawer' }))
    const body = within(canvasElement.ownerDocument.body)
    const dialog = body.getByRole('dialog', { name: 'Left drawer' })
    expect(dialog).toBeInTheDocument()
    await userEvent.keyboard('{Escape}')
    expect(args.onClose).toHaveBeenCalledTimes(1)
  },
}

export const PositionRight: Story = {
  args: { title: 'Right drawer', position: 'right', children: 'Right drawer content' },
  render: (args) => <Demo {...args} />,
}
