import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { FormBar } from './FormBar.tsx'

const meta: Meta<typeof FormBar> = {
  title: 'Blocks/FormBar',
  component: FormBar,
}

export default meta

type Story = StoryObj<typeof FormBar>

export const Default: Story = {
  args: {
    children: (
      <>
        <Button variant="default">Hủy</Button>
        <Button type="submit">Lưu</Button>
      </>
    ),
  },
}

export const WithStatus: Story = {
  args: {
    status: (
      <Text size="sm" tone="dimmed" role="status">
        <FormBar.UnsavedDot /> Thay đổi chưa lưu
      </Text>
    ),
    children: <Button type="submit">Lưu</Button>,
  },
}

export const Sticky: Story = {
  render: () => (
    <div data-testid="scrollbox" style={{ height: 300, overflow: 'auto' }} tabIndex={0}>
      <div style={{ height: 2000 }}>Form content…</div>
      <FormBar data-testid="bar">
        <Button type="submit">Lưu</Button>
      </FormBar>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const box = canvas.getByTestId('scrollbox')
    const bar = canvas.getByTestId('bar')

    await waitFor(() => expect(bar).toHaveAttribute('data-stuck'))
    expect(Math.abs(bar.getBoundingClientRect().bottom - box.getBoundingClientRect().bottom)).toBeLessThanOrEqual(1)

    box.scrollTop = box.scrollHeight
    await waitFor(() => expect(bar).not.toHaveAttribute('data-stuck'))
  },
}
