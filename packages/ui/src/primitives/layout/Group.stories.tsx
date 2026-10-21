import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Text } from '../typography/Text.tsx'
import { Group } from './Group.tsx'

const meta: Meta<typeof Group> = {
  title: 'Layout/Group',
  component: Group,
}

export default meta

type Story = StoryObj<typeof Group>

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

export const Align: Story = {
  args: {
    align: 'start',
    children: (
      <>
        <Item label="Short" />
        <Item label="A taller item" />
      </>
    ),
  },
}

export const Justify: Story = {
  render: () => (
    <Group justify="space-between" style={{ width: 240 }}>
      <Item label="Left" />
      <Item label="Right" />
    </Group>
  ),
}

export const Wrap: Story = {
  args: {
    wrap: false,
    children: (
      <>
        <Item label="First" />
        <Item label="Second" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const group = within(canvasElement).getByText('First').parentElement
    expect(group).not.toBeNull()
    expect(getComputedStyle(group as HTMLElement).flexWrap).toBe('nowrap')
  },
}

export const Grow: Story = {
  render: () => (
    <Group grow style={{ width: 240 }}>
      <Item label="First" />
      <Item label="Second" />
    </Group>
  ),
}

export const ShrinkTruncates: Story = {
  render: () => (
    <div style={{ width: 300 }}>
      <Group wrap={false} data-testid="outer">
        <Group shrink>
          <Text truncate>{'x'.repeat(200)}</Text>
        </Group>
        <Button>A</Button>
      </Group>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const outerGroup = within(canvasElement).getByTestId('outer')
    const text = within(canvasElement).getByText('x'.repeat(200))
    expect(outerGroup.scrollWidth).toBeLessThanOrEqual(300)
    expect(text.scrollWidth).toBeGreaterThan(text.clientWidth)
  },
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
    const group = within(canvasElement).getByText('First').parentElement as HTMLElement
    expect(group.tagName).toBe('UL')
    expect(getComputedStyle(group).columnGap).toBe('16px')
  },
}
