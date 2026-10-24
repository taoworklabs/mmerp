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
  args: { href: '#', children: 'Xem chi tiết' },
}

export const Tone: Story = {
  args: { href: '#', tone: 'danger', children: 'Hủy đơn hàng' },
}

export const UnderlineAlways: Story = {
  args: { href: '#', underline: 'always', children: 'Luôn gạch chân' },
}

export const UnderlineNever: Story = {
  args: { href: '#', underline: 'never', children: 'Không gạch chân' },
}

export const RenderAsButton: Story = {
  args: { render: <button type="button" />, children: 'Trông như liên kết, hiển thị là nút' },
  play: async ({ canvasElement }) => {
    const element = within(canvasElement).getByText('Trông như liên kết, hiển thị là nút')
    expect(element.tagName).toBe('BUTTON')
  },
}

export const AsButton: Story = {
  tags: ['!dev', '!autodocs'],
  render: () => (
    <>
      <Anchor render={<button type="button" />}>+ Bổ sung</Anchor>
      <p>Văn bản đối chiếu</p>
    </>
  ),
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByText('+ Bổ sung')
    const reference = within(canvasElement).getByText('Văn bản đối chiếu')
    const buttonStyle = getComputedStyle(button)
    expect(buttonStyle.borderTopWidth).toBe('0px')
    // Checked by channel, not as a literal colour string: this file may not contain colour literals.
    const channels = buttonStyle.backgroundColor.replace(/[^\d.,]/g, '').split(',')
    expect(channels).toHaveLength(4)
    expect(Number(channels[3])).toBe(0)
    expect(buttonStyle.paddingLeft).toBe('0px')
    expect(buttonStyle.fontSize).toBe(getComputedStyle(reference).fontSize)
  },
}
