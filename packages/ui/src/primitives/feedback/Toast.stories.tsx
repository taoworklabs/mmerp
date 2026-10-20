import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { toast, Toaster } from './Toast.tsx'

const meta: Meta<typeof Toaster> = {
  title: 'Feedback/Toast',
  component: Toaster,
  args: { closeLabel: 'Close' },
  decorators: [
    (Story, { args }) => (
      <>
        <Toaster closeLabel={args.closeLabel} />
        <Story />
      </>
    ),
  ],
}

export default meta

type Story = StoryObj<typeof Toaster>

const undoSpy = fn()

export const Success: Story = {
  render: () => <Button onClick={() => toast.show({ message: 'Saved', timeout: 300 })}>Show toast</Button>,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Show toast' }))

    const body = within(canvasElement.ownerDocument.body)
    expect(await body.findByRole('status')).toHaveTextContent('Saved')

    await waitFor(() => expect(body.queryByRole('status')).toBeNull(), { timeout: 1000 })
  },
}

export const Sticky: Story = {
  render: () => <Button onClick={() => toast.show({ message: 'Sticky', timeout: null })}>Show sticky toast</Button>,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Show sticky toast' }))

    const body = within(canvasElement.ownerDocument.body)
    const item = await body.findByRole('status')
    expect(item).toHaveTextContent('Sticky')

    await new Promise((resolve) => setTimeout(resolve, 1000))
    expect(body.getByRole('status')).toHaveTextContent('Sticky')

    await userEvent.click(body.getByRole('button', { name: 'Close' }))
    await waitFor(() => expect(body.queryByRole('status')).not.toBeInTheDocument())
  },
}

export const Undo: Story = {
  render: () => (
    <Button
      onClick={() => toast.show({ message: 'Deleted', timeout: null, action: { label: 'Undo', onClick: undoSpy } })}
    >
      Show toast with undo
    </Button>
  ),
  play: async ({ canvasElement }) => {
    undoSpy.mockClear()
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Show toast with undo' }))

    const body = within(canvasElement.ownerDocument.body)
    await body.findByRole('status')
    await userEvent.click(body.getByRole('button', { name: 'Undo' }))

    expect(undoSpy).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(body.queryByRole('status')).not.toBeInTheDocument())
  },
}

export const Colours: Story = {
  render: () => (
    <Button
      onClick={() => {
        for (const color of ['success', 'danger', 'info', 'warning'] as const) {
          toast.show({ message: `${color} toast`, color, timeout: null })
        }
      }}
    >
      Show one toast per colour
    </Button>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(canvas.getByRole('button', { name: 'Show one toast per colour' }))

    const body = within(canvasElement.ownerDocument.body)
    const items = await body.findAllByRole('status')
    expect(items).toHaveLength(4)

    for (const color of ['success', 'danger', 'info', 'warning'] as const) {
      const probe = document.createElement('div')
      probe.style.borderColor = `var(--mmerp-color-${color}-6)`
      canvasElement.appendChild(probe)

      const item = items.find((el) => el.textContent?.includes(color))
      expect(getComputedStyle(item as Element).borderLeftColor).toBe(getComputedStyle(probe).borderColor)
      probe.remove()
    }
  },
}
