import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { Panel } from './Panel.tsx'

const meta: Meta<typeof Panel> = {
  title: 'Blocks/Panel',
  component: Panel,
  // React elements, not values to edit.
  argTypes: { actions: { control: false }, children: { control: false } },
}

export default meta

type Story = StoryObj<typeof Panel>

export const Framed: Story = {
  args: {
    title: 'Liên hệ',
    children: <Text size="sm">contact@example.com</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'Liên hệ' })).toBeInTheDocument()
  },
}

export const Plain: Story = {
  args: {
    title: 'Lịch sử',
    plain: true,
    children: <Text size="sm">Chưa có hoạt động nào.</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'Lịch sử' })).toBeInTheDocument()
  },
}

export const WithActions: Story = {
  args: {
    title: 'Tệp đính kèm',
    actions: <Button size="xs">Tải lên</Button>,
    children: <Text size="sm">Chưa có tệp nào.</Text>,
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    expect(canvas.getByRole('region', { name: 'Tệp đính kèm' })).toBeInTheDocument()
    expect(canvas.getByRole('button', { name: 'Tải lên' })).toBeInTheDocument()
  },
}
