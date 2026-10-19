import { Button, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { PersonAvatar } from '@/shared/ui/avatar'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { Page } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { Case, Demo } from './Demo'

const crumbs = [
  { label: 'Nhân sự', to: '/dev/ui/templates/page' },
  { label: 'Nhân viên', to: '/dev/ui/templates/page' },
]

const facts: [string, ReactNode][] = [
  ['Quản lý trực tiếp', 'Trần Thị Bình'],
  ['Thâm niên', '4 năm 2 tháng'],
  ['Loại hợp đồng', 'Không xác định thời hạn'],
  ['Phép còn lại', '8,5 ngày'],
]

// Every header frames the same stand-in content, so the cases differ only in the header.
function Body() {
  const { t } = useTranslation()
  return (
    <Text size="sm" c="dimmed">
      {t('shared.devui.page_content')}
    </Text>
  )
}

export function Headers() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.headers')} description={t('shared.devui.headers_hint')}>
      <Case label={t('shared.devui.case_header_list')}>
        <Page title="Nhân viên" breadcrumbs={crumbs.slice(0, 1)} description={t('shared.devui.description')} actions={<Button>{t('shared.devui.primary')}</Button>}>
          <Body />
        </Page>
      </Case>
      <Case label={t('shared.devui.case_header_record')}>
        <Page
          title="Nguyễn Văn An"
          breadcrumbs={crumbs}
          current
          leading={<PersonAvatar name="Nguyễn Văn An" size="lg" />}
          status={<StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>}
          description="NV0019 · Kế toán"
          summary={<KeyFacts facts={facts} />}
          actions={<Button variant="default">{t('shared.devui.secondary')}</Button>}
        >
          <Body />
        </Page>
      </Case>
      <Case label={t('shared.devui.case_header_back')}>
        <Page title="NP-2026-00001" breadcrumbs={crumbs} withBack actions={<Button>{t('shared.devui.primary')}</Button>}>
          <Body />
        </Page>
      </Case>
    </Demo>
  )
}

// A leave request as an inbox or side panel previews it; the note is left out when empty.
