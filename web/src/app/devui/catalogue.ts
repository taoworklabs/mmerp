// The catalogue's entries as data: the sidebar, the routes and each page's header and links read
// from here, and catalogue.test.ts checks it against the lint allowlist and the translations.

export type Layer = 'start' | 'foundations' | 'mantine' | 'mmerp' | 'patterns' | 'templates'

export const layers: Layer[] = ['start', 'foundations', 'mantine', 'mmerp', 'patterns', 'templates']

// An entry shows its component name as-is, or an i18n label. `path` is `<layer>/<slug>`.
export type Entry = {
  layer: Layer
  path: string
  name?: string
  label?: string
  // mmerp: where to import it from.
  source?: string
  // mantine: the @mantine/core names the page documents, when more than its name.
  covers?: readonly string[]
  // Paths of the entries a page links to: related components, or what a pattern or template uses.
  related?: readonly string[]
}

export const entries = [
  { layer: 'start', path: 'start/overview', label: 'shared.devui.overview' },
  { layer: 'foundations', path: 'foundations/principles', label: 'shared.devui.principles' },
  { layer: 'foundations', path: 'foundations/colour', label: 'shared.devui.colour' },
  { layer: 'foundations', path: 'foundations/typography', label: 'shared.devui.typography' },
  { layer: 'foundations', path: 'foundations/spacing', label: 'shared.devui.spacing' },
  { layer: 'foundations', path: 'foundations/surfaces', label: 'shared.devui.surfaces' },
  { layer: 'foundations', path: 'foundations/icons', label: 'shared.devui.icons' },
  { layer: 'mantine', path: 'mantine/anchor', name: 'Anchor' },
  { layer: 'mantine', path: 'mantine/button', name: 'Button' },
  { layer: 'mantine', path: 'mantine/layout-text', name: 'Group · Stack · Divider · List · Text', covers: ['Group', 'Stack', 'Divider', 'List', 'Text'] },
  { layer: 'mantine', path: 'mantine/loader', name: 'Loader' },
  { layer: 'mantine', path: 'mantine/skeleton', name: 'Skeleton' },
  { layer: 'mantine', path: 'mantine/tooltip', name: 'Tooltip' },
  { layer: 'mmerp', path: 'mmerp/activity-feed', name: 'ActivityFeed', source: '@/shared/document' },
  { layer: 'mmerp', path: 'mmerp/approval-panel', name: 'ApprovalPanel', source: '@/shared/document' },
  { layer: 'mmerp', path: 'mmerp/bulk-bar', name: 'BulkBar', source: '@/shared/ui/BulkBar' },
  { layer: 'mmerp', path: 'mmerp/cell-grid', name: 'CellGrid', source: '@/shared/ui/CellGrid' },
  { layer: 'mmerp', path: 'mmerp/data-table', name: 'DataTable', source: '@/shared/ui/DataTable' },
  { layer: 'mmerp', path: 'mmerp/document-preview', name: 'DocumentPreview', source: '@/shared/document' },
  { layer: 'mmerp', path: 'mmerp/document-status', name: 'DocumentStatus', source: '@/shared/document' },
  { layer: 'mmerp', path: 'mmerp/editable-panel', name: 'EditablePanel', source: '@/shared/ui/form' },
  { layer: 'mmerp', path: 'mmerp/field-list', name: 'FieldList', source: '@/shared/ui/FieldList' },
  { layer: 'mmerp', path: 'mmerp/filters', name: 'Filters', source: '@/shared/ui/form' },
  { layer: 'mmerp', path: 'mmerp/form', name: 'Form · fields', source: '@/shared/ui/form' },
  { layer: 'mmerp', path: 'mmerp/form-modal', name: 'FormModal', source: '@/shared/ui/page' },
  { layer: 'mmerp', path: 'mmerp/inline-edit-field', name: 'InlineEditField', source: '@/shared/ui/InlineEditField' },
  { layer: 'mmerp', path: 'mmerp/key-facts', name: 'KeyFacts', source: '@/shared/ui/KeyFacts' },
  { layer: 'mmerp', path: 'mmerp/lines-table', name: 'LinesTable', source: '@/shared/ui/form' },
  { layer: 'mmerp', path: 'mmerp/more-actions', name: 'MoreActions', source: '@/shared/ui/MoreActions' },
  { layer: 'mmerp', path: 'mmerp/notice', name: 'Notice', source: '@/shared/ui/Notice' },
  { layer: 'mmerp', path: 'mmerp/notify', name: 'notify', source: '@/shared/ui/notify' },
  { layer: 'mmerp', path: 'mmerp/panel', name: 'Panel', source: '@/shared/ui/page' },
  { layer: 'mmerp', path: 'mmerp/person-avatar', name: 'PersonAvatar', source: '@/shared/ui/avatar' },
  { layer: 'mmerp', path: 'mmerp/query-view', name: 'QueryView', source: '@/shared/ui/QueryView' },
  { layer: 'mmerp', path: 'mmerp/section', name: 'Section', source: '@/shared/ui/Section' },
  { layer: 'mmerp', path: 'mmerp/side-panel', name: 'SidePanel', source: '@/shared/ui/SidePanel' },
  { layer: 'mmerp', path: 'mmerp/states', name: 'States', source: '@/shared/ui/states' },
  { layer: 'mmerp', path: 'mmerp/status-badge', name: 'StatusBadge', source: '@/shared/ui/StatusBadge' },
  { layer: 'mmerp', path: 'mmerp/tiles', name: 'Tiles', source: '@/shared/ui/Tile' },
  { layer: 'mmerp', path: 'mmerp/tree-list', name: 'TreeList', source: '@/shared/ui/TreeList' },
  { layer: 'mmerp', path: 'mmerp/use-action', name: 'useAction', source: '@/shared/ui/useAction' },
  { layer: 'mmerp', path: 'mmerp/attachment-panel', name: 'useAttachmentSection', source: '@/shared/document' },
  { layer: 'mmerp', path: 'mmerp/use-confirm', name: 'useConfirm', source: '@/shared/ui/confirm' },
  { layer: 'patterns', path: 'patterns/filter-search', label: 'shared.devui.pattern_filter_search' },
  { layer: 'patterns', path: 'patterns/create-edit', label: 'shared.devui.pattern_create_edit' },
  { layer: 'patterns', path: 'patterns/confirm-delete', label: 'shared.devui.pattern_confirm_delete' },
  { layer: 'patterns', path: 'patterns/loading-empty-error', label: 'shared.devui.pattern_loading_empty_error' },
  { layer: 'patterns', path: 'patterns/bulk', label: 'shared.devui.pattern_bulk' },
  { layer: 'patterns', path: 'patterns/document-approval', label: 'shared.devui.pattern_document_approval' },
  { layer: 'templates', path: 'templates/auth', name: 'AuthPage', source: '@/shared/ui/page' },
  { layer: 'templates', path: 'templates/document', name: 'DocumentPage', source: '@/shared/ui/page' },
  { layer: 'templates', path: 'templates/inbox', name: 'InboxPage', source: '@/shared/ui/page' },
  { layer: 'templates', path: 'templates/list', name: 'ListPage', source: '@/shared/ui/page' },
  { layer: 'templates', path: 'templates/page', name: 'Page', source: '@/shared/ui/page' },
  { layer: 'templates', path: 'templates/record', name: 'RecordPage', source: '@/shared/ui/page' },
] as const satisfies readonly Entry[]

export type Path = (typeof entries)[number]['path']

export const entryByPath = (path: string): Entry => {
  const e = (entries as readonly Entry[]).find((x) => x.path === path)
  if (!e) throw new Error(`devui: no entry ${path}`)
  return e
}
