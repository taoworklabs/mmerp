import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { ApprovalInstance, AttachmentList, Discussion } from '@/shared/api/core'
import type { DocumentSection } from '@/shared/ui/page'
import { ApprovalPanel } from './ApprovalPanel'
import { useAttachmentSection } from './AttachmentPanel'
import { ActivityFeed } from './ActivityFeed'
import { documentKeys } from './keys'

const sample = { type: 'dev.sample', id: 1 }

const attachments: AttachmentList = {
  hidden: false,
  max_mb: 20,
  accept: ['.pdf', '.jpg', '.jpeg', '.png', '.docx', '.xlsx'],
  allowed_actions: ['attach'],
  items: [
    { id: 1, name: 'giay-kham-benh.pdf', size: 248_320, content_type: 'application/pdf', uploaded_by_name: 'Nguyễn Văn An', uploaded_at: '2026-03-09T08:15:00Z', allowed_actions: ['delete'] },
    { id: 2, name: 'bang-ban-giao-cong-viec.xlsx', size: 3_460_000, content_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', uploaded_by_name: 'Trần Thị Bình', uploaded_at: '2026-03-09T09:40:00Z', allowed_actions: [] },
  ],
}

const discussion: Discussion = {
  max_length: 4000,
  allowed_actions: ['comment'],
  items: [
    { id: 1, author_name: 'Trần Thị Bình', body: 'Ai làm thay việc của em trong hai ngày nghỉ?', created_at: '2026-03-09T10:02:00Z' },
    { id: 2, author_name: 'Nguyễn Văn An', body: 'Chị Lan nhận phần đối soát.\nEm đã gửi kèm bảng bàn giao.', created_at: '2026-03-09T10:20:00Z' },
    { id: 3, author_name: 'Trần Thị Bình', body: '@lan.nguyen chị xác nhận giúp em, rồi báo @an.', created_at: '2026-03-09T10:31:00Z' },
  ],
}

// A decided first step, then two parallel steps the viewer may vote on, so each carries its own decision.
const approval: ApprovalInstance = {
  id: 1,
  action: 'post',
  status: 'open',
  allowed_actions: [],
  submitted_by_name: 'Nguyễn Văn An',
  submitted_at: '2026-03-09T08:20:00Z',
  plan: { seq: [{ step: 1 }, { par: [{ step: 2 }, { step: 3 }] }] },
  steps: [
    {
      position: 1,
      status: 'approved',
      approvals: 1,
      rejections: 1,
      fallback: false,
      voters: [{ id: 2, name: 'Trần Thị Bình', decision: 'approved', cast_by_name: null, reason: null, decided_at: '2026-03-09T09:05:00Z' }],
      allowed_actions: [],
    },
    {
      position: 2,
      status: 'open',
      approvals: 2,
      rejections: 1,
      fallback: false,
      voters: [
        { id: 3, name: 'Nguyễn Thị Lan', cast_by_name: 'Phạm Minh Đức', decision: 'approved', reason: null, decided_at: '2026-03-09T10:40:00Z' },
        { id: 4, name: 'Lê Văn Cường', decision: null, cast_by_name: null, reason: null, decided_at: null },
      ],
      allowed_actions: ['approve', 'reject'],
    },
    {
      position: 3,
      status: 'open',
      approvals: 1,
      rejections: 1,
      fallback: false,
      voters: [{ id: 4, name: 'Lê Văn Cường', decision: null, cast_by_name: null, reason: null, decided_at: null }],
      allowed_actions: ['approve', 'reject'],
    },
  ],
}

// Who the comment box offers after @.
const mentionable = [
  { login: 'lan.nguyen', name: 'Nguyễn Thị Lan' },
  { login: 'an', name: 'Nguyễn Văn An' },
]

// The panels read sample data from a cache of their own, filled once, never refetched nor
// collected (an unused query is dropped after 5 minutes by default, then fetched from the API).
const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, gcTime: Infinity, retry: false } } })
client.setQueryData(documentKeys.attachments(sample.type, sample.id), attachments)
client.setQueryData(documentKeys.discussion(sample.type, sample.id), discussion)
client.setQueryData(documentKeys.mentionable(sample.type, sample.id), mentionable)
client.setQueryData(documentKeys.history(sample.type, sample.id), [])
client.setQueryData(documentKeys.approval(sample.type, sample.id), { instance: approval })

// WithSampleSections renders the approval panel, the attachment panel and the activity feed on sample data, for the component catalogue.
export function WithSampleSections({ children }: { children: (sections: DocumentSection[], activity: ReactNode, approval: ReactNode) => ReactNode }) {
  return (
    <QueryClientProvider client={client}>
      <Sections>{children}</Sections>
    </QueryClientProvider>
  )
}

function Sections({ children }: { children: (sections: DocumentSection[], activity: ReactNode, approval: ReactNode) => ReactNode }) {
  return children([useAttachmentSection(sample.type, sample.id)], <ActivityFeed docType={sample.type} id={sample.id} />, <ApprovalPanel docType={sample.type} id={sample.id} />)
}
