import type { Meta, StoryObj } from '@storybook/react-vite'
import { List } from './List.tsx'

const meta: Meta<typeof List> = {
  title: 'Typography/List',
  component: List,
}

export default meta

type Story = StoryObj<typeof List>

export const Default: Story = {
  render: () => (
    <List>
      <List.Item>First item</List.Item>
      <List.Item>Second item</List.Item>
    </List>
  ),
}

export const Ordered: Story = {
  render: () => (
    <List type="ordered">
      <List.Item>First item</List.Item>
      <List.Item>Second item</List.Item>
    </List>
  ),
}

export const Spacing: Story = {
  render: () => (
    <List spacing="lg">
      <List.Item>First item</List.Item>
      <List.Item>Second item</List.Item>
    </List>
  ),
}
