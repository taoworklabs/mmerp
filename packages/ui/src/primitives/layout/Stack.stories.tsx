import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'
import { Button } from '../input/Button.tsx'
import { Text } from '../typography/Text.tsx'
import { Group } from './Group.tsx'
import { Stack } from './Stack.tsx'

const meta: Meta<typeof Stack> = {
  title: 'Layout/Stack',
  component: Stack,
  // React elements, not values to edit.
  argTypes: { children: { control: false } },
}

export default meta

type Story = StoryObj<typeof Stack>

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

export const Gap: Story = {
  args: {
    gap: 'md',
    children: (
      <>
        <Item label="Mục thứ nhất" />
        <Item label="Mục thứ hai" />
      </>
    ),
  },
  play: async ({ canvasElement }) => {
    const stack = within(canvasElement).getByText('Mục thứ nhất').parentElement
    expect(stack).not.toBeNull()
    expect(getComputedStyle(stack as HTMLElement).rowGap).toBe('16px')
  },
}

export const Align: Story = {
  args: {
    align: 'center',
    children: (
      <>
        <Item label="Hẹp" />
        <Item label="Một mục rộng hơn nhiều" />
      </>
    ),
  },
}

export const Justify: Story = {
  render: () => (
    <Stack justify="space-between" style={{ height: 120 }}>
      <Item label="Trên cùng" />
      <Item label="Dưới cùng" />
    </Stack>
  ),
}

export const FillTruncates: Story = {
  render: () => (
    <div style={{ width: 300 }}>
      <Group wrap={false} data-testid="outer">
        <Stack fill>
          <Text truncate>{'Phiếu đề nghị thanh toán tạm ứng công tác phí cho chuyến khảo sát thị trường miền Trung của phòng Kinh doanh quý IV năm 2026'}</Text>
        </Stack>
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
    const stack = within(canvasElement).getByText('Mục thứ nhất').parentElement as HTMLElement
    expect(stack.tagName).toBe('UL')
    expect(getComputedStyle(stack).rowGap).toBe('16px')
  },
}
