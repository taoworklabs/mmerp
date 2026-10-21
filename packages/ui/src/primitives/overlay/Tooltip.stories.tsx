import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Tooltip } from './Tooltip.tsx'

const meta: Meta<typeof Tooltip> = {
  title: 'Overlay/Tooltip',
  component: Tooltip,
}

export default meta

type Story = StoryObj<typeof Tooltip>

export const Default: Story = {
  args: { label: 'Helpful hint', children: <Button>Hover me</Button> },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Hover me' })
    // :focus-visible (which the tooltip's focus trigger requires) only matches
    // keyboard-driven focus, not a plain programmatic .focus() call.
    await userEvent.tab()
    expect(trigger).toHaveFocus()

    const body = within(canvasElement.ownerDocument.body)
    const tooltip = await body.findByRole('tooltip')
    expect(tooltip).toHaveTextContent('Helpful hint')
  },
}

export const SideBottom: Story = {
  args: { label: 'Helpful hint', side: 'bottom', children: <Button>Hover me</Button> },
}

export const SideLeft: Story = {
  args: { label: 'Helpful hint', side: 'left', children: <Button>Hover me</Button> },
}

export const SideRight: Story = {
  args: { label: 'Helpful hint', side: 'right', children: <Button>Hover me</Button> },
}

export const Disabled: Story = {
  args: { label: 'Helpful hint', disabled: true, children: <Button>Hover me</Button> },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Hover me' })
    await userEvent.tab()
    expect(trigger).toHaveFocus()
    const body = within(canvasElement.ownerDocument.body)
    expect(body.queryByRole('tooltip')).not.toBeInTheDocument()
  },
}
