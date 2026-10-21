import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconCheck } from '@tabler/icons-react'
import { icon } from '../../tokens/tokens.ts'
import { ThemeIcon } from './ThemeIcon.tsx'

const meta: Meta<typeof ThemeIcon> = {
  title: 'Data display/ThemeIcon',
  component: ThemeIcon,
}

export default meta

type Story = StoryObj<typeof ThemeIcon>

export const Light: Story = {
  args: { variant: 'light', children: <IconCheck size={icon.text.size} stroke={icon.text.stroke} /> },
}

export const Filled: Story = {
  args: { variant: 'filled', children: <IconCheck size={icon.text.size} stroke={icon.text.stroke} /> },
}
