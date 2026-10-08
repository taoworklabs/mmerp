import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import type { AttachmentList, Discussion } from '@/shared/api/core'
import type { DocumentSection } from '@/shared/ui/page'
import { useAttachmentSection } from './AttachmentPanel'
import { useDiscussionSection } from './DiscussionPanel'
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

// Who the comment box offers after @.
const mentionable = [
  { login: 'lan.nguyen', name: 'Nguyễn Thị Lan' },
  { login: 'an', name: 'Nguyễn Văn An' },
]

// The panels read sample data from a cache of their own, filled once and never refetched.
const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity, retry: false } } })
client.setQueryData(documentKeys.attachments(sample.type, sample.id), attachments)
client.setQueryData(documentKeys.discussion(sample.type, sample.id), discussion)
client.setQueryData(documentKeys.mentionable(sample.type, sample.id), mentionable)

// WithSampleSections renders the attachment and discussion panels on sample data, for /dev/ui.
export function WithSampleSections({ children }: { children: (sections: DocumentSection[]) => ReactNode }) {
  return (
    <QueryClientProvider client={client}>
      <Sections>{children}</Sections>
    </QueryClientProvider>
  )
}

function Sections({ children }: { children: (sections: DocumentSection[]) => ReactNode }) {
  return children([useAttachmentSection(sample.type, sample.id), useDiscussionSection(sample.type, sample.id)])
}
