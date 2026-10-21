import type { Meta, StoryObj } from '@storybook/react-vite'
import { KeyFacts } from './KeyFacts.tsx'

const meta: Meta<typeof KeyFacts> = {
  title: 'Blocks/KeyFacts',
  component: KeyFacts,
}

export default meta

type Story = StoryObj<typeof KeyFacts>

export const Default: Story = {
  args: {
    facts: [
      ['Manager', 'Trần Thị B'],
      ['Tenure', '2 years'],
      ['Leave left', '8 days'],
      ['Overtime', '12 hours'],
    ],
  },
}
