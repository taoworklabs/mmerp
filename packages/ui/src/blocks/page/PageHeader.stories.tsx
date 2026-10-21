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
}

export default meta

type Story = StoryObj<typeof PageHeader>

export const List: Story = {
  args: {
    title: 'Customers',
    description: '128 customers',
    actions: (
      <>
        <Button variant="default">Export</Button>
        <Button>New customer</Button>
      </>
    ),
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
          breadcrumbsLabel="Breadcrumb"
          breadcrumbs={[{ label: 'Customers', render: <a href="#" /> }]}
          back={
            <PageHeader.BackButton
              label="Back to customers"
              onClick={() => {
                /* history navigation is the caller's */
              }}
            />
          }
          leading={<Avatar alt="Nguyễn Văn A">NA</Avatar>}
          status={<Badge color="success">Active</Badge>}
          description="customer@example.com"
          summary={
            <Text size="sm" tone="dimmed">
              Summary placeholder
            </Text>
          }
          tabs={
            <Tabs.List>
              <Tabs.Tab value="overview">Overview</Tabs.Tab>
              <Tabs.Tab value="orders">Orders</Tabs.Tab>
            </Tabs.List>
          }
        />
      </Tabs>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const nav = canvas.getByRole('navigation', { name: 'Breadcrumb' })
    const items = nav.querySelectorAll('li')
    expect(items[items.length - 1]).toHaveAttribute('aria-current', 'page')

    const back = canvas.getByRole('button', { name: 'Back to customers' })
    expect(back).toBeInTheDocument()

    // A crumb reads like dimmed text, not a default browser link: no underline until hovered,
    // the dimmed token's colour rather than the primitive's own brand blue.
    const crumb = canvas.getByRole('link', { name: 'Customers' })
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
    title: 'New customer',
    breadcrumbsLabel: 'Breadcrumb',
    breadcrumbs: [{ label: 'Customers', render: <a href="#" /> }],
  },
  play: async ({ canvasElement }) => {
    // The trail stops at the parent list; the page itself isn't shown yet, so nothing is "current".
    expect(canvasElement.querySelectorAll('[aria-current]')).toHaveLength(0)
  },
}

export const LongTitle: Story = {
  args: {
    title: 'x'.repeat(200),
    current: true,
    breadcrumbsLabel: 'Breadcrumb',
    breadcrumbs: [{ label: 'Customers', render: <a href="#" /> }],
    style: { width: 320 },
  },
  play: async ({ canvasElement }) => {
    const header = canvasElement.firstElementChild as HTMLElement
    const title = within(canvasElement).getByRole('heading', { level: 1 })
    expect(header.scrollWidth).toBeLessThanOrEqual(header.clientWidth)
    expect(title.scrollWidth).toBeGreaterThan(title.clientWidth)

    // The current crumb repeats the same long title; it must truncate to one line rather than
    // wrapping or widening the breadcrumb row.
    const nav = within(canvasElement).getByRole('navigation', { name: 'Breadcrumb' })
    // The li itself never overflows (its content is clipped inside); the truncating span does.
    const crumb = nav.querySelector('li[aria-current="page"] span') as HTMLElement
    expect(crumb.scrollWidth).toBeGreaterThan(crumb.clientWidth)
    expect(crumb.getBoundingClientRect().height).toBeLessThanOrEqual(24)
  },
}
