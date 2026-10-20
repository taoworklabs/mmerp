import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Stack } from './Stack.tsx'

const meta: Meta<typeof Stack> = {
  title: 'Layout/Stack',
  component: Stack,
}

export default meta

type Story = StoryObj<typeof Stack>

function Item({ label }: { label: string }) {
  return <div>{label}</div>
}

export const Default: Story = {
  args: {
    children: (
      <>
        <Item label="First" />
        <Item label="Second" />
      </>
    ),
  },
}

export const Gap: Story = {
  args: {
    gap: 'md',
    children: (
      <>
        <Item label="First" />
        <Item label="Second" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const stack = within(canvasElement).getByText('First').parentElement
    expect(stack).not.toBeNull()
    expect(getComputedStyle(stack as HTMLElement).rowGap).toBe('16px')
  },
}

export const Align: Story = {
  args: {
    align: 'center',
    children: (
      <>
        <Item label="Narrow" />
        <Item label="A much wider item" />
      </>
    ),
  },
}

export const Justify: Story = {
  render: () => (
    <Stack justify="space-between" style={{ height: 120 }}>
      <Item label="Top" />
      <Item label="Bottom" />
    </Stack>
  ),
}

export const Render: Story = {
  args: {
    render: <ul />,
    gap: 'md',
    children: (
      <>
        <li>First</li>
        <li>Second</li>
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const stack = within(canvasElement).getByText('First').parentElement as HTMLElement
    expect(stack.tagName).toBe('UL')
    expect(getComputedStyle(stack).rowGap).toBe('16px')
  },
}
