import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { ScrollArea } from './ScrollArea.tsx'

const meta: Meta<typeof ScrollArea> = {
  title: 'Layout/ScrollArea',
  component: ScrollArea,
}

export default meta

type Story = StoryObj<typeof ScrollArea>

let viewportEl: HTMLDivElement | null = null

export const Overflow: Story = {
  render: () => (
    <ScrollArea
      viewportRef={(el) => {
        viewportEl = el
      }}
      style={{ height: 160, width: 280 }}
    >
      {Array.from({ length: 40 }, (_, i) => (
        <p key={i}>Dòng {i + 1}</p>
      ))}
    </ScrollArea>
  ),
  play: async () => {
    expect(viewportEl).toBeTruthy()
    expect(viewportEl!.scrollHeight).toBeGreaterThan(viewportEl!.clientHeight)
  },
}
