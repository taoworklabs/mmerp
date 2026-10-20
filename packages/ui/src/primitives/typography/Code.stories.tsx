import type { Meta, StoryObj } from '@storybook/react-vite'
import { Code } from './Code.tsx'

const meta: Meta<typeof Code> = {
  title: 'Typography/Code',
  component: Code,
}

export default meta

type Story = StoryObj<typeof Code>

export const Default: Story = {
  args: { children: 'const x = 1' },
}

export const Block: Story = {
  args: { block: true, children: 'function add(a, b) {\n  return a + b\n}' },
}
