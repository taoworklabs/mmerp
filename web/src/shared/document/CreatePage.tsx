import type { UseQueryResult } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { useInvalidateRecord, useRecordLinks } from '@/shared/area'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page, type Crumb } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'

type Gate = Pick<UseQueryResult<string[]>, 'isPending' | 'isError' | 'data'>

// createGate says what a create page shows for the actions the API allows before a record exists.
export function createGate(actions: Gate | string[], action: string): 'pending' | 'error' | 'forbidden' | 'ok' {
  if (!Array.isArray(actions)) {
    if (actions.isPending) return 'pending'
    if (actions.isError || !actions.data) return 'error'
    actions = actions.data
  }
  return actions.includes(action) ? 'ok' : 'forbidden'
}

type Props = {
  docType: string
  // The type's actions before a record exists, as a query or already loaded.
  actions: UseQueryResult<string[]> | string[]
  // The action that opens the form; create by default.
  action?: string
  title: string
  breadcrumbs: Crumb[]
  description: string
  // The success toast once created.
  created: string
  // The form, given what to do once it created a record, and the allowed actions.
  // Where to go once created, when not the area's page for the record.
  openAt?: (id: number) => string
  children: (onCreated: (id: number) => Promise<void>, actions: string[]) => ReactNode
}

// CreatePage is the page of a new document: forbidden without the action, then the form; once
// created it refreshes the record, says so and opens it.
export function CreatePage({ docType, actions, action = 'create', title, breadcrumbs, description, created, openAt, children }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const link = useRecordLinks()
  const gate = createGate(actions, action)
  if (gate === 'pending') return <PageSkeleton />
  if (gate === 'error' && !Array.isArray(actions)) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (gate !== 'ok') return <ForbiddenPage />
  const allowed = Array.isArray(actions) ? actions : actions.data!
  const onCreated = async (id: number) => {
    await invalidate(docType, id)
    notifySuccess(created)
    const to = openAt ? openAt(id) : link(docType, id)
    // The form is clean after a successful save, so leaving is not blocked.
    if (to) setTimeout(() => navigate(to, { replace: true }))
  }
  return (
    <Page centered title={title} breadcrumbs={breadcrumbs} description={description}>
      {children(onCreated, allowed)}
    </Page>
  )
}
