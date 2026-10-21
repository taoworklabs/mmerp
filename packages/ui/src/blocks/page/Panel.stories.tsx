import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { Panel } from './Panel.tsx'

const meta: Meta<typeof Panel> = {
  title: 'Blocks/Panel',
  component: Panel,
}

export default meta

type Story = StoryObj<typeof Panel>

export const Framed: Story = {
  args: {
    title: 'Contact',
    children: <Text size="sm">contact@example.com</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'Contact' })).toBeInTheDocument()
  },
}

export const Plain: Story = {
  args: {
    title: 'History',
    plain: true,
    children: <Text size="sm">No activity yet.</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'History' })).toBeInTheDocument()
  },
}

export const WithActions: Story = {
  args: {
    title: 'Attachments',
    actions: <Button size="xs">Upload</Button>,
    children: <Text size="sm">No files yet.</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'Attachments' })).toBeInTheDocument()
    expect(canvas.getByRole('button', { name: 'Upload' })).toBeInTheDocument()
  },
}
