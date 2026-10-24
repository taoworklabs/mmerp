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
      <Popover.Trigger render={<Button>Xem số ngày phép</Button>} />
      <Popover.Popup side={side} aria-label="Số ngày phép">
        Còn 8 ngày phép năm 2026
      </Popover.Popup>
    </Popover.Root>
  )
}

async function openAndCheck(canvasElement: HTMLElement) {
  const canvas = within(canvasElement)
  await userEvent.click(canvas.getByRole('button', { name: 'Xem số ngày phép' }))
  const body = within(canvasElement.ownerDocument.body)
  expect(body.getByText('Còn 8 ngày phép năm 2026')).toBeInTheDocument()
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
        <Popover.Trigger render={<Button fullWidth>Mở</Button>} />
        <Popover.Popup width="target" aria-label="Số ngày phép">
          Rộng bằng nút mở
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
        label="Ngày chứng từ"
        placeholder="Chọn ngày"
        onFocus={() => setOpen(true)}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown') setOpen(true)
        }}
      />
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Popup anchor={inputRef} initialFocus={false} finalFocus={false} aria-label="Lịch chọn ngày">
          Lịch tháng 10/2026
        </Popover.Popup>
      </Popover.Root>
    </>
  )
}

export const AnchoredToInput: Story = {
  render: () => <AnchoredDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const input = canvas.getByRole('textbox', { name: 'Ngày chứng từ' })

    await userEvent.click(input)
    const body = within(canvasElement.ownerDocument.body)
    await waitFor(() => expect(body.getByText('Lịch tháng 10/2026')).toBeInTheDocument())
    expect(input).toHaveFocus()
    // Proves the input was never wrapped by a Trigger (which would merge type="button" onto it).
    expect((input as HTMLInputElement).type).toBe('text')

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(body.queryByText('Lịch tháng 10/2026')).not.toBeInTheDocument())
  },
}
