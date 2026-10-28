import { IconCalendarStats } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { useMe } from '@/shared/auth/me'
import { formatMonthShort, today } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FilterChip, LegalEntityFilter, MonthFilter, YearFilter, useLegalEntities } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { shiftMonth } from '../me/days'

// The first periods mmerp has the forms for: the 2026 versions.
const firstQuarter = '2026-Q3'
const firstYear = 2026

type Period = 'month' | 'quarter' | 'year'
type Report = { kind: string; form: string; period: Period }

const reports: Report[] = [
  { kind: 'hrm.report.d02lt', form: 'D02-LT', period: 'month' },
  { kind: 'hrm.report.pit_05kk', form: '05/KK-TNCN', period: 'quarter' },
  { kind: 'hrm.report.pit_05qtt', form: '05/QTT-TNCN', period: 'year' },
  { kind: 'hrm.report.pit_05_1bk', form: '05-1/BK-QTT-TNCN', period: 'year' },
  { kind: 'hrm.report.pit_05_3bk', form: '05-3/BK-QTT-TNCN', period: 'year' },
]

const quarterOf = (month: string) => `${month.slice(0, 4)}-Q${Math.ceil(Number(month.slice(5, 7)) / 3)}`

// quartersUpTo lists the quarters from the first the forms cover to last, latest first.
function quartersUpTo(last: string): string[] {
  const out: string[] = []
  for (let q = firstQuarter; q <= last; ) {
    out.unshift(q)
    const [y, n] = [Number(q.slice(0, 4)), Number(q.slice(6))]
    q = n === 4 ? `${y + 1}-Q1` : `${y}-Q${n + 1}`
  }
  return out
}

// ReportsPage starts the insurance and PIT filings of a legal entity, each from the posted
// payrolls of its period. The legal entity and the periods are on the URL.
export function ReportsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const [params, setParams] = useSearchParams()
  const now = today(me.timezone).slice(0, 7)
  const entities = useLegalEntities({ product: 'hrm', permission: 'hrm.salary.view' }, params.get('legal_entity_id') ?? undefined)
  // By default the last period that has ended.
  const month = params.get('month') ?? shiftMonth(now, -1)
  const quarters = quartersUpTo(quarterOf(now))
  const quarter = params.get('quarter') ?? quarters.find((q) => q < quarterOf(now)) ?? quarters[0] ?? firstQuarter
  const thisYear = Number(now.slice(0, 4))
  const years = Array.from({ length: Math.max(1, thisYear - firstYear + 1) }, (_, i) => firstYear + i)
  const year = Number(params.get('year')) || Math.max(firstYear, thisYear - 1)
  const set = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    next.set(key, value)
    setParams(next, { replace: true })
  }

  const periodParam = (p: Period) => (p === 'month' ? { month } : p === 'quarter' ? { quarter } : { year })
  const periodText = (p: Period) =>
    p === 'month' ? formatMonthShort(month) : p === 'quarter' ? t('hrm.reports.quarter', { n: quarter.slice(6), year: quarter.slice(0, 4) }) : String(year)
  const columns: Column<Report>[] = [
    { key: 'form', header: t('hrm.reports.form'), role: 'title', kind: 'code', render: (r) => r.form },
    { key: 'name', header: t('hrm.reports.name'), role: 'meta', render: (r) => t(`hrm.reports.${r.kind.slice('hrm.report.'.length)}`) },
    { key: 'period', header: t('hrm.reports.period'), role: 'meta', render: (r) => periodText(r.period) },
    {
      key: 'export',
      header: t('hrm.reports.file'),
      role: 'status',
      render: (r) =>
        entities.chosen && (
          <ExportButton kind={r.kind} params={{ legal_entity_id: entities.chosen, ...periodParam(r.period) }} label={t('hrm.reports.export')} />
        ),
    },
  ]

  return (
    <ListPage
      title={t('hrm.reports.title')}
      description={t('hrm.reports.description')}
      filters={
        <>
          <LegalEntityFilter label={t('hrm.reports.legal_entity')} options={entities.options} value={entities.chosen} onChange={(v) => set('legal_entity_id', v)} />
          <MonthFilter label={t('hrm.reports.month')} value={month} onChange={(m) => m && set('month', m)} />
          <FilterChip
            label={t('hrm.reports.quarter_filter')}
            icon={IconCalendarStats}
            data={quarters.map((q) => ({ value: q, label: t('hrm.reports.quarter', { n: q.slice(6), year: q.slice(0, 4) }) }))}
            value={quarter}
            onChange={(v) => v && set('quarter', v)}
            clearable={false}
          />
          <YearFilter label={t('hrm.reports.year')} value={year} years={years} onChange={(y) => set('year', String(y))} />
        </>
      }
    >
      {!entities.isPending && entities.options.length === 0 ? (
        <EmptyState title={t('hrm.reports.no_legal_entity')} />
      ) : (
        <DataTable label={t('hrm.reports.title')} columns={columns} rows={reports} rowKey={(r) => r.kind} />
      )}
    </ListPage>
  )
}
