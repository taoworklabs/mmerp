import { Button, Loader, Stack, Text } from '@mmerp/ui'
import { IconCalculator, IconPrinter } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useSearchParams } from 'react-router'
import { api, unwrap, type Payroll, type PayrollLine } from '@/shared/api/hrm'
import { DocumentScreen, useDocumentMutation } from '@/shared/document'
import { errorText, formatDateTime, formatMonth } from '@/shared/i18n'
import { ExportButton, finished, jobError, useJobStatus } from '@/shared/jobs'
import { useMe } from '@/shared/auth/me'
import { Notice } from '@/shared/ui/Notice'
import { FormModal } from '@/shared/ui/page'
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
  const [payslipOf, setPayslipOf] = useState<PayrollLine | null>(null)
  // The computing job is on the URL, so a reload keeps following it.
  const follow = (jobId: number) => setSearch({ job: String(jobId) }, { replace: true })

  const state = loaded(payroll, t, { to: '/hrm/payrolls', label: t('hrm.payrolls.back') })
  if (!state.ok) return state.fallback
  const p = state.data
  const can = (a: string) => p.allowed_actions.includes(a)
  const computing = job !== null && !finished(status.data) && !status.isError
  return (
    <DocumentScreen
      docType={payrollDocType}
      doc={p}
      list={{ label: t('hrm.payrolls.title'), to: '/hrm/payrolls' }}
      facts={[p.legal_entity_name, formatMonth(p.period_start.slice(0, 7))]}
      fullWidth
      extraActions={(onError) => (
        <>
          {can('compute') && <ComputeButton payroll={p} busy={computing} onQueued={follow} onError={onError} />}
          {can('export') && <ExportButton kind={payrollDocType} params={{ payroll_id: p.id }} label={t('hrm.payroll.export')} />}
        </>
      )}
      actions={computing ? [] : p.allowed_actions}
      printLabel={t('hrm.payroll.print_payslips')}
      printKind={payslipTemplate}
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
    </DocumentScreen>
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
      <Notice tone="info" icon={<Loader size="sm" label={t('shared.state.loading')} />} announce="polite">
        {t('hrm.payroll.computing')}
      </Notice>
    )
  const warned = (p.lines ?? []).filter((l) => l.warnings.length > 0)
  return (
    <>
      {failed && (
        <Notice tone="danger" title={t('hrm.payroll.compute_failed')}>
          {failed}
        </Notice>
      )}
      {!p.computed_at && !failed && (
        <Notice tone="warning">{t('hrm.payroll.not_computed')}</Notice>
      )}
      {p.sources_changed && (
        <Notice tone="warning" announce="alert" title={t('hrm.payroll.sources_changed')}>
          {t('hrm.payroll.sources_changed_hint')}
        </Notice>
      )}
      {warned.length > 0 && (
        <Notice tone="warning" title={t('hrm.payroll.warnings')}>
          {warned.map((l) => `${l.employee_code} · ${l.employee_name}: ${l.warnings.map((w) => t(`hrm.payroll.warning.${w}`)).join(', ')}`).join('; ')}
        </Notice>
      )}
      {p.computed_at && (
        <Text size="sm" tone="dimmed">
          {t('hrm.payroll.computed_at', { at: formatDateTime(p.computed_at, me.timezone) })}
        </Text>
      )}
    </>
  )
}
