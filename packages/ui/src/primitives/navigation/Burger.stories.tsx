import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, within } from 'storybook/test'
import { Burger, type BurgerProps } from './Burger.tsx'

const meta: Meta<typeof Burger> = {
  title: 'Navigation/Burger',
  component: Burger,
  args: { 'aria-label': 'Mở/đóng menu' },
}

export default meta

type Story = StoryObj<typeof Burger>

function Controlled(props: BurgerProps) {
  const [open, setOpen] = useState(props.open)
  return <Burger {...props} open={open} onOpenChange={setOpen} />
}

export const Default: Story = {
  args: { open: false },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const button = canvas.getByRole('button', { name: 'Mở/đóng menu' })
    expect(button).toHaveAttribute('aria-expanded', 'false')

    await userEvent.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'true')
  },
}

export const Open: Story = {
  args: { open: true },
  render: (args) => <Controlled {...args} />,
}
