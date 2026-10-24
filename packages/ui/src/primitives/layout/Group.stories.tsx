import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Text } from '../typography/Text.tsx'
import { Group } from './Group.tsx'

const meta: Meta<typeof Group> = {
  title: 'Layout/Group',
  component: Group,
  // React elements, not values to edit.
  argTypes: { children: { control: false } },
}

export default meta

type Story = StoryObj<typeof Group>

function Item({ label }: { label: string }) {
  return <div>{label}</div>
}

export const Default: Story = {
  args: {
    children: (
      <>
        <Item label="Mục thứ nhất" />
        <Item label="Mục thứ hai" />
      </>
    ),
  },
}

export const Align: Story = {
  args: {
    align: 'start',
    children: (
      <>
        <Item label="Ngắn" />
        <Item label="Mục cao hơn" />
      </>
    ),
  },
}

export const Justify: Story = {
  render: () => (
    <Group justify="space-between" style={{ width: 240 }}>
      <Item label="Trái" />
      <Item label="Phải" />
    </Group>
  ),
}

export const Wrap: Story = {
  args: {
    wrap: false,
    children: (
      <>
        <Item label="Mục thứ nhất" />
        <Item label="Mục thứ hai" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const group = within(canvasElement).getByText('Mục thứ nhất').parentElement
    expect(group).not.toBeNull()
    expect(getComputedStyle(group as HTMLElement).flexWrap).toBe('nowrap')
  },
}

export const Grow: Story = {
  render: () => (
    <Group grow style={{ width: 240 }}>
      <Item label="Mục thứ nhất" />
      <Item label="Mục thứ hai" />
    </Group>
  ),
}

export const ShrinkTruncates: Story = {
  render: () => (
    <div style={{ width: 300 }}>
      <Group wrap={false} data-testid="outer">
        <Group shrink>
          <Text truncate>{'Phiếu đề nghị thanh toán tạm ứng công tác phí cho chuyến khảo sát thị trường miền Trung của phòng Kinh doanh quý IV năm 2026'}</Text>
        </Group>
        <Button>Lưu</Button>
      </Group>
    </div>
  ),
  play: async ({ canvasElement }) => {
    const outerGroup = within(canvasElement).getByTestId('outer')
    const text = within(canvasElement).getByText('Phiếu đề nghị thanh toán tạm ứng công tác phí cho chuyến khảo sát thị trường miền Trung của phòng Kinh doanh quý IV năm 2026')
    expect(outerGroup.scrollWidth).toBeLessThanOrEqual(300)
    expect(text.scrollWidth).toBeGreaterThan(text.clientWidth)
  },
}

export const Render: Story = {
  args: {
    render: <ul />,
    gap: 'md',
    children: (
      <>
        <li>Mục thứ nhất</li>
        <li>Mục thứ hai</li>
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const group = within(canvasElement).getByText('Mục thứ nhất').parentElement as HTMLElement
    expect(group.tagName).toBe('UL')
    expect(getComputedStyle(group).columnGap).toBe('16px')
  },
}
