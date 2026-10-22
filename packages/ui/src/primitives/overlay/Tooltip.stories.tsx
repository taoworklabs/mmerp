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
  args: { label: 'Xuất danh sách ra tệp Excel', children: <Button>Xuất file</Button> },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Xuất file' })
    // :focus-visible (which the tooltip's focus trigger requires) only matches
    // keyboard-driven focus, not a plain programmatic .focus() call.
    await userEvent.tab()
    expect(trigger).toHaveFocus()

    const body = within(canvasElement.ownerDocument.body)
    const tooltip = await body.findByRole('tooltip')
    expect(tooltip).toHaveTextContent('Xuất danh sách ra tệp Excel')
  },
}

export const SideBottom: Story = {
  args: { label: 'Xuất danh sách ra tệp Excel', side: 'bottom', children: <Button>Xuất file</Button> },
}

export const SideLeft: Story = {
  args: { label: 'Xuất danh sách ra tệp Excel', side: 'left', children: <Button>Xuất file</Button> },
}

export const SideRight: Story = {
  args: { label: 'Xuất danh sách ra tệp Excel', side: 'right', children: <Button>Xuất file</Button> },
}

export const Disabled: Story = {
  args: { label: 'Xuất danh sách ra tệp Excel', disabled: true, children: <Button>Xuất file</Button> },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const trigger = canvas.getByRole('button', { name: 'Xuất file' })
    await userEvent.tab()
    expect(trigger).toHaveFocus()
    const body = within(canvasElement.ownerDocument.body)
    expect(body.queryByRole('tooltip')).not.toBeInTheDocument()
  },
}
