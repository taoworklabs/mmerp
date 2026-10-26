import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconCheck, IconChevronRight } from '@tabler/icons-react'
import { Fragment } from 'react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { icon } from '../../tokens/tokens.ts'
import { Text } from '../typography/Text.tsx'
import { Button, type ButtonColor, type ButtonVariant } from './Button.tsx'

const meta: Meta<typeof Button> = {
  title: 'Input/Button',
  component: Button,
  args: { onClick: fn() },
  // React elements, not values to edit.
  argTypes: { leftSection: { control: false }, rightSection: { control: false } },
}

export default meta

type Story = StoryObj<typeof Button>

const variants: ButtonVariant[] = ['filled', 'light', 'subtle', 'default', 'outline']
const colors: ButtonColor[] = ['primary', 'gray', 'danger', 'success', 'warning']

// Every variant x colour on one canvas, so the palette is compared side by side.
export const Variants: Story = {
  render: (args) => (
    <div style={{ display: 'grid', gridTemplateColumns: `auto repeat(${colors.length}, max-content)`, gap: 12, alignItems: 'center', justifyContent: 'start' }}>
      <span />
      {colors.map((color) => (
        <Text key={color} size="xs" tone="dimmed">
          {color}
        </Text>
      ))}
      {variants.map((variant) => (
        <Fragment key={variant}>
          <Text size="xs" tone="dimmed">
            {variant}
          </Text>
          {colors.map((color) => (
            <Button key={color} {...args} variant={variant} color={color}>
              Lưu nháp
            </Button>
          ))}
        </Fragment>
      ))}
    </div>
  ),
}

export const Xs: Story = {
  args: { size: 'xs', children: 'Xuất Excel' },
}

export const Loading: Story = {
  args: { loading: true, children: 'Đang lưu' },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Đang lưu' })
    expect(button).toHaveAttribute('aria-busy', 'true')
    expect(button).toBeDisabled()
  },
}

export const Disabled: Story = {
  args: { disabled: true, children: 'Gửi duyệt' },
}

export const WithSections: Story = {
  args: {
    children: 'Tiếp tục',
    leftSection: <IconCheck size={icon.button.size} stroke={icon.button.stroke} />,
    rightSection: <IconChevronRight size={icon.button.size} stroke={icon.button.stroke} />,
  },
}

export const FullWidth: Story = {
  args: { fullWidth: true, children: 'Lưu và đóng' },
  render: (args) => (
    <div style={{ width: 320 }}>
      <Button {...args} />
    </div>
  ),
}

export const Default: Story = {
  args: { children: 'Tạo đơn hàng' },
  play: async ({ canvasElement, args }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Tạo đơn hàng' })
    button.focus()
    await userEvent.keyboard(' ')
    await expect(args.onClick).toHaveBeenCalledTimes(1)
  },
}

export const Grouped: Story = {
  render: () => (
    <Button.Group>
      <Button>Tuần này</Button>
      <Button>Tháng này</Button>
    </Button.Group>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const first = canvas.getByRole('button', { name: 'Tuần này' })
    const second = canvas.getByRole('button', { name: 'Tháng này' })
    expect(second.getBoundingClientRect().left).toBeCloseTo(first.getBoundingClientRect().right - 1, 0)

    // The neighbour's -1px overlap would otherwise paint over the focused button's own ring.
    await userEvent.tab()
    expect(first).toHaveFocus()
    expect(getComputedStyle(first).zIndex).toBe('1')
  },
}
