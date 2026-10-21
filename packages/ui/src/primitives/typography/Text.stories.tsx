import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Text } from './Text.tsx'

const meta: Meta<typeof Text> = {
  title: 'Typography/Text',
  component: Text,
}

export default meta

type Story = StoryObj<typeof Text>

export const Default: Story = {
  args: { children: 'Default text' },
}

export const SizeXs: Story = {
  args: { size: 'xs', children: 'Extra small text' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Extra small text')
    expect(getComputedStyle(text).fontSize).toBe('12px')
  },
}

export const SizeXl: Story = {
  args: { size: 'xl', children: 'Extra large text' },
}

export const ToneDimmed: Story = {
  args: { tone: 'dimmed', children: 'Dimmed text' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Dimmed text')
    // Compare against the token variable itself rather than a hardcoded colour literal.
    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(text).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}

export const ToneDanger: Story = {
  args: { tone: 'danger', children: 'Danger text' },
}

export const ToneSuccess: Story = {
  args: { tone: 'success', children: 'Success text' },
}

export const ToneWarning: Story = {
  args: { tone: 'warning', children: 'Warning text' },
}

export const TonePrimary: Story = {
  args: { tone: 'primary', children: 'Primary text' },
}

export const Weight: Story = {
  args: { weight: 650, children: 'Heavy text' },
}

export const Align: Story = {
  args: { align: 'end', children: 'Right-aligned text', style: { display: 'block', width: 240 } },
}

export const Truncate: Story = {
  args: {
    truncate: true,
    style: { width: 200 },
    children: 'A'.repeat(600),
  },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText(/^A+$/)
    expect(getComputedStyle(text).textOverflow).toBe('ellipsis')
    expect(text.scrollWidth).toBeGreaterThan(text.clientWidth)
    expect(text.getBoundingClientRect().width).toBe(200)
  },
}

export const LineClamp: Story = {
  args: {
    lineClamp: 2,
    style: { width: 200 },
    children: 'A long paragraph that wraps across several lines and then gets clamped after the second one so the rest is hidden from view.',
  },
}

export const Render: Story = {
  args: { render: <span />, children: 'Rendered as a span' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Rendered as a span')
    expect(text.tagName).toBe('SPAN')
  },
}
