import { Group } from '@mantine/core'
import type { UseQueryResult } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ActivityFeed, ApprovalPanel, DocumentPreview, DocumentStatus, type DocStatus, useAttachmentSection } from '@/shared/document'
import { sample, SampleDocument } from '@/shared/document/samples'
import { Panel, SeeAll } from '@/shared/ui/page'
import { Case } from './Demo'
import { DocPage } from './DocPage'

// The main props of each component; data, not interface text.
const code = {
  activity: `<ActivityFeed docType id recent? />   // recent={5} in the side column, with SeeAll`,
  approval: `<ApprovalPanel docType id />`,
  attachments: `const section = useAttachmentSection(docType, id)
<Panel title={section.title}>{section.content}</Panel>`,
  preview: `<DocumentPreview docType query rows={(d) => [[label, value], …]} />`,
  status: `<DocumentStatus docType status="draft" | "pending_approval" | "posted" | "cancelled" />`,
}

export function ActivityFeedDoc() {
  const { t } = useTranslation()
  return (
    <SampleDocument>
      <DocPage
        path="mmerp/activity-feed"
        example={
          <Case>
            <ActivityFeed docType={sample.type} id={sample.id} />
          </Case>
        }
        variants={
          <Case label={t('shared.devui.case_activity_recent')}>
            <Panel plain title={t('shared.record.recent_activity')} actions={<SeeAll tab="activity" />}>
              <ActivityFeed docType={sample.type} id={sample.id} recent={5} />
            </Panel>
          </Case>
        }
        code={code.activity}
      />
    </SampleDocument>
  )
}

export function ApprovalPanelDoc() {
  return (
    <SampleDocument>
      <DocPage
        path="mmerp/approval-panel"
        example={
          <Case>
            <ApprovalPanel docType={sample.type} id={sample.id} />
          </Case>
        }
        code={code.approval}
      />
    </SampleDocument>
  )
}

function Attachments() {
  const section = useAttachmentSection(sample.type, sample.id)
  return <Panel title={section.title}>{section.content}</Panel>
}

export function AttachmentPanelDoc() {
  return (
    <SampleDocument>
      <DocPage
        path="mmerp/attachment-panel"
        example={
          <Case>
            <Attachments />
          </Case>
        }
        code={code.attachments}
      />
    </SampleDocument>
  )
}

// A leave request as an inbox or side panel previews it; the note is left out when empty.
type Leave = { status: DocStatus; employee: string; period: string; note: string }
const leave = { isPending: false, isError: false, data: { status: 'pending_approval', employee: 'NV001 · Nguyễn Văn An', period: '12/10/2026 – 16/10/2026', note: '' } } as UseQueryResult<Leave>

export function DocumentPreviewDoc() {
  return (
    <SampleDocument>
      <DocPage
        path="mmerp/document-preview"
        example={
          <Case>
            <DocumentPreview docType={sample.type} query={leave} rows={(d) => [['Nhân viên', d.employee], ['Thời gian nghỉ', d.period], d.note && ['Ghi chú', d.note]]} />
          </Case>
        }
        code={code.preview}
      />
    </SampleDocument>
  )
}

const docStatuses: DocStatus[] = ['draft', 'pending_approval', 'posted', 'cancelled']

export function DocumentStatusDoc() {
  return (
    <DocPage
      path="mmerp/document-status"
      example={
        <Case>
          <Group>
            {docStatuses.map((s) => (
              <DocumentStatus key={s} docType="shared" status={s} />
            ))}
          </Group>
        </Case>
      }
      code={code.status}
    />
  )
}
