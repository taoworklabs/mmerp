import { Button, Group } from '@mmerp/ui'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap, type PITAuthorisationPerson } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { currentYear, errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { LegalEntityFilter, useLegalEntities, YearFilter } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { Section } from '@/shared/ui/Section'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { useAction } from '@/shared/ui/useAction'
import { hrmKeys } from '../keys'

// The first year mmerp has the finalisation forms for.
const firstYear = 2026

// PITAuthorisationsPage records who authorised a legal entity to finalise their PIT for a year,
// among those on its posted payrolls of that year. The legal entity and the year are on the URL.
export function PITAuthorisationsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [params, setParams] = useSearchParams()
  const entities = useLegalEntities({ product: 'hrm', permission: 'hrm.salary.view' }, params.get('legal_entity_id') ?? undefined)
  const entity = entities.chosen
  const thisYear = currentYear(me.timezone)
  const years = Array.from({ length: Math.max(1, thisYear - firstYear + 1) }, (_, i) => firstYear + i)
  const year = Number(params.get('year')) || Math.max(firstYear, thisYear - 1)
  const set = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    next.set(key, value)
    setParams(next, { replace: true })
  }
  const query = useQuery({
    queryKey: hrmKeys.pitAuthorisations(entity ?? 0, year),
    queryFn: () => unwrap(api.GET('/hrm/pit-authorisations/{legal_entity}/{year}', { params: { path: { legal_entity: entity!, year } } })),
    enabled: entity !== undefined,
  })

  let body
  if (entities.isPending || (entity !== undefined && query.isPending)) body = <ContentSkeleton />
  else if (entities.isError || query.isError) body = <ErrorState message={errorText(t, entities.error ?? query.error)} onRetry={() => void query.refetch()} />
  else if (entity === undefined || !query.data) body = <EmptyState title={t('hrm.reports.no_legal_entity')} />
  else if (query.data.people.length === 0) body = <EmptyState title={t('hrm.pit_authorisations.empty', { year })} />
  else
    body = (
      // Keyed so a refetch after saving, or another entity or year, starts from what is stored.
      <Authorisations
        key={`${entity}-${year}-${query.dataUpdatedAt}`}
        entity={entity}
        year={year}
        people={query.data.people}
        edit={query.data.allowed_actions.includes('edit')}
      />
    )

  return (
    <ListPage
      title={t('hrm.pit_authorisations.title')}
      description={t('hrm.pit_authorisations.description')}
      filters={
        <>
          <LegalEntityFilter label={t('hrm.reports.legal_entity')} options={entities.options} value={entity} onChange={(v) => set('legal_entity_id', v)} />
          <YearFilter label={t('hrm.reports.year')} value={year} years={years} onChange={(y) => set('year', String(y))} />
        </>
      }
    >
      {body}
    </ListPage>
  )
}

function Authorisations({ entity, year, people, edit }: { entity: number; year: number; people: PITAuthorisationPerson[]; edit: boolean }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const action = useAction()
  const stored = new Set<string | number>(people.filter((p) => p.authorised).map((p) => p.employee_id))
  const [picked, setPicked] = useState(stored)
  // Who authorised another legal entity cannot be picked here.
  const pickable = (ids: Set<string | number>) => new Set([...ids].filter((id) => !people.some((p) => p.employee_id === id && p.elsewhere)))
  const changed = picked.size !== stored.size || [...picked].some((id) => !stored.has(id))

  const save = () =>
    action.run(
      async () => {
        await unwrap(api.PUT('/hrm/pit-authorisations/{legal_entity}/{year}', { params: { path: { legal_entity: entity, year } }, body: { employee_ids: [...picked].map(Number) } }))
        await qc.invalidateQueries({ queryKey: hrmKeys.pitAuthorisations(entity, year) })
      },
      { success: t('hrm.pit_authorisations.saved') },
    )

  const columns: Column<PITAuthorisationPerson>[] = [
    { key: 'code', header: t('hrm.employee.code'), role: 'title', kind: 'code', render: (p) => p.code },
    { key: 'name', header: t('hrm.employee.full_name'), role: 'meta', render: (p) => p.full_name },
    { key: 'december', header: t('hrm.pit_authorisations.december'), role: 'meta', render: (p) => (p.december ? t('hrm.pit_authorisations.yes') : '–') },
    {
      key: 'status',
      header: t('hrm.pit_authorisations.status'),
      role: 'status',
      render: (p) =>
        p.elsewhere ? t('hrm.pit_authorisations.elsewhere') : picked.has(p.employee_id) ? t('hrm.pit_authorisations.authorised') : t('hrm.pit_authorisations.not_authorised'),
    },
  ]

  return (
    <Section
      title={t('hrm.pit_authorisations.people', { year })}
      description={t('hrm.pit_authorisations.evidence')}
      actions={
        edit && (
          <Group gap="xs">
            <Button variant="default" onClick={() => setPicked(pickable(new Set(people.filter((p) => p.december).map((p) => p.employee_id))))}>
              {t('hrm.pit_authorisations.pick_december')}
            </Button>
            <Button disabled={!changed} onClick={() => void save()}>
              {t('hrm.pit_authorisations.save')}
            </Button>
          </Group>
        )
      }
    >
      {action.error && <ErrorState message={action.error} />}
      <DataTable
        label={t('hrm.pit_authorisations.people', { year })}
        columns={columns}
        rows={people}
        rowKey={(p) => p.employee_id}
        selection={edit ? { selected: picked, onChange: (next) => setPicked(pickable(next)) } : undefined}
      />
    </Section>
  )
}
