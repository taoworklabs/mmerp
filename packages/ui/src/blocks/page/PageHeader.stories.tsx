import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Avatar } from '../../primitives/data-display/Avatar.tsx'
import { Badge } from '../../primitives/feedback/Badge.tsx'
import { Button } from '../../primitives/input/Button.tsx'
import { Tabs } from '../../primitives/navigation/Tabs.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { PageHeader } from './PageHeader.tsx'

const meta: Meta<typeof PageHeader> = {
  title: 'Blocks/PageHeader',
  component: PageHeader,
  // React elements, not values to edit.
  argTypes: { actions: { control: false } },
}

export default meta

type Story = StoryObj<typeof PageHeader>

export const List: Story = {
  args: {
    title: 'Khách hàng',
    description: '128 khách hàng',
    actions: (
      <>
        <Button variant="default">Xuất file</Button>
        <Button>Thêm khách hàng</Button>
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const secondary = canvas.getByRole('button', { name: 'Xuất file' }).getBoundingClientRect()
    const primary = canvas.getByRole('button', { name: 'Thêm khách hàng' }).getBoundingClientRect()
    // The buttons sit together at the end, not spread across the row.
    expect(primary.left - secondary.right).toBeLessThanOrEqual(16)
  },
}

export const Record: Story = {
  render: () => (
    // Padded like it always is in the app (inside Page's own frame): flush against the
    // viewport's own corner, headless Chromium's default pointer position reports a false
    // :hover there, which would make the crumb assertion below flaky.
    <div style={{ padding: 24 }}>
      <Tabs defaultValue="overview">
        <PageHeader
          title="Nguyễn Văn A"
          current
          breadcrumbsLabel="Đường dẫn"
          breadcrumbs={[{ label: 'Khách hàng', render: <a href="#" /> }]}
          back={
            <PageHeader.BackButton
              label="Quay lại danh sách khách hàng"
              onClick={() => {
                /* history navigation is the caller's */
              }}
            />
          }
          leading={<Avatar alt="Nguyễn Văn A">NA</Avatar>}
          status={<Badge color="success">Đang giao dịch</Badge>}
          description="customer@example.com"
          summary={
            <Text size="sm" tone="dimmed">
              Khách hàng từ 03/2021 · Công nợ 12.500.000 ₫
            </Text>
          }
          tabs={
            <Tabs.List>
              <Tabs.Tab value="overview">Tổng quan</Tabs.Tab>
              <Tabs.Tab value="orders">Đơn bán hàng</Tabs.Tab>
            </Tabs.List>
          }
        />
      </Tabs>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const nav = canvas.getByRole('navigation', { name: 'Đường dẫn' })
    const items = nav.querySelectorAll('li')
    expect(items[items.length - 1]).toHaveAttribute('aria-current', 'page')

    const back = canvas.getByRole('button', { name: 'Quay lại danh sách khách hàng' })
    expect(back).toBeInTheDocument()

    // A crumb reads like dimmed text, not a default browser link: no underline until hovered,
    // the dimmed token's colour rather than the primitive's own brand blue.
    const crumb = canvas.getByRole('link', { name: 'Khách hàng' })
    expect(getComputedStyle(crumb).textDecorationLine).toBe('none')

    const probe = document.createElement('div')
    probe.style.color = 'var(--mmerp-color-dimmed)'
    canvasElement.appendChild(probe)
    expect(getComputedStyle(crumb).color).toBe(getComputedStyle(probe).color)
    probe.remove()
  },
}

export const NewRecord: Story = {
  args: {
    title: 'Thêm khách hàng',
    breadcrumbsLabel: 'Đường dẫn',
    breadcrumbs: [{ label: 'Khách hàng', render: <a href="#" /> }],
  },
  play: async ({ canvasElement }) => {
    // The trail stops at the parent list; the page itself isn't shown yet, so nothing is "current".
    expect(canvasElement.querySelectorAll('[aria-current]')).toHaveLength(0)
  },
}

export const LongTitle: Story = {
  args: {
    title: 'Công ty TNHH Thương mại Dịch vụ Xuất nhập khẩu Nguyễn Hoàng Phát – Chi nhánh Thành phố Hồ Chí Minh, Quận Bình Thạnh, Phường 25 (khách hàng thân thiết từ năm 2018)',
    current: true,
    breadcrumbsLabel: 'Đường dẫn',
    breadcrumbs: [{ label: 'Khách hàng', render: <a href="#" /> }],
    style: { width: 320 },
  },
  play: async ({ canvasElement }) => {
    const header = canvasElement.firstElementChild as HTMLElement
    const title = within(canvasElement).getByRole('heading', { level: 1 })
    expect(header.scrollWidth).toBeLessThanOrEqual(header.clientWidth)
    expect(title.scrollWidth).toBeGreaterThan(title.clientWidth)

    // The current crumb repeats the same long title; it must truncate to one line rather than
    // wrapping or widening the breadcrumb row.
    const nav = within(canvasElement).getByRole('navigation', { name: 'Đường dẫn' })
    // The li itself never overflows (its content is clipped inside); the truncating span does.
    const crumb = nav.querySelector('li[aria-current="page"] span') as HTMLElement
    expect(crumb.scrollWidth).toBeGreaterThan(crumb.clientWidth)
    expect(crumb.getBoundingClientRect().height).toBeLessThanOrEqual(24)
  },
}
