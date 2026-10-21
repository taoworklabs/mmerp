import type { Meta, StoryObj } from '@storybook/react-vite'
import { Avatar } from './Avatar.tsx'

const meta: Meta<typeof Avatar> = {
  title: 'Data display/Avatar',
  component: Avatar,
}

export default meta

type Story = StoryObj<typeof Avatar>

// Inline data URI so the story never depends on the network.
const sampleImage = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='64' height='64'%3E%3Crect width='64' height='64' fill='steelblue'/%3E%3C/svg%3E"

export const Image: Story = {
  args: { src: sampleImage, alt: 'Nguyễn Văn A' },
}

export const Initials: Story = {
  args: { alt: 'Nguyễn Văn A', children: 'NA' },
}
