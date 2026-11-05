import { Anchor, Button, Group } from '@mmerp/ui'
import { IconBook, IconRefresh } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router'
import { api, unwrap, type PendingDoc } from '@/shared/api/accounting'
import { useRecordLinks } from '@/shared/area'
import { useCan } from '@/shared/auth/me'
import { formatDate } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { LegalEntityFilter, useLegalEntities } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge, type StatusTone } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useAction } from '@/shared/ui/useAction'
import { kindLabel } from '../balance'
import { asList } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

const reasonTone: Record<PendingDoc['reason'], StatusTone> = { queued: 'info', unmapped: 'warning', unbalanced: 'negative' }

// PendingPage: the documents of a legal entity whose lines have not reached the ledger, and why;
// each opens in the area that owns it.
export function PendingPage() {
  const { t } = useTranslation()
  const links = useRecordLinks()
  const canPost = useCan('accounting.post')
  const action = useAction()
  const [search, setSearch] = useSearchParams()
  const entities = useLegalEntities({ product: 'accounting', permission: 'accounting.view' }, search.get('legal_entity_id') ?? undefined)
  const le = entities.chosen
  const pending = useQuery({
    queryKey: accountingKeys.pending(le ?? 0),
    queryFn: async () => (await unwrap(api.GET('/accounting/pending', { params: { query: { legal_entity_id: le! } } }))).items ?? [],
    enabled: !!le,
  })

  const post = () =>
    action.run(
      async () => {
        const { queued } = await unwrap(api.POST('/accounting/pending/post', { body: { legal_entity_id: le! } }))
        await pending.refetch()
        return queued
      },
      { success: t('accounting.pending.posted') },
    )

  const columns: Column<PendingDoc>[] = [
    {
      key: 'number',
      header: t('accounting.pending.number'),
      role: 'title',
      kind: 'code',
      render: (d) => {
        const link = links(d.doc_type, d.doc_id)
        return link ? <Anchor render={<Link to={link} />}>{d.number}</Anchor> : d.number
      },
    },
    { key: 'type', header: t('accounting.pending.type'), role: 'meta', render: (d) => t(`${d.doc_type}.name`, { defaultValue: d.doc_type }) },
    { key: 'date', header: t('accounting.pending.date'), role: 'meta', kind: 'date', render: (d) => formatDate(d.date) },
    { key: 'missing', header: t('accounting.pending.missing_kinds'), role: 'meta', render: (d) => (d.missing_kinds ?? []).map((k) => kindLabel(t, k)).join(', ') },
    {
      key: 'reason',
      header: t('accounting.pending.reason'),
      role: 'status',
      kind: 'status',
      render: (d) => <StatusBadge tone={reasonTone[d.reason]}>{t(`accounting.pending.reason_${d.reason}`)}</StatusBadge>,
    },
  ]

  return (
    <ListPage
      title={t('accounting.pending.title')}
      description={t('accounting.pending.description')}
      action={
        <Group gap="xs">
          <Button variant="default" leftSection={<IconRefresh {...icon.button} />} loading={pending.isFetching && !pending.isPending} onClick={() => void pending.refetch()}>
            {t('accounting.pending.refresh')}
          </Button>
          {canPost && !!pending.data?.length && (
            <Button leftSection={<IconBook {...icon.button} />} onClick={() => void post()}>
              {t('accounting.pending.post')}
            </Button>
          )}
        </Group>
      }
      filters={
        <LegalEntityFilter
          label={t('accounting.common.legal_entity')}
          options={entities.options}
          value={le}
          onChange={(v) => setSearch(new URLSearchParams({ legal_entity_id: v }))}
        />
      }
      list={
        le
          ? {
              query: asList(pending, pending.data),
              table: { label: t('accounting.pending.title'), columns, rowKey: (d) => `${d.doc_type}:${d.doc_id}` },
              empty: { title: t('accounting.pending.empty'), description: t('accounting.pending.empty_description') },
              noResults: t('accounting.common.no_results'),
            }
          : undefined
      }
    >
      {!le && !entities.isPending && <EmptyState icon={IconBook} title={t('accounting.common.no_legal_entity')} />}
      {action.error && <ErrorState message={action.error} />}
      {action.dialog}
    </ListPage>
  )
}
