import { Anchor, Button, Group } from '@mantine/core'
import { IconPrinter, IconShoppingCartPlus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/sales'
import { ApprovalPanel, DocumentActions, DocumentHistory, DocumentStatus, useAttachmentSection, useDiscussionSection } from '@/shared/document'
import { errorText } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs'
import { loaded } from '@/shared/ui/loaded'
import { notifySuccess } from '@/shared/ui/notify'
import { DocumentPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { DocForm } from '../components/DocForm'
import { DocProgress } from '../components/DocProgress'
import { useDoc } from '../hooks/useSales'
import { docTypeOf, pathOf, salesKeys, type Kind } from '../keys'

// DocPage shows a quotation or order; a draft the user may edit is edited in place.
export function DocPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const id = Number(useParams().id)
  const docType = docTypeOf(kind)
  const list = `/sales/${pathOf(kind)}`
  const doc = useDoc(kind, id)
  const attachments = useAttachmentSection(docType, id)
  const discussion = useDiscussionSection(docType, id)
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
      await qc.invalidateQueries({ queryKey: salesKeys.docs.all('quote') })
      await qc.invalidateQueries({ queryKey: salesKeys.docs.all('order') })
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
        d.allowed_actions.length > 0 && (
          <Group gap="xs">
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
              onDelete={async (version) => {
                await unwrap(
                  kind === 'quote'
                    ? api.DELETE('/sales/quotes/{id}', { params: { path: { id: d.id }, query: { version } } })
                    : api.DELETE('/sales/orders/{id}', { params: { path: { id: d.id }, query: { version } } }),
                )
                await qc.invalidateQueries({ queryKey: salesKeys.docs.all(kind) })
                navigate(list, { replace: true })
              }}
            />
          </Group>
        )
      }
      approval={<ApprovalPanel docType={docType} id={d.id} />}
      sections={[attachments, discussion]}
      history={<DocumentHistory docType={docType} id={d.id} />}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <DocForm key={d.version} kind={kind} doc={d} />
    </DocumentPage>
  )
}
