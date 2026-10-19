import { Alert } from '@mantine/core'
import { IconAlertCircle, IconAlertTriangle, IconCircleCheck, IconInfoCircle, IconSparkles, type Icon as TablerIcon } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { icon as iconSize } from './theme'

type Tone = 'info' | 'success' | 'warning' | 'danger' | 'discovery'

const icons: Record<Tone, TablerIcon> = {
  info: IconInfoCircle,
  success: IconCircleCheck,
  warning: IconAlertTriangle,
  danger: IconAlertCircle,
  discovery: IconSparkles,
}

// Notice is a message in the page flow: the tone picks the colour and icon, so every screen says
// "warning" the same way. Danger is announced at once; `announce` lets another tone be heard too
// ('alert' for something that just went wrong, 'polite' for progress or a result).
export function Notice({
  tone,
  title,
  icon,
  announce = tone === 'danger' ? 'alert' : undefined,
  children,
}: {
  tone: Tone
  title?: string
  icon?: ReactNode
  announce?: 'alert' | 'polite'
  children?: ReactNode
}) {
  const Icon = icons[tone]
  return (
    <Alert
      color={tone}
      title={title}
      icon={icon ?? <Icon {...iconSize.button} />}
      role={announce === 'alert' ? 'alert' : undefined}
      aria-live={announce === 'polite' ? 'polite' : undefined}
    >
      {children}
    </Alert>
  )
}
