import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconCheck } from '@tabler/icons-react'
import { expect, within } from 'storybook/test'
import { icon } from '../../tokens/tokens.ts'
import { ActionIcon } from './ActionIcon.tsx'

const meta: Meta<typeof ActionIcon> = {
  title: 'Input/ActionIcon',
  component: ActionIcon,
}

export default meta

type Story = StoryObj<typeof ActionIcon>

export const Default: Story = {
  args: { 'aria-label': 'Xác nhận', children: <IconCheck size={icon.button.size} stroke={icon.button.stroke} /> },
}

export const Light: Story = {
  args: { 'aria-label': 'Xác nhận', variant: 'light', children: <IconCheck size={icon.button.size} stroke={icon.button.stroke} /> },
}

export const Xs: Story = {
  args: { 'aria-label': 'Xác nhận', size: 'xs', children: <IconCheck size={icon.text.size} stroke={icon.text.stroke} /> },
}

export const Loading: Story = {
  args: { 'aria-label': 'Xác nhận', loading: true, children: <IconCheck size={icon.button.size} stroke={icon.button.stroke} /> },
  play: async ({ canvasElement }) => {
    const button = within(canvasElement).getByRole('button', { name: 'Xác nhận' })
    expect(button).toHaveAttribute('aria-busy', 'true')
    expect(button).toBeDisabled()
  },
}

export const Disabled: Story = {
  args: { 'aria-label': 'Xác nhận', disabled: true, children: <IconCheck size={icon.button.size} stroke={icon.button.stroke} /> },
}
