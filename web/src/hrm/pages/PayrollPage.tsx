import { Alert, Button, Group, Loader, Stack, Text } from '@mantine/core'
import { IconAlertTriangle, IconCalculator, IconPrinter } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useSearchParams } from 'react-router'
import { api, unwrap, type Payroll, type PayrollLine } from '@/shared/api/hrm'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus, useDocumentMutation } from '@/shared/document'
import { errorText, formatDateTime, formatMonth } from '@/shared/i18n'
import { ExportButton, finished, jobError, useJobStatus } from '@/shared/jobs'
import { useMe } from '@/shared/auth/me'
import { DocumentPage, FormModal } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { AdjustmentsSection } from '../components/AdjustmentsSection'
import { loaded } from '@/shared/ui/loaded'
import { ModalFooter } from '@/shared/ui/ModalFooter'
import { PayrollTable } from '../components/PayrollTable'
import { usePayroll } from '../hooks/usePayroll'
import { payrollDocType } from '../keys'

// Payslips print through the export of their template.
const payslipTemplate = 'printing.hrm.payslip'

export function PayrollPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const [search, setSearch] = useSearchParams()
  const job = Number(search.get('job')) || null
  const payroll = usePayroll(id)
  const status = useJobStatus(job, { docType: payrollDocType, id })
  const [error, setError] = useState<string | null>(null)
  const [payslipOf, setPayslipOf] = useState<PayrollLine | null>(null)
  // The computing job is on the URL, so a reload keeps following it.
  const follow = (jobId: number) => setSearch({ job: String(jobId) }, { replace: true })

  const state = loaded(payroll, t, { to: '/hrm/payrolls', label: t('hrm.payrolls.back') })
  if (!state.ok) return state.fallback
  const p = state.data
  const can = (a: string) => p.allowed_actions.includes(a)
  const computing = job !== null && !finished(status.data) && !status.isError
  return (
    <DocumentPage
      title={p.number}
      breadcrumbs={[{ label: t('hrm.payrolls.title'), to: '/hrm/payrolls' }]}
      description={
        <Group gap="xs" component="span">
          <span>{p.legal_entity_name}</span>
          <span aria-hidden>·</span>
          <span>{formatMonth(p.period_start.slice(0, 7))}</span>
          <DocumentStatus docType={payrollDocType} status={p.status} />
        </Group>
      }
      error={error}
      actions={
        (
          <Group gap="xs">
            <ApprovalDecision docType={payrollDocType} id={p.id} onError={setError} />
            {can('compute') && <ComputeButton payroll={p} busy={computing} onQueued={follow} onError={setError} />}
            {can('export') && <ExportButton kind={payrollDocType} params={{ payroll_id: p.id }} label={t('hrm.payroll.export')} />}
            <DocumentActions
              docType={payrollDocType}
              id={p.id}
              version={p.version}
              number={p.number}
              actions={computing ? [] : p.allowed_actions}
              onError={setError}
              listPath="/hrm/payrolls"
              printLabel={t('hrm.payroll.print_payslips')}
              printKind={payslipTemplate}
            />
          </Group>
        )
      }
      fullWidth
      approval={<ApprovalPanel docType={payrollDocType} id={p.id} decideElsewhere />}
      activity={<ActivityFeed docType={payrollDocType} id={p.id} />}
    >
      <Stack gap="md">
        <PayrollState payroll={p} computing={computing} failed={status.data?.state === 'failed' ? jobError(t, status.data) : status.isError ? errorText(t, status.error) : null} />
        {p.computed_at && <PayrollTable payroll={p} onPrint={can('print') ? setPayslipOf : undefined} />}
        <AdjustmentsSection payroll={p} editable={can('adjust') && !computing} onQueued={follow} />
      </Stack>
      {payslipOf && (
        <FormModal opened title={t('hrm.payroll.print_payslip')} onClose={() => setPayslipOf(null)}>
          <Stack gap="md">
            <Text>{`${payslipOf.employee_code} · ${payslipOf.employee_name}`}</Text>
            <ModalFooter>
              <ExportButton kind={payslipTemplate} params={{ id: p.id, parts: [payslipOf.employee_id] }} label={t('hrm.payroll.print_payslip')} icon={IconPrinter} />
            </ModalFooter>
          </Stack>
        </FormModal>
      )}
    </DocumentPage>
  )
}

// ComputeButton queues a computation from the current sources, at the version on screen.
function ComputeButton({ payroll, busy, onQueued, onError }: { payroll: Payroll; busy: boolean; onQueued: (job: number) => void; onError: (e: string | null) => void }) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(payrollDocType, payroll.id, payroll.version)
  const [starting, setStarting] = useState(false)
  async function compute() {
    setStarting(true)
    onError(null)
    try {
      const out = await mutation.run((version) => unwrap(api.POST('/hrm/payrolls/{id}/compute', { params: { path: { id: payroll.id } }, body: { version } })))
      onQueued(out.job_id)
    } catch (err) {
      onError(errorText(t, err))
    } finally {
      setStarting(false)
    }
  }
  return (
    <>
      <Button variant="default" leftSection={<IconCalculator {...icon.button} />} loading={starting || busy} onClick={() => void compute()}>
        {t('hrm.payroll.compute')}
      </Button>
      {mutation.dialog}
    </>
  )
}

// PayrollState says where the computation is: running, failed, not done, out of date, or
// done with overtime past the tax-free limits.
function PayrollState({ payroll: p, computing, failed }: { payroll: Payroll; computing: boolean; failed: string | null }) {
  const { t } = useTranslation()
  const me = useMe()
  if (computing)
    return (
      <Alert color="info" aria-live="polite">
        <Group gap="sm">
          <Loader size="sm" />
          <Text size="sm">{t('hrm.payroll.computing')}</Text>
        </Group>
      </Alert>
    )
  const warned = (p.lines ?? []).filter((l) => l.warnings.length > 0)
  return (
    <>
      {failed && (
        <Alert color="danger" icon={<IconAlertTriangle {...icon.button} />} role="alert" title={t('hrm.payroll.compute_failed')}>
          {failed}
        </Alert>
      )}
      {!p.computed_at && !failed && (
        <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />}>
          {t('hrm.payroll.not_computed')}
        </Alert>
      )}
      {p.sources_changed && (
        <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />} role="alert" title={t('hrm.payroll.sources_changed')}>
          {t('hrm.payroll.sources_changed_hint')}
        </Alert>
      )}
      {warned.length > 0 && (
        <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />} title={t('hrm.payroll.warnings')}>
          {warned.map((l) => `${l.employee_code} · ${l.employee_name}: ${l.warnings.map((w) => t(`hrm.payroll.warning.${w}`)).join(', ')}`).join('; ')}
        </Alert>
      )}
      {p.computed_at && (
        <Text size="sm" c="dimmed">
          {t('hrm.payroll.computed_at', { at: formatDateTime(p.computed_at, me.timezone) })}
        </Text>
      )}
    </>
  )
}
