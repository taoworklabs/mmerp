import { Anchor, List, Menu, Stack, Text } from '@mantine/core'
import { IconLock } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, ApiError, unwrap, type PeriodLock } from '@/shared/api/core'
import { useRecordLink } from '@/shared/area'
import { errorText, formatDate } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { DateField, Form, FormActions } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

// PeriodLocksPage: each legal entity's lock date. Documents dated on or before it are frozen.
export function PeriodLocksPage() {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<PeriodLock | null>(null)
  const locks = useQuery({ queryKey: coreKeys.periodLocks(), queryFn: async () => (await unwrap(api.GET('/period-locks'))) ?? [] })
  const columns: Column<PeriodLock>[] = [
    { key: 'name', header: t('core.period.legal_entity'), role: 'title', render: (l) => l.legal_entity_name },
    { key: 'locked_until', header: t('core.period.locked_until'), role: 'meta', render: (l) => (l.locked_until ? formatDate(l.locked_until) : t('core.period.open')) },
  ]
  let body
  if (locks.isPending) body = <ContentSkeleton />
  else if (locks.isError) body = <ErrorState message={errorText(t, locks.error)} onRetry={() => void locks.refetch()} />
  else if (locks.data.length === 0) body = <EmptyState title={t('core.period.empty')} />
  else
    body = (
      <DataTable
        label={t('core.period.title')}
        columns={columns}
        rows={locks.data}
        rowKey={(l) => l.legal_entity_id}
        menu={(l) => (
          <Menu.Item leftSection={<IconLock {...icon.text} />} onClick={() => setEditing(l)}>
            {t('core.period.change')}
          </Menu.Item>
        )}
      />
    )
  return (
    <ListPage title={t('core.period.title')} description={t('core.period.description')}>
      {body}
      {editing && <LockModal lock={editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type Pending = { doc_type: string; id: number; number: string; date: string }

function LockModal({ lock, onClose }: { lock: PeriodLock; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [pending, setPending] = useState<Pending[]>([])
  const form = useForm<{ locked_until: string | null }>({ mode: 'onBlur', defaultValues: { locked_until: lock.locked_until ?? null } })

  async function save(v: { locked_until: string | null }) {
    setPending([])
    try {
      await unwrap(api.PUT('/period-locks/{legal_entity}', { params: { path: { legal_entity: lock.legal_entity_id } }, body: { locked_until: v.locked_until } }))
    } catch (err) {
      if (err instanceof ApiError && err.code === 'period_has_pending_documents') setPending((err.params.documents as Pending[]) ?? [])
      throw err
    }
    await qc.invalidateQueries({ queryKey: coreKeys.periodLocks() })
    notifySuccess(t('core.period.saved'))
    onClose()
  }

  return (
    <FormModal opened title={t('core.period.change_title', { name: lock.legal_entity_name })} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <DateField name="locked_until" label={t('core.period.locked_until')} description={t('core.period.locked_until_hint')} clearable />
        {pending.length > 0 && (
          <Stack gap="xs">
            <Text size="sm" fw={500}>
              {t('core.period.pending')}
            </Text>
            <List size="sm">
              {pending.map((d) => (
                <PendingItem key={d.id} doc={d} />
              ))}
            </List>
          </Stack>
        )}
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}

function PendingItem({ doc }: { doc: Pending }) {
  const link = useRecordLink(doc.doc_type, doc.id)
  const text = `${doc.number} · ${formatDate(doc.date)}`
  return (
    <List.Item>
      {link ? (
        <Anchor component={Link} to={link} size="sm">
          {text}
        </Anchor>
      ) : (
        text
      )}
    </List.Item>
  )
}
