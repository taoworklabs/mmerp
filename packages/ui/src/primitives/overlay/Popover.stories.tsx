import type { Meta, StoryObj } from '@storybook/react-vite'
import { useRef, useState } from 'react'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { TextInput } from '../input/TextInput.tsx'
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

// DateInput's own route: a controlled Root with no Trigger, anchored to the input itself via
// `anchor`, so the input stays a plain text input — never a toggle button — and keeps focus.
function AnchoredDemo() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [open, setOpen] = useState(false)
  return (
    <>
      <TextInput
        ref={inputRef}
        label="Date"
        placeholder="Pick a date"
        onFocus={() => setOpen(true)}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown') setOpen(true)
        }}
      />
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Popup anchor={inputRef} initialFocus={false} finalFocus={false} aria-label="Date picker">
          Calendar placeholder
        </Popover.Popup>
      </Popover.Root>
    </>
  )
}

export const AnchoredToInput: Story = {
  render: () => <AnchoredDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const input = canvas.getByRole('textbox', { name: 'Date' })

    await userEvent.click(input)
    const body = within(canvasElement.ownerDocument.body)
    await waitFor(() => expect(body.getByText('Calendar placeholder')).toBeInTheDocument())
    expect(input).toHaveFocus()
    // Proves the input was never wrapped by a Trigger (which would merge type="button" onto it).
    expect((input as HTMLInputElement).type).toBe('text')

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByText('Calendar placeholder')).not.toBeInTheDocument())
  },
}
