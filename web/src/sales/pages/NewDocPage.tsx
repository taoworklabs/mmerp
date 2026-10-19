import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { DocForm } from '../components/DocForm'
import { useDocActions } from '../hooks/useSales'
import { docTypeOf, pathOf, type Kind } from '../keys'

export function NewDocPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const actions = useDocActions(kind)
  return (
    <CreatePage
      docType={docTypeOf(kind)}
      actions={actions}
      title={t(`sales.${pathOf(kind)}.create`)}
      breadcrumbs={[{ label: t(`sales.${pathOf(kind)}.title`), to: `/sales/${pathOf(kind)}` }]}
      description={t(`sales.${pathOf(kind)}.create_description`)}
      created={t(`sales.${kind}.created`)}
    >
      {(onCreated) => <DocForm kind={kind} onCreated={onCreated} />}
    </CreatePage>
  )
}
