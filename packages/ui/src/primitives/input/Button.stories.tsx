import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconCheck, IconChevronRight } from '@tabler/icons-react'
import { expect, fn, userEvent, within } from 'storybook/test'
import { icon } from '../../tokens/tokens.ts'
import { Button } from './Button.tsx'

const meta: Meta<typeof Button> = {
  title: 'Input/Button',
  component: Button,
  args: { onClick: fn() },
}

export default meta

type Story = StoryObj<typeof Button>

export const FilledPrimary: Story = {
  args: { color: 'primary', variant: 'filled', children: 'Filled primary' },
}

export const FilledGray: Story = {
  args: { color: 'gray', variant: 'filled', children: 'Filled gray' },
}

export const FilledDanger: Story = {
  args: { color: 'danger', variant: 'filled', children: 'Filled danger' },
}

export const LightPrimary: Story = {
  args: { color: 'primary', variant: 'light', children: 'Light primary' },
}

export const LightGray: Story = {
  args: { color: 'gray', variant: 'light', children: 'Light gray' },
}

export const LightDanger: Story = {
  args: { color: 'danger', variant: 'light', children: 'Light danger' },
}

export const SubtlePrimary: Story = {
  args: { color: 'primary', variant: 'subtle', children: 'Subtle primary' },
}

export const SubtleGray: Story = {
  args: { color: 'gray', variant: 'subtle', children: 'Subtle gray' },
}

export const SubtleDanger: Story = {
  args: { color: 'danger', variant: 'subtle', children: 'Subtle danger' },
}

export const DefaultVariantPrimary: Story = {
  args: { color: 'primary', variant: 'default', children: 'Default primary' },
}

export const DefaultVariantGray: Story = {
  args: { color: 'gray', variant: 'default', children: 'Default gray' },
}

export const DefaultVariantDanger: Story = {
  args: { color: 'danger', variant: 'default', children: 'Default danger' },
}

export const OutlinePrimary: Story = {
  args: { color: 'primary', variant: 'outline', children: 'Outline primary' },
}

export const OutlineGray: Story = {
  args: { color: 'gray', variant: 'outline', children: 'Outline gray' },
}

export const OutlineDanger: Story = {
  args: { color: 'danger', variant: 'outline', children: 'Outline danger' },
}

export const Xs: Story = {
  args: { size: 'xs', children: 'Extra small' },
}

export const Loading: Story = {
  args: { loading: true, children: 'Saving' },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Saving' })
    expect(button).toHaveAttribute('aria-busy', 'true')
    expect(button).toBeDisabled()
  },
}

export const Disabled: Story = {
  args: { disabled: true, children: 'Disabled' },
}

export const WithSections: Story = {
  args: {
    children: 'Continue',
    leftSection: <IconCheck size={icon.button.size} stroke={icon.button.stroke} />,
    rightSection: <IconChevronRight size={icon.button.size} stroke={icon.button.stroke} />,
  },
}

export const FullWidth: Story = {
  args: { fullWidth: true, children: 'Full width' },
  render: (args) => (
    <div style={{ width: 320 }}>
      <Button {...args} />
    </div>
  ),
}

export const Default: Story = {
  args: { children: 'Click me' },
  play: async ({ canvasElement, args }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Click me' })
    button.focus()
    await userEvent.keyboard(' ')
    await expect(args.onClick).toHaveBeenCalledTimes(1)
  },
}
