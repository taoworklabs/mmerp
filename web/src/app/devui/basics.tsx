import { Badge, Button, Group, Menu, Text } from '@mantine/core'
import { IconDownload, IconPencil, IconTrash } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { DocumentStatus, type DocStatus } from '@/shared/document'
import { MenuAction } from '@/shared/ui/MenuAction'
import { MoreActions } from '@/shared/ui/MoreActions'
import { Notice } from '@/shared/ui/Notice'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { Case, Demo } from './Demo'

export function Menus() {
  const { t } = useTranslation()
  const items = (
    <>
      <MenuAction icon={IconDownload} onClick={() => {}}>
        {t('shared.devui.export')}
      </MenuAction>
      <MenuAction icon={IconPencil} onClick={() => {}}>
        {t('shared.devui.edit')}
      </MenuAction>
      <Menu.Divider />
      <MenuAction icon={IconTrash} danger onClick={() => {}}>
        {t('shared.devui.danger')}
      </MenuAction>
    </>
  )
  return (
    <Demo title={t('shared.devui.menus')} description={t('shared.devui.menus_hint')}>
      <Case label={t('shared.devui.case_menu_page')}>
        <Group gap="xs">
          <Button variant="default">{t('shared.devui.secondary')}</Button>
          <MoreActions place="page">{items}</MoreActions>
          <Button>{t('shared.devui.primary')}</Button>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_menu_row')}>
        <Group gap="xs">
          <MoreActions>{items}</MoreActions>
        </Group>
      </Case>
    </Demo>
  )
}

const docStatuses: DocStatus[] = ['draft', 'pending_approval', 'posted', 'cancelled']

export function Badges() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.badges')} description={t('shared.devui.badges_hint')}>
      <Case label={t('shared.devui.case_doc_status')}>
        <Group>
          {docStatuses.map((s) => (
            <DocumentStatus key={s} docType="shared" status={s} />
          ))}
        </Group>
      </Case>
      <Case label={t('shared.devui.case_record_status')}>
        <Group>
          <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
          <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
          <StatusBadge tone="info">{t('shared.devui.status_info')}</StatusBadge>
          <StatusBadge tone="warning">{t('shared.devui.status_warning')}</StatusBadge>
          <StatusBadge tone="negative">{t('shared.devui.status_negative')}</StatusBadge>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_count')}>
        <Group gap="xs">
          <Text size="sm">{t('shared.devui.section')}</Text>
          <Badge color="danger" size="sm" circle aria-label={t('shared.devui.count_label', { count: 2 })}>
            2
          </Badge>
        </Group>
      </Case>
    </Demo>
  )
}

export function Alerts() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.alerts')} description={t('shared.devui.alerts_hint')}>
      {(['info', 'success', 'warning', 'danger', 'discovery'] as const).map((tone) => (
        <Case key={tone} label={t(`shared.devui.case_tone_${tone}`)}>
          <Notice tone={tone}>{t(`shared.devui.alert_${tone}`)}</Notice>
        </Case>
      ))}
    </Demo>
  )
}

