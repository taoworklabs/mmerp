import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { ChevronRightIcon } from '../../internal/icons.tsx'
import { Anchor } from '../typography/Anchor.tsx'
import { Breadcrumbs } from './Breadcrumbs.tsx'

const meta: Meta<typeof Breadcrumbs> = {
  title: 'Navigation/Breadcrumbs',
  component: Breadcrumbs,
  args: { 'aria-label': 'Đường dẫn' },
}

export default meta

type Story = StoryObj<typeof Breadcrumbs>

export const Default: Story = {
  render: (args) => (
    <Breadcrumbs {...args}>
      <Anchor href="#">Trang chủ</Anchor>
      <Anchor href="#">Khách hàng</Anchor>
      <span>Nguyễn Văn A</span>
    </Breadcrumbs>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const nav = canvas.getByRole('navigation', { name: 'Đường dẫn' })
    expect(nav.querySelector('ol')).not.toBeNull()

    const current = canvas.getByText('Nguyễn Văn A').closest('li')
    expect(current).toHaveAttribute('aria-current', 'page')
    expect(canvas.getByText('Trang chủ').closest('li')).not.toHaveAttribute('aria-current')

    expect(canvas.getAllByText('/')).toHaveLength(2)
  },
}

// An icon separator, as PageHeader passes: it sits on the text's centre line, not its baseline.
export const CustomSeparator: Story = {
  render: (args) => (
    <Breadcrumbs {...args} separator={<ChevronRightIcon />}>
      <Anchor href="#">Trang chủ</Anchor>
      <Anchor href="#">Khách hàng</Anchor>
      <span>Nguyễn Văn A</span>
    </Breadcrumbs>
  ),
  play: async ({ canvasElement }) => {
    const middle = (el: Element) => {
      const r = el.getBoundingClientRect()
      return r.top + r.height / 2
    }
    const text = within(canvasElement).getByText('Trang chủ')
    for (const icon of canvasElement.querySelectorAll('svg')) {
      expect(Math.abs(middle(icon) - middle(text))).toBeLessThanOrEqual(0.5)
    }
  },
}

export const NotCurrentPage: Story = {
  // A trail that stops short of the page itself (e.g. a new-record form): the last link is a
  // parent, not the current page, so nothing in the trail should read as "current".
  render: (args) => (
    <Breadcrumbs {...args} currentLast={false}>
      <Anchor href="#">Trang chủ</Anchor>
      <Anchor href="#">Khách hàng</Anchor>
    </Breadcrumbs>
  ),
  play: async ({ canvasElement }) => {
    expect(canvasElement.querySelectorAll('[aria-current]')).toHaveLength(0)
  },
}
