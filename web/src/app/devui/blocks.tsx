import { Button, Text } from '@mantine/core'
import { IconFileDescription, IconPencil, IconUsersGroup } from '@tabler/icons-react'
import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ActivityFeed, ApprovalPanel, DocumentPreview, useAttachmentSection, type DocStatus } from '@/shared/document'
import { sample, SampleDocument } from '@/shared/document/samples'
import { PersonAvatar } from '@/shared/ui/avatar'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel, Form, FormActions, FormFields, TextField } from '@/shared/ui/form'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { Page, Panel, RecordColumns, SeeAll } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
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

const values: [string, ReactNode][] = [
  ['Phòng ban', 'Kế toán'],
  ['Ngày vào làm', '05/08/2022'],
  ['Email', 'an.nguyen@demo.vn'],
  ['Địa chỉ', '4 phố số 1, Đống Đa, Hà Nội'],
]

export function Cards() {
  const { t } = useTranslation()
  const form = useForm({ defaultValues: { department: 'Kế toán', email: '' } })
  return (
    <Demo title={t('shared.devui.cards')} description={t('shared.devui.cards_hint')}>
      <Case label={t('shared.devui.case_key_facts')}>
        <KeyFacts facts={facts} />
      </Case>
      <Case label={t('shared.devui.case_panel')}>
        <Panel
          title={t('shared.devui.section_company')}
          icon={IconFileDescription}
          actions={
            <Button variant="subtle" leftSection={<IconPencil {...icon.button} />}>
              {t('shared.devui.edit')}
            </Button>
          }
        >
          <FieldList stacked rows={values} />
        </Panel>
      </Case>
      <Case label={t('shared.devui.case_editable_panel')}>
        <EditablePanel
          title={t('shared.devui.section_staff')}
          icon={IconUsersGroup}
          editable
          edit={(close) => (
            <Form form={form} onSubmit={async () => close()}>
              <FormFields>
                <TextField name="department" label="Phòng ban" />
                <TextField name="email" label="Email" />
              </FormFields>
              <FormActions submitLabel={t('shared.devui.save')} onCancel={close} cancelLabel={t('shared.form.cancel')} />
            </Form>
          )}
        >
          {(open) => (
            <FieldList
              stacked
              rows={[
                ['Phòng ban', 'Kế toán'],
                ['Email', <MissingValue key="e" label="Email" onAdd={open} />],
              ]}
            />
          )}
        </EditablePanel>
      </Case>
      <Case label={t('shared.devui.case_record_columns')}>
        <RecordColumns
          main={
            <Panel title={t('shared.devui.section_company')} icon={IconFileDescription}>
              <FieldList stacked rows={values} />
            </Panel>
          }
          side={
            <Panel plain title={t('shared.devui.section_staff')} actions={<SeeAll tab="list" />}>
              <FieldList rows={[['NV001', 'Nguyễn Văn An'], ['NV002', 'Trần Thị Bình']]} />
            </Panel>
          }
        />
      </Case>
    </Demo>
  )
}

export function ActivityDemo() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.activity')} description={t('shared.devui.activity_hint')}>
      <SampleDocument>
        <Case label={t('shared.devui.case_activity_full')}>
          <ActivityFeed docType={sample.type} id={sample.id} />
        </Case>
        <Case label={t('shared.devui.case_activity_recent')}>
          <Panel plain title={t('shared.record.recent_activity')} actions={<SeeAll tab="activity" />}>
            <ActivityFeed docType={sample.type} id={sample.id} recent={5} />
          </Panel>
        </Case>
      </SampleDocument>
    </Demo>
  )
}

function Attachments() {
  const section = useAttachmentSection(sample.type, sample.id)
  return <Panel title={section.title}>{section.content}</Panel>
}

// A leave request as an inbox or side panel previews it; the note is left out when empty.
type Leave = { status: DocStatus; employee: string; period: string; note: string }
const leave = { isPending: false, isError: false, data: { status: 'pending_approval', employee: 'NV001 · Nguyễn Văn An', period: '12/10/2026 – 16/10/2026', note: '' } } as UseQueryResult<Leave>

export function DocumentPanels() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.doc_panels')} description={t('shared.devui.doc_panels_hint')}>
      <SampleDocument>
        <Case label={t('shared.devui.case_approval')}>
          <ApprovalPanel docType={sample.type} id={sample.id} />
        </Case>
        <Case label={t('shared.devui.case_attachments')}>
          <Attachments />
        </Case>
        <Case label={t('shared.devui.case_preview')}>
          <DocumentPreview docType={sample.type} query={leave} rows={(d) => [['Nhân viên', d.employee], ['Thời gian nghỉ', d.period], d.note && ['Ghi chú', d.note]]} />
        </Case>
      </SampleDocument>
    </Demo>
  )
}
