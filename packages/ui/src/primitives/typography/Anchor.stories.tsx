import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Anchor } from './Anchor.tsx'

const meta: Meta<typeof Anchor> = {
  title: 'Typography/Anchor',
  component: Anchor,
}

export default meta

type Story = StoryObj<typeof Anchor>

export const Default: Story = {
  args: { href: '#', children: 'A link' },
}

export const Tone: Story = {
  args: { href: '#', tone: 'danger', children: 'A danger-toned link' },
}

export const UnderlineAlways: Story = {
  args: { href: '#', underline: 'always', children: 'Always underlined' },
}

export const UnderlineNever: Story = {
  args: { href: '#', underline: 'never', children: 'Never underlined' },
}

export const RenderAsButton: Story = {
  args: { render: <button type="button" />, children: 'Acts like an anchor, renders a button' },
  play: async ({ canvasElement }) => {
    const element = within(canvasElement).getByText('Acts like an anchor, renders a button')
    expect(element.tagName).toBe('BUTTON')
  },
}
