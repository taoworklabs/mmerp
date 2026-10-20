import type { Meta, StoryObj } from '@storybook/react-vite'
import { useState } from 'react'
import { expect, userEvent, within } from 'storybook/test'
import { Burger, type BurgerProps } from './Burger.tsx'

const meta: Meta<typeof Burger> = {
  title: 'Navigation/Burger',
  component: Burger,
  args: { 'aria-label': 'Toggle navigation' },
}

export default meta

type Story = StoryObj<typeof Burger>

function Controlled(props: BurgerProps) {
  const [opened, setOpened] = useState(props.opened)
  return <Burger {...props} opened={opened} onClick={() => setOpened((o) => !o)} />
}

export const Default: Story = {
  args: { opened: false },
  render: (args) => <Controlled {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const button = canvas.getByRole('button', { name: 'Toggle navigation' })
    expect(button).toHaveAttribute('aria-expanded', 'false')

    await userEvent.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'true')
  },
}

export const Opened: Story = {
  args: { opened: true },
  render: (args) => <Controlled {...args} />,
}
