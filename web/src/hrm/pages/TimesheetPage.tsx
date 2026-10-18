import { Button, Group } from '@mantine/core'
import { IconFileImport } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus } from '@/shared/document'
import { formatMonth } from '@/shared/i18n'
import { ExportButton, ImportDialog } from '@/shared/jobs'
import { DocumentPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { loaded } from '@/shared/ui/loaded'
import { TimesheetGrid } from '../components/TimesheetGrid'
import { useTimesheet } from '../hooks/useTimesheet'
import { timesheetDocType } from '../keys'

export function TimesheetPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const timesheet = useTimesheet(id)
  const [error, setError] = useState<string | null>(null)
  const [importing, setImporting] = useState(false)

  const state = loaded(timesheet, t, { to: '/hrm/timesheets', label: t('hrm.timesheets.back') })
  if (!state.ok) return state.fallback
  const ts = state.data
  const can = (a: string) => ts.allowed_actions.includes(a)
  const params = { timesheet_id: ts.id }
  return (
    <DocumentPage
      fullWidth
      title={ts.number}
      breadcrumbs={[{ label: t('hrm.timesheets.title'), to: '/hrm/timesheets' }]}
      description={
        <Group gap="xs" component="span">
          <span>{ts.org_unit_name}</span>
          <span aria-hidden>·</span>
          <span>{formatMonth(ts.period_start.slice(0, 7))}</span>
          <DocumentStatus docType={timesheetDocType} status={ts.status} />
        </Group>
      }
      error={error}
      actions={
        (
          <Group gap="xs">
            <ApprovalDecision docType={timesheetDocType} id={ts.id} onError={setError} />
            {can('import') && (
              <Button variant="default" leftSection={<IconFileImport {...icon.button} />} onClick={() => setImporting(true)}>
                {t('hrm.timesheet.import')}
              </Button>
            )}
            {can('export') && <ExportButton kind={timesheetDocType} params={params} label={t('hrm.timesheet.export')} />}
            <DocumentActions
              docType={timesheetDocType}
              id={ts.id}
              version={ts.version}
              number={ts.number}
              actions={ts.allowed_actions}
              onError={setError}
              listPath="/hrm/timesheets"
            />
          </Group>
        )
      }
      approval={<ApprovalPanel docType={timesheetDocType} id={ts.id} decideElsewhere />}
      activity={<ActivityFeed docType={timesheetDocType} id={ts.id} />}
    >
      {/* Keyed by version: an import or a reload after a conflict starts from the saved data. */}
      <TimesheetGrid key={ts.version} timesheet={ts} />
      {importing && (
        <ImportDialog
          kind={timesheetDocType}
          params={params}
          title={t('hrm.timesheet.import_title', { number: ts.number })}
          description={t('hrm.timesheet.import_hint')}
          writes={{ docType: timesheetDocType, id: ts.id }}
          onClose={() => setImporting(false)}
        />
      )}
    </DocumentPage>
  )
}
