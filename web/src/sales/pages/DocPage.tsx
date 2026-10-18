import { Anchor, Button, Group } from '@mantine/core'
import { IconPrinter, IconShoppingCartPlus } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/sales'
import { useInvalidateRecord } from '@/shared/area'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus, useAttachmentSection } from '@/shared/document'
import { errorText } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { DocumentPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { DocForm } from '../components/DocForm'
import { DocProgress } from '../components/DocProgress'
import { useDoc } from '../hooks/useSales'
import { docTypeOf, pathOf, quoteDocType, type Kind } from '../keys'

// DocPage shows a quotation or order; a draft the user may edit is edited in place.
export function DocPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const id = Number(useParams().id)
  const docType = docTypeOf(kind)
  const list = `/sales/${pathOf(kind)}`
  const doc = useDoc(kind, id)
  const attachments = useAttachmentSection(docType, id)
  const [error, setError] = useState<string | null>(null)
  const [ordering, setOrdering] = useState(false)

  const state = loaded(doc, t, { to: list, label: t(`sales.${pathOf(kind)}.back`) })
  if (!state.ok) return state.fallback
  const d = state.data

  async function makeOrder() {
    setOrdering(true)
    setError(null)
    try {
      const { id: order } = await unwrap(api.POST('/sales/quotes/{id}/order', { params: { path: { id: d.id } } }))
      await invalidate(quoteDocType, d.id)
      notifySuccess(t('sales.quote.ordered'))
      navigate(`/sales/orders/${order}`)
    } catch (err) {
      setError(errorText(t, err))
    } finally {
      setOrdering(false)
    }
  }

  const linked = d.order ?? d.quote
  return (
    <DocumentPage
      title={d.number}
      breadcrumbs={[{ label: t(`sales.${pathOf(kind)}.title`), to: list }]}
      description={
        <Group gap="xs" component="span">
          <span>{d.customer.name}</span>
          <span aria-hidden>·</span>
          <span>{d.org_unit_name}</span>
          <DocumentStatus docType={docType} status={d.status} />
          <DocProgress doc={d} />
          {linked && (
            <Anchor component={Link} to={`/sales/${d.order ? 'orders' : 'quotes'}/${linked.id}`} size="sm">
              {t(d.order ? 'sales.quote.order_link' : 'sales.order.quote_link', { number: linked.number })}
            </Anchor>
          )}
        </Group>
      }
      error={error}
      actions={
        (
          <Group gap="xs">
            <ApprovalDecision docType={docType} id={d.id} onError={setError} />
            {d.allowed_actions.includes('print') && (
              <ExportButton kind={`printing.${docType}`} params={{ id: d.id }} label={t(`sales.${kind}.print`)} icon={IconPrinter} />
            )}
            {d.allowed_actions.includes('create_order') && (
              <Button variant="default" leftSection={<IconShoppingCartPlus {...icon.button} />} loading={ordering} onClick={() => void makeOrder()}>
                {t('sales.quote.create_order')}
              </Button>
            )}
            <DocumentActions
              docType={docType}
              id={d.id}
              version={d.version}
              number={d.number}
              actions={d.allowed_actions}
              onError={setError}
              listPath={list}
            />
          </Group>
        )
      }
      approval={<ApprovalPanel docType={docType} id={d.id} decideElsewhere />}
      sections={[attachments]}
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
      activity={<ActivityFeed docType={docType} id={d.id} />}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <DocForm key={d.version} kind={kind} doc={d} />
    </DocumentPage>
  )
}
