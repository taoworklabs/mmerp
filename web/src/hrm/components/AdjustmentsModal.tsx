import { ActionIcon, Button, Group, Stack, Text, Tooltip } from '@mantine/core'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Payroll } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { Form, FormActions, MoneyField, MonthField, TextField } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { payrollDocType } from '../keys'
import { EmployeeField } from './EmployeeField'

type Item = { employee_id: string | null; amount: number | null; source_period: string | null; reason: string }

// AdjustmentsModal replaces the back pay and recoveries of a draft payroll; saving queues a
// new computation, whose job onQueued follows.
export function AdjustmentsModal({ payroll, onQueued, onClose }: { payroll: Payroll; onQueued: (job: number) => void; onClose: () => void }) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(payrollDocType, payroll.id, payroll.version)
  const current = payroll.adjustments ?? []
  const form = useForm<{ items: Item[] }>({
    mode: 'onBlur',
    defaultValues: {
      items: current.map((a) => ({ employee_id: String(a.employee_id), amount: a.amount, source_period: a.source_period, reason: a.reason })),
    },
  })
  const items = useFieldArray({ control: form.control, name: 'items' })

  async function save(v: { items: Item[] }) {
    const body = v.items.map((i) => ({ employee_id: Number(i.employee_id), amount: i.amount ?? 0, source_period: i.source_period ?? '', reason: i.reason.trim() }))
    const out = await mutation.run((version) => unwrap(api.PUT('/hrm/payrolls/{id}/adjustments', { params: { path: { id: payroll.id } }, body: { version, items: body } })))
    onQueued(out.job_id)
    onClose()
  }

  return (
    <FormModal opened title={t('hrm.payroll.adjustments')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <Text size="sm" c="dimmed">
          {t('hrm.payroll.adjustments_hint')}
        </Text>
        {items.fields.map((item, i) => {
          const n = i + 1
          const was = current[i]
          return (
            <Stack key={item.id} gap="xs">
              <EmployeeField
                name={`items.${i}.employee_id`}
                label={t('hrm.payroll.adjustment_employee', { n })}
                current={was && { id: was.employee_id, code: was.employee_code, name: was.employee_name }}
                withTerminated
                required
              />
              <MoneyField name={`items.${i}.amount`} label={t('hrm.payroll.adjustment_amount', { n })} description={t('hrm.payroll.adjustment_amount_hint')} min={-1e12} required />
              <MonthField name={`items.${i}.source_period`} label={t('hrm.payroll.adjustment_period', { n })} required />
              <TextField name={`items.${i}.reason`} label={t('hrm.payroll.adjustment_reason', { n })} required maxLength={1000} />
              <Group justify="flex-end">
                <Tooltip label={t('hrm.payroll.remove_adjustment', { n })}>
                  <ActionIcon variant="subtle" color="danger" size="lg" aria-label={t('hrm.payroll.remove_adjustment', { n })} onClick={() => items.remove(i)}>
                    <IconTrash {...icon.button} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            </Stack>
          )
        })}
        <Group>
          <Button size="xs" variant="light" leftSection={<IconPlus {...icon.text} />} onClick={() => items.append({ employee_id: null, amount: null, source_period: null, reason: '' })}>
            {t('hrm.payroll.add_adjustment')}
          </Button>
        </Group>
        <FormActions submitLabel={t('hrm.payroll.save_compute')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
      {mutation.dialog}
    </FormModal>
  )
}
