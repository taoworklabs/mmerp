import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, waitFor, within } from 'storybook/test'
import { page } from 'vitest/browser'
import { AppShell } from '../../primitives/layout/AppShell.tsx'
import { Page } from './Page.tsx'
import { PageHeader } from './PageHeader.tsx'

const meta: Meta<typeof Page> = {
  title: 'Blocks/Page',
  component: Page,
}

export default meta

type Story = StoryObj<typeof Page>

export const Full: Story = {
  args: {
    header: <PageHeader title="Customers" />,
    children: <div>Page content</div>,
  },
}

export const Centered: Story = {
  args: {
    variant: 'centered',
    header: <PageHeader title="Edit profile" />,
    children: <div>Form content</div>,
  },
}

export const Detail: Story = {
  args: {
    variant: 'detail',
    header: <PageHeader title="Sales order SO-0001" />,
    children: <div>Detail content</div>,
  },
}

function FillDemo() {
  return (
    <AppShell header={<div>App header</div>}>
      <Page variant="fill" header={<PageHeader title="Orders" />}>
        <Page.Content data-testid="content">
          {Array.from({ length: 60 }, (_, index) => (
            <div key={index}>Row {index + 1}</div>
          ))}
          <div>Footer line</div>
        </Page.Content>
      </Page>
    </AppShell>
  )
}

export const Fill: Story = {
  render: () => <FillDemo />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await page.viewport(900, 500)

    const title = canvas.getByRole('heading', { name: 'Orders' })
    const topBefore = title.getBoundingClientRect().top

    const content = canvas.getByTestId('content') as HTMLElement
    content.scrollTop = content.scrollHeight

    await waitFor(() => expect(canvas.getByText('Footer line').getBoundingClientRect().bottom).toBeLessThanOrEqual(500.5))
    expect(title.getBoundingClientRect().top).toBe(topBefore)
  },
}
