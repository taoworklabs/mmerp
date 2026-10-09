import { Alert, Button, Group, Loader, Stack, Text } from '@mantine/core'
import { IconAlertTriangle, IconCalculator, IconPrinter, IconReceipt2 } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams, useSearchParams } from 'react-router'
import { api, unwrap, type Payroll, type PayrollLine } from '@/shared/api/hrm'
import { ApprovalPanel, DocumentActions, DocumentHistory, DocumentStatus, documentKeys, useDocumentMutation, useDiscussionSection } from '@/shared/document'
import { errorText, formatDateTime, formatMonth, formatNumber } from '@/shared/i18n'
import { ExportButton, finished, jobError, useJobStatus } from '@/shared/jobs'
import { useMe } from '@/shared/auth/me'
import { DataTable } from '@/shared/ui/DataTable'
import { DocumentPage, FormModal } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { AdjustmentsModal } from '../components/AdjustmentsModal'
import { loaded } from '@/shared/ui/loaded'
import { PayrollTable } from '../components/PayrollTable'
import { usePayroll } from '../hooks/usePayroll'
import { hrmKeys, payrollDocType } from '../keys'

// Payslips print through the export of their template.
const payslipTemplate = 'printing.hrm.payslip'

export function PayrollPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const id = Number(useParams().id)
  const discussion = useDiscussionSection(payrollDocType, id)
  const [search, setSearch] = useSearchParams()
  const job = Number(search.get('job')) || null
  const payroll = usePayroll(id)
  const status = useJobStatus(job, [hrmKeys.payrolls.detail(id), hrmKeys.payrolls.lists(), documentKeys.history(payrollDocType, id)])
  const [error, setError] = useState<string | null>(null)
  const [adjusting, setAdjusting] = useState(false)
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
        p.allowed_actions.length > 0 && (
          <Group gap="xs">
            {can('compute') && <ComputeButton payroll={p} busy={computing} onQueued={follow} onError={setError} />}
            {can('adjust') && (
              <Button variant="default" leftSection={<IconReceipt2 {...icon.button} />} disabled={computing} onClick={() => setAdjusting(true)}>
                {t('hrm.payroll.adjustments')}
              </Button>
            )}
            {can('export') && <ExportButton kind={payrollDocType} params={{ payroll_id: p.id }} label={t('hrm.payroll.export')} />}
            {can('print') && <ExportButton kind={payslipTemplate} params={{ id: p.id }} label={t('hrm.payroll.print_payslips')} icon={IconPrinter} />}
            <DocumentActions
              docType={payrollDocType}
              id={p.id}
              version={p.version}
              number={p.number}
              actions={computing ? [] : p.allowed_actions}
              onError={setError}
              onDelete={async (version) => {
                await unwrap(api.DELETE('/hrm/payrolls/{id}', { params: { path: { id: p.id }, query: { version } } }))
                navigate('/hrm/payrolls', { replace: true })
              }}
            />
          </Group>
        )
      }
      fullWidth
      approval={<ApprovalPanel docType={payrollDocType} id={p.id} />}
      sections={[discussion]}
      history={<DocumentHistory docType={payrollDocType} id={p.id} />}
    >
      <Stack gap="md">
        <PayrollState payroll={p} computing={computing} failed={status.data?.state === 'failed' ? jobError(t, status.data) : status.isError ? errorText(t, status.error) : null} />
        {p.computed_at && <PayrollTable payroll={p} onPrint={can('print') ? setPayslipOf : undefined} />}
        {p.adjustments && p.adjustments.length > 0 && (
          <Stack gap="xs">
            <Text fw={600}>{t('hrm.payroll.adjustments')}</Text>
            <DataTable
              label={t('hrm.payroll.adjustments')}
              rows={p.adjustments}
              rowKey={(a) => `${a.employee_id}:${a.source_period}:${a.reason}:${a.amount}`}
              columns={[
                { key: 'employee', header: t('hrm.payroll.employee'), role: 'title', render: (a) => `${a.employee_code} · ${a.employee_name}` },
                { key: 'period', header: t('hrm.payroll.source_period'), role: 'meta', render: (a) => formatMonth(a.source_period) },
                { key: 'reason', header: t('hrm.payroll.reason'), role: 'meta', render: (a) => a.reason },
                { key: 'amount', header: t('hrm.payroll.col.adjustment'), role: 'meta', numeric: true, render: (a) => formatNumber(a.amount) },
              ]}
            />
          </Stack>
        )}
      </Stack>
      {adjusting && <AdjustmentsModal payroll={p} onQueued={follow} onClose={() => setAdjusting(false)} />}
      {payslipOf && (
        <FormModal opened title={t('hrm.payroll.print_payslip')} onClose={() => setPayslipOf(null)}>
          <Stack gap="md">
            <Text>{`${payslipOf.employee_code} · ${payslipOf.employee_name}`}</Text>
            <Group justify="flex-end">
              <ExportButton kind={payslipTemplate} params={{ id: p.id, parts: [payslipOf.employee_id] }} label={t('hrm.payroll.print_payslip')} icon={IconPrinter} />
            </Group>
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
        <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />} title={t('hrm.payroll.overtime_limit')}>
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
