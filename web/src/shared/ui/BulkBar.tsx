import { Button, Group, Text } from '@mantine/core'
import { IconX } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { icon } from './theme'

// BulkBar takes the filter bar's place while rows are selected: how many, the actions that apply
// to all of them (each starts a job), and a way to clear the selection.
export function BulkBar({ count, onClear, children }: { count: number; onClear: () => void; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <Group gap="xs" role="toolbar" aria-label={t('shared.bulk.label')}>
      <Text size="sm" fw={600}>
        {t('shared.bulk.selected', { count, n: formatNumber(count) })}
      </Text>
      {children}
      <Button variant="subtle" color="gray" leftSection={<IconX {...icon.button} />} onClick={onClear}>
        {t('shared.bulk.clear')}
      </Button>
    </Group>
  )
}
