import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { Page } from './Page.tsx'
import { PageHeader } from './PageHeader.tsx'
import { Panel } from './Panel.tsx'
import { ActionBar, RecordColumns } from './RecordLayout.tsx'

const meta: Meta<typeof RecordColumns> = {
  title: 'Blocks/RecordLayout',
  component: RecordColumns,
  // A record's page centres and caps its columns; RecordColumns alone just fills its container.
  decorators: [
    (Story) => (
      <Page variant="detail" header={<PageHeader title="HĐ-2024-0012" />}>
        <Story />
      </Page>
    ),
  ],
  parameters: { layout: 'fullscreen' },
  // React elements, not values to edit.
  argTypes: { main: { control: false }, side: { control: false } },
}

export default meta

type Story = StoryObj<typeof RecordColumns>

const main = (
  <Panel title="Thông tin chung">
    <Text size="sm">Nội dung cột chính</Text>
  </Panel>
)

const side = (
  <>
    <Panel title="Trạng thái" plain>
      <Text size="sm">Chờ duyệt</Text>
    </Panel>
    <Panel title="Tệp đính kèm" plain>
      <Text size="sm">Chưa có tệp nào.</Text>
    </Panel>
  </>
)

export const Columns: Story = {
  args: { main, side },
}

export const DocumentRatio: Story = {
  args: { main, side, ratio: 'document' },
}

// The bar only shows below 64em, where the page header no longer has room for the actions.
export const WithActionBar: Story = {
  globals: { viewport: { value: 'drawer' } },
  render: () => (
    <div>
      <RecordColumns main={main} side={side} />
      <ActionBar>
        <Button variant="default">Hủy</Button>
        <Button>Lưu</Button>
      </ActionBar>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const cancel = canvas.getByRole('button', { name: 'Hủy' }).getBoundingClientRect()
    const save = canvas.getByRole('button', { name: 'Lưu' }).getBoundingClientRect()
    expect(save.left - cancel.right).toBeGreaterThanOrEqual(8)
    expect(save.right).toBeGreaterThan(canvasElement.ownerDocument.documentElement.clientWidth / 2)
  },
}
