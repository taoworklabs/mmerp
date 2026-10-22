import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Title } from './Title.tsx'

const meta: Meta<typeof Title> = {
  title: 'Typography/Title',
  component: Title,
}

export default meta

type Story = StoryObj<typeof Title>

export const Default: Story = {
  args: { children: 'Page title' },
}

export const OrderH2: Story = {
  args: { order: 2, children: 'Section title' },
  play: async ({ canvasElement }) => {
    const title = within(canvasElement).getByText('Section title')
    expect(title.tagName).toBe('H2')
    expect(getComputedStyle(title).fontSize).toBe('16px')
  },
}

export const OrderH3: Story = {
  args: { order: 3, children: 'Subsection title' },
}
