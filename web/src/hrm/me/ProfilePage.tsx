import { FieldList, Stack } from '@mmerp/ui'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/hrm'
import { formatDate } from '@/shared/i18n'
import { Page } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { Section } from '@/shared/ui/Section'
import { hrmKeys } from '../keys'

// ProfilePage: the user's own record, read only; changes go through HR.
export function ProfilePage() {
  const { t } = useTranslation()
  const q = useQuery({ queryKey: hrmKeys.me.profile(), queryFn: () => unwrap(api.GET('/hrm/me')) })
  return (
    <Page title={t('hrm.me.profile.title')} description={t('hrm.me.profile.description')} centered>
      <QueryView query={q}>
        {(p) => {
          const dash = '–'
          const date = (d: string | null) => (d ? formatDate(d) : dash)
          return (
            <Stack gap="lg">
              <Section title={t('hrm.me.profile.general')}>
                <FieldList
                  rows={[
                    [t('hrm.employee.code'), p.code],
                    [t('hrm.employee.full_name'), p.full_name],
                    [t('hrm.employee.date_of_birth'), date(p.date_of_birth)],
                    [t('hrm.employee.gender'), p.gender ? t(`hrm.employee.gender.${p.gender}`) : dash],
                    [t('hrm.employee.phone'), p.phone ?? dash],
                    [t('hrm.employee.email'), p.email ?? dash],
                    [t('hrm.employee.address'), p.address ?? dash, 'wide'],
                  ]}
                />
              </Section>
              <Section title={t('hrm.me.profile.work')}>
                <FieldList
                  rows={[
                    [t('hrm.employee.org_unit'), p.org_unit_name],
                    [t('hrm.employee.manager'), p.manager_name ?? dash],
                    [t('hrm.employee.hire_date'), formatDate(p.hire_date)],
                    [t('hrm.employee.termination_date'), date(p.termination_date)],
                  ]}
                />
              </Section>
              {p.contract && (
                <Section title={t('hrm.me.profile.contract')}>
                  <FieldList
                    rows={[
                      [t('hrm.contract.field.contract_type'), p.contract.type_name],
                      [t('hrm.contract.field.start_date'), formatDate(p.contract.start_date)],
                      [t('hrm.contract.field.end_date'), date(p.contract.end_date)],
                    ]}
                  />
                </Section>
              )}
            </Stack>
          )
        }}
      </QueryView>
    </Page>
  )
}
