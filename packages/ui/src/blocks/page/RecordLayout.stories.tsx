import type { Meta, StoryObj } from '@storybook/react-vite'
import { Button } from '../../primitives/input/Button.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { Panel } from './Panel.tsx'
import { ActionBar, RecordColumns } from './RecordLayout.tsx'

const meta: Meta<typeof RecordColumns> = {
  title: 'Blocks/RecordLayout',
  component: RecordColumns,
}

export default meta

type Story = StoryObj<typeof RecordColumns>

const main = (
  <Panel title="Details">
    <Text size="sm">Main column content</Text>
  </Panel>
)

const side = (
  <>
    <Panel title="Status" plain>
      <Text size="sm">Pending approval</Text>
    </Panel>
    <Panel title="Attachments" plain>
      <Text size="sm">No files yet.</Text>
    </Panel>
  </>
)

export const Columns: Story = {
  args: { main, side },
}

export const DocumentRatio: Story = {
  args: { main, side, ratio: 'document' },
}

export const WithActionBar: Story = {
  render: () => (
    <div>
      <RecordColumns main={main} side={side} />
      <ActionBar>
        <Button variant="default">Cancel</Button>
        <Button>Save</Button>
      </ActionBar>
    </div>
  ),
}
