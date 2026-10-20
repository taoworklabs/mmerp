import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Popover } from './Popover.tsx'

const meta: Meta = {
  title: 'Overlay/Popover',
}

export default meta

type Story = StoryObj

function Demo({ side }: { side: 'top' | 'bottom' | 'left' | 'right' }) {
  return (
    <Popover.Root>
      <Popover.Trigger render={<Button>Open popover</Button>} />
      <Popover.Popup side={side} aria-label="Popover example">
        Popover content
      </Popover.Popup>
    </Popover.Root>
  )
}

async function openAndCheck(canvasElement: HTMLElement) {
  const canvas = within(canvasElement)
  await userEvent.click(canvas.getByRole('button', { name: 'Open popover' }))
  const body = within(canvasElement.ownerDocument.body)
  expect(body.getByText('Popover content')).toBeInTheDocument()
}

export const SideTop: Story = {
  render: () => <Demo side="top" />,
  play: async ({ canvasElement }) => openAndCheck(canvasElement),
}

export const SideBottom: Story = {
  render: () => <Demo side="bottom" />,
}

export const SideLeft: Story = {
  render: () => <Demo side="left" />,
}

export const SideRight: Story = {
  render: () => <Demo side="right" />,
}

export const TargetWidth: Story = {
  render: () => (
    <div style={{ width: 240 }}>
      <Popover.Root>
        <Popover.Trigger render={<Button fullWidth>Open</Button>} />
        <Popover.Popup width="target" aria-label="Popover example">
          Matches trigger width
        </Popover.Popup>
      </Popover.Root>
    </div>
  ),
}
