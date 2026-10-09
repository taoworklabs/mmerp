import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { DocForm } from '../components/DocForm'
import { useDocActions } from '../hooks/useSales'
import { pathOf, salesKeys, type Kind } from '../keys'

export function NewDocPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const actions = useDocActions(kind)
  const list = `/sales/${pathOf(kind)}`
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  return (
    <Page
      title={t(`sales.${pathOf(kind)}.create`)}
      breadcrumbs={[{ label: t(`sales.${pathOf(kind)}.title`), to: list }]}
      description={t(`sales.${pathOf(kind)}.create_description`)}
    >
      <DocForm
        kind={kind}
        onCreated={async (id) => {
          await qc.invalidateQueries({ queryKey: salesKeys.docs.lists(kind) })
          notifySuccess(t(`sales.${kind}.created`))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`${list}/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
