import { Group } from '@mmerp/ui'
import { Fragment, useState, type ReactNode } from 'react'
import { DocumentPage, type Crumb, type DocumentSection } from '@/shared/ui/page'
import { ActivityFeed } from './ActivityFeed'
import { ApprovalDecision, ApprovalPanel } from './ApprovalPanel'
import { useAttachmentSection } from './AttachmentPanel'
import { DocumentActions } from './DocumentActions'
import { DocumentStatus, type DocStatus } from './DocumentStatus'

type Doc = { id: number; version: number; number: string; status: DocStatus; allowed_actions: string[] }

type Props = {
  docType: string
  doc: Doc
  // The document's list: the breadcrumb, and where a deleted draft leaves for.
  list: Crumb
  // Key facts in the header, joined by "·"; falsy ones are left out.
  facts: ReactNode[]
  // Shown after the status (progress, a linked document).
  badges?: ReactNode
  // Whom the approval is about, when it is a person.
  subject?: string
  // The record type takes attachments.
  attachments?: boolean
  fullWidth?: boolean
  // The document's own buttons, before the lifecycle ones; a failure goes to the page's banner.
  extraActions?: (onError: (message: string | null) => void) => ReactNode
  // The lifecycle actions to draw; allowed_actions by default.
  actions?: string[]
  printLabel?: string
  printKind?: string
  children: ReactNode
}

// DocumentScreen is one document's page: number, facts and status in the header, approval
// decision and lifecycle buttons from allowed_actions, then approval, attachments and activity.
export function DocumentScreen(props: Props) {
  return props.attachments ? <WithAttachments {...props} /> : <Screen {...props} sections={[]} />
}

function WithAttachments(props: Props) {
  const attachments = useAttachmentSection(props.docType, props.doc.id)
  return <Screen {...props} sections={[attachments]} />
}

function Screen({ docType, doc: d, list, facts, badges, subject, fullWidth, extraActions, actions, printLabel, printKind, sections, children }: Props & { sections: DocumentSection[] }) {
  const [error, setError] = useState<string | null>(null)
  return (
    <DocumentPage
      title={d.number}
      breadcrumbs={[list]}
      status={
        <Group gap="xs" wrap={false}>
          <DocumentStatus docType={docType} status={d.status} />
          {badges}
        </Group>
      }
      description={
        <Group gap="xs" render={<span />}>
          {facts
            .filter((f) => f !== null && f !== undefined && f !== false && f !== '')
            .map((f, i) => (
              <Fragment key={i}>
                {i > 0 && <span aria-hidden>·</span>}
                {typeof f === 'string' ? <span>{f}</span> : f}
              </Fragment>
            ))}
        </Group>
      }
      error={error}
      actions={
        <Group gap="xs" justify="end">
          <ApprovalDecision docType={docType} id={d.id} onError={setError} />
          {extraActions?.(setError)}
          <DocumentActions
            docType={docType}
            id={d.id}
            version={d.version}
            number={d.number}
            actions={actions ?? d.allowed_actions}
            onError={setError}
            listPath={list.to}
            printLabel={printLabel}
            printKind={printKind}
          />
        </Group>
      }
      approval={<ApprovalPanel docType={docType} id={d.id} subject={subject} decideElsewhere />}
      sections={sections}
      fullWidth={fullWidth}
      activity={<ActivityFeed docType={docType} id={d.id} />}
    >
      {children}
    </DocumentPage>
  )
}
