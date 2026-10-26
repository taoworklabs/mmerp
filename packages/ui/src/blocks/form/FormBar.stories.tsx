import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { TextInput } from '../../primitives/input/TextInput.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { FormBar } from './FormBar.tsx'

const meta: Meta<typeof FormBar> = {
  title: 'Blocks/FormBar',
  component: FormBar,
  // The bar spans its form; shown below a form-width stand-in so it isn't read as a stray rule.
  decorators: [
    (Story) => (
      <div style={{ maxWidth: 640, display: 'grid', gap: 'var(--mmerp-space-lg)' }}>
        <TextInput label="Họ tên" defaultValue="Nguyễn Văn An" />
        <Story />
      </div>
    ),
  ],
  // React elements, not values to edit.
  argTypes: { children: { control: false }, status: { control: false } },
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

// A failed submit on a phone: the error line wraps above the buttons instead of overflowing.
export const ErrorStatusNarrow: Story = {
  decorators: [(Story) => <div style={{ width: 340 }}><Story /></div>],
  args: {
    'data-testid': 'bar',
    status: (
      <>
        <Text size="sm" role="status">
          <FormBar.UnsavedDot /> Thay đổi chưa lưu
        </Text>
        <Text size="sm" tone="danger" role="alert">
          3 trường cần sửa · Đến trường lỗi tiếp
        </Text>
      </>
    ),
    children: (
      <>
        <Button variant="default">Hủy</Button>
        <Button type="submit">Lưu</Button>
      </>
    ),
  } as never,
  play: async ({ canvasElement }) => {
    const bar = within(canvasElement).getByTestId('bar')
    expect(bar.scrollWidth).toBeLessThanOrEqual(bar.clientWidth)
    expect(within(bar).getByRole('button', { name: 'Lưu' }).getBoundingClientRect().right).toBeLessThanOrEqual(bar.getBoundingClientRect().right)
  },
}

export const Sticky: Story = {
  render: () => (
    <div data-testid="scrollbox" style={{ height: 300, overflow: 'auto' }} tabIndex={0}>
      <div style={{ height: 2000 }}>Nội dung biểu mẫu…</div>
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
