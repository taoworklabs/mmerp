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
  args: { children: 'Văn bản mặc định' },
}

export const SizeXs: Story = {
  args: { size: 'xs', children: 'Chữ cỡ rất nhỏ' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Chữ cỡ rất nhỏ')
    expect(getComputedStyle(text).fontSize).toBe('12px')
  },
}

export const SizeXl: Story = {
  args: { size: 'xl', children: 'Chữ cỡ rất lớn' },
}

export const ToneDimmed: Story = {
  args: { tone: 'dimmed', children: 'Chữ mờ' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Chữ mờ')
    // Compare against the token variable itself rather than a hardcoded colour literal.
    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(text).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}

export const ToneDanger: Story = {
  args: { tone: 'danger', children: 'Chữ cảnh báo lỗi' },
}

export const ToneSuccess: Story = {
  args: { tone: 'success', children: 'Chữ thành công' },
}

export const ToneWarning: Story = {
  args: { tone: 'warning', children: 'Chữ cảnh báo' },
}

export const TonePrimary: Story = {
  args: { tone: 'primary', children: 'Chữ màu chính' },
}

export const Weight: Story = {
  args: { weight: 650, children: 'Chữ đậm' },
}

export const Align: Story = {
  args: { align: 'end', children: 'Chữ căn phải', style: { display: 'block', width: 240 } },
}

const longText = 'Biên bản bàn giao tài sản, công cụ dụng cụ và hồ sơ chứng từ kế toán khi nhân viên nghỉ việc tại chi nhánh Đà Nẵng '.repeat(5).trim()

export const Truncate: Story = {
  args: {
    truncate: true,
    style: { width: 200 },
    children: longText,
  },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText(longText)
    expect(getComputedStyle(text).textOverflow).toBe('ellipsis')
    expect(text.scrollWidth).toBeGreaterThan(text.clientWidth)
    expect(text.getBoundingClientRect().width).toBe(200)
  },
}

export const LineClamp: Story = {
  args: {
    lineClamp: 2,
    style: { width: 200 },
    children: 'Đoạn văn dài xuống dòng qua nhiều dòng, rồi bị cắt sau dòng thứ hai để phần còn lại được ẩn khỏi tầm nhìn của người đọc.',
  },
}

export const Render: Story = {
  args: { render: <span />, children: 'Hiển thị dưới dạng span' },
  play: async ({ canvasElement }) => {
    const text = within(canvasElement).getByText('Hiển thị dưới dạng span')
    expect(text.tagName).toBe('SPAN')
  },
}

export const InheritSize: Story = {
  args: { render: <span />, size: 'inherit', weight: 600, children: 'Theo cỡ chữ của đoạn quanh nó' },
}
