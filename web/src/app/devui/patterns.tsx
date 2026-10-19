import { Button, Group, SegmentedControl, Stack, Text } from '@mantine/core'
import { IconCircleDot, IconDownload, IconTrash, IconUsersGroup } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { ApprovalPanel, ActivityFeed, DocumentStatus } from '@/shared/document'
import { sample, SampleDocument } from '@/shared/document/samples'
import { formatNumber } from '@/shared/i18n'
import { BulkBar } from '@/shared/ui/BulkBar'
import { type Column, DataTable } from '@/shared/ui/DataTable'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { EditablePanel, FilterChip, Form, FormActions, FormFields, FormSection, MoneyField, SearchInput, TextField } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { MenuAction } from '@/shared/ui/MenuAction'
import { MoreActions } from '@/shared/ui/MoreActions'
import { Notice } from '@/shared/ui/Notice'
import { FormModal, Panel, RecordColumns } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { EmptyState } from '@/shared/ui/states'
import { useAction } from '@/shared/ui/useAction'
import { fakeQuery, type QueryCase, queryStates } from './actions'
import { Case, rows, type Row } from './Demo'
import { DocPage, DoDont } from './DocPage'

function useColumns(): Column<Row>[] {
  const { t } = useTranslation()
  return [
    { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', render: (r) => r.code },
    { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
    { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
  ]
}

// FilterSearchPattern filters in the browser; a real list sends the filters to the server through the URL.
export function FilterSearchPattern() {
  const { t } = useTranslation()
  const [q, setQ] = useState('')
  const [band, setBand] = useState<string | null>(null)
  const bands = [
    { value: 'high', label: t('shared.devui.band_high') },
    { value: 'low', label: t('shared.devui.band_low') },
  ]
  const found = rows.filter((r) => (!q || `${r.code} ${r.name}`.toLowerCase().includes(q.toLowerCase())) && (!band || (band === 'high') === r.amount >= 10_000_000))
  const columns = useColumns()
  return (
    <DocPage
      path="patterns/filter-search"
      example={
        <Case>
          <Group gap="xs">
            <SearchInput label={t('shared.devui.name')} value={q} onSearch={setQ} />
            <FilterChip label={t('shared.devui.amount')} icon={IconCircleDot} data={bands} value={band} onChange={setBand} />
          </Group>
          {found.length > 0 ? (
            <DataTable label={t('shared.devui.table')} rows={found} rowKey={(r) => r.id} columns={columns} />
          ) : (
            <EmptyState
              title={t('shared.devui.no_results')}
              action={
                <Button
                  variant="default"
                  onClick={() => {
                    setQ('')
                    setBand(null)
                  }}
                >
                  {t('shared.list.clear_filters')}
                </Button>
              }
            />
          )}
        </Case>
      }
      doDont={<DoDont good={<Text size="sm">{t('shared.devui.do_filter_url')}</Text>} bad={<Text size="sm">{t('shared.devui.dont_filter_client')}</Text>} />}
    />
  )
}

export function CreateEditPattern() {
  const { t } = useTranslation()
  const [opened, setOpened] = useState(false)
  const create = useForm({ defaultValues: { name: '', salary: undefined } })
  const edit = useForm({ defaultValues: { department: 'Kế toán', email: '' } })
  return (
    <DocPage
      path="patterns/create-edit"
      example={
        <>
          <Case label={t('shared.devui.case_create_modal')}>
            <Group>
              <Button onClick={() => setOpened(true)}>{t('shared.devui.add')}</Button>
            </Group>
            <FormModal opened={opened} onClose={() => setOpened(false)} title={t('shared.devui.add')}>
              <Form form={create} onSubmit={async () => setOpened(false)}>
                <FormSection title={t('shared.devui.form')}>
                  <TextField name="name" label={t('shared.devui.name')} required />
                  <MoneyField name="salary" label={t('shared.devui.money')} />
                </FormSection>
                <FormActions submitLabel={t('shared.devui.save')} onCancel={() => setOpened(false)} cancelLabel={t('shared.form.cancel')} />
              </Form>
            </FormModal>
          </Case>
          <Case label={t('shared.devui.case_edit_in_place')}>
            <EditablePanel
              title={t('shared.devui.section_staff')}
              icon={IconUsersGroup}
              editable
              edit={(close) => (
                <Form form={edit} onSubmit={async () => close()}>
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
        </>
      }
    />
  )
}

export function ConfirmDeletePattern() {
  const { t } = useTranslation()
  const { run, error, dialog } = useAction()
  const remove = () =>
    void run(
      async () => {
        throw new ApiError(503, 'network_error')
      },
      { confirm: { title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true } },
    )
  return (
    <DocPage
      path="patterns/confirm-delete"
      example={
        <Case>
          <Stack gap="md">
            <Group justify="space-between">
              <Text size="sm">{`${rows[0]!.code} · ${rows[0]!.name}`}</Text>
              <MoreActions>
                <MenuAction icon={IconDownload} onClick={() => {}}>
                  {t('shared.devui.export')}
                </MenuAction>
                <MenuAction icon={IconTrash} danger onClick={remove}>
                  {t('shared.devui.danger')}
                </MenuAction>
              </MoreActions>
            </Group>
            {error && <Notice tone="danger">{error}</Notice>}
          </Stack>
          {dialog}
        </Case>
      }
      doDont={<DoDont good={<Text size="sm">{t('shared.devui.do_confirm_verb')}</Text>} bad={<Text size="sm">{t('shared.devui.dont_confirm_undoable')}</Text>} />}
    />
  )
}

export function LoadingEmptyErrorPattern() {
  const { t } = useTranslation()
  const [state, setState] = useState<QueryCase>('data')
  const columns = useColumns()
  return (
    <DocPage
      path="patterns/loading-empty-error"
      example={
        <Case>
          <SegmentedControl w="fit-content" value={state} onChange={(v) => setState(v as QueryCase)} data={queryStates.map((s) => ({ value: s, label: t(`shared.devui.query_${s}`) }))} />
          <QueryView
            query={fakeQuery(state)}
            select={(data) => (state === 'no_results' ? [] : data)}
            empty={<EmptyState title={t('shared.devui.empty')} />}
            noResults={<EmptyState title={t('shared.devui.no_results')} action={<Button variant="default">{t('shared.list.clear_filters')}</Button>} />}
            notFound={<EmptyState title={t('shared.notfound.description')} />}
          >
            {(data) => <DataTable label={t('shared.devui.table')} rows={data} rowKey={(r) => r.id} columns={columns} />}
          </QueryView>
        </Case>
      }
    />
  )
}

export function BulkPattern() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<Set<string | number>>(new Set())
  const peek = usePeek()
  const view = useListView('dev.ui', useColumns())
  const open = rows.find((r) => String(r.id) === peek.id)
  return (
    <DocPage
      path="patterns/bulk"
      example={
        <Case>
          {selected.size > 0 ? (
            <BulkBar count={selected.size} onClear={() => setSelected(new Set())}>
              <Button variant="default">{t('shared.devui.export')}</Button>
            </BulkBar>
          ) : (
            <Group justify="space-between">
              <Text size="sm" c="dimmed">
                {t('shared.devui.bulk_none')}
              </Text>
              {view.tools}
            </Group>
          )}
          <DataTable
            label={t('shared.devui.bulk')}
            columns={view.columns}
            rows={rows}
            rowKey={(r) => r.id}
            href={(r) => `/dev/ui/patterns/bulk?peek=${r.id}`}
            onRowClick={(r) => peek.open(r.id)}
            selection={{ selected, onChange: setSelected }}
            onResize={view.onResize}
          />
          <SidePanel title={open?.name ?? ''} href="/dev/ui/patterns/bulk" opened={!!open} onClose={peek.close}>
            {open && (
              <FieldList
                rows={[
                  [t('shared.devui.code'), open.code],
                  [t('shared.devui.amount'), formatNumber(open.amount)],
                ]}
              />
            )}
          </SidePanel>
        </Case>
      }
    />
  )
}

// ponytail: no DocumentActions here; it runs real mutations the offline catalogue cannot answer.
// Add it once app/devui/api.ts serves document transitions.
export function DocumentApprovalPattern() {
  const { t } = useTranslation()
  return (
    <SampleDocument>
      <DocPage
        path="patterns/document-approval"
        example={
          <Case>
            <Group>
              <DocumentStatus docType="shared" status="pending_approval" />
            </Group>
            <RecordColumns
              main={
                <Panel title={t('shared.document.tab.activity')}>
                  <ActivityFeed docType={sample.type} id={sample.id} recent={5} />
                </Panel>
              }
              side={<ApprovalPanel docType={sample.type} id={sample.id} />}
            />
          </Case>
        }
      />
    </SampleDocument>
  )
}
