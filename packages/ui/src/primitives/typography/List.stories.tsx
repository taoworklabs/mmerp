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
      <List.Item>Nộp bản sao căn cước công dân</List.Item>
      <List.Item>Ký hợp đồng lao động</List.Item>
    </List>
  ),
}

export const Ordered: Story = {
  render: () => (
    <List type="ordered">
      <List.Item>Nộp bản sao căn cước công dân</List.Item>
      <List.Item>Ký hợp đồng lao động</List.Item>
    </List>
  ),
}

export const Spacing: Story = {
  render: () => (
    <List spacing="lg">
      <List.Item>Nộp bản sao căn cước công dân</List.Item>
      <List.Item>Ký hợp đồng lao động</List.Item>
    </List>
  ),
}
