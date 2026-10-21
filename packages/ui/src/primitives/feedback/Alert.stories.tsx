import type { Meta, StoryObj } from '@storybook/react-vite'
import { IconAlertTriangle, IconCircleCheck, IconInfoCircle } from '@tabler/icons-react'
import { expect, within } from 'storybook/test'
import { icon } from '../../tokens/tokens.ts'
import { Alert } from './Alert.tsx'

const meta: Meta<typeof Alert> = {
  title: 'Feedback/Alert',
  component: Alert,
}

export default meta

type Story = StoryObj<typeof Alert>

export const Info: Story = {
  args: {
    color: 'info',
    title: 'Heads up',
    icon: <IconInfoCircle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Informational message.',
  },
}

export const Success: Story = {
  args: {
    color: 'success',
    title: 'Saved',
    icon: <IconCircleCheck size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Your changes were saved.',
  },
}

export const Warning: Story = {
  args: {
    color: 'warning',
    title: 'Check this',
    icon: <IconAlertTriangle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'Something needs attention.',
  },
}

export const Danger: Story = {
  args: {
    color: 'danger',
    title: 'Failed',
    icon: <IconAlertTriangle size={icon.text.size} stroke={icon.text.stroke} />,
    children: 'The request failed.',
  },
  play: async ({ canvasElement }) => {
    const alert = within(canvasElement).getByRole('alert')
    expect(alert).toBeInTheDocument()
  },
}
