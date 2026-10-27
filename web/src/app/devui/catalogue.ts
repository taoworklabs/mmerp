// The catalogue's entries as data: the sidebar, the routes and each page's header and links read
// from here, and catalogue.test.ts checks it against the lint allowlist and the translations.

export type Layer = 'start' | 'foundations' | 'kit' | 'mmerp' | 'patterns' | 'templates'

export const layers: Layer[] = ['start', 'foundations', 'kit', 'mmerp', 'patterns', 'templates']

// An entry shows its component name as-is, or an i18n label. `path` is `<layer>/<slug>`.
export type Entry = {
  layer: Layer
  path: string
  name?: string
  label?: string
  // mmerp: where to import it from, and the exports to import when the page covers a family.
  source?: string
  exports?: readonly string[]
  // kit: the @mmerp/ui names the page documents, when more than its name.
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
  { layer: 'kit', path: 'kit/anchor', name: 'Anchor', related: ['kit/button'] },
  { layer: 'kit', path: 'kit/button', name: 'Button', related: ['mmerp/more-actions', 'patterns/confirm-delete', 'kit/anchor'] },
  { layer: 'kit', path: 'kit/field-list', name: 'FieldList', related: ['kit/panel', 'kit/key-facts'] },
  { layer: 'kit', path: 'kit/layout-text', name: 'Group · Stack · Divider · List · Text', covers: ['Group', 'Stack', 'Divider', 'List', 'Text'], related: ['mmerp/section', 'kit/panel'] },
  { layer: 'kit', path: 'kit/key-facts', name: 'KeyFacts', related: ['templates/record', 'kit/field-list'] },
  { layer: 'kit', path: 'kit/loader', name: 'Loader', related: ['mmerp/states', 'kit/skeleton'] },
  { layer: 'kit', path: 'kit/panel', name: 'Panel', related: ['kit/field-list', 'templates/record'] },
  { layer: 'kit', path: 'kit/skeleton', name: 'Skeleton', related: ['mmerp/states', 'kit/loader'] },
  { layer: 'kit', path: 'kit/tooltip', name: 'Tooltip', related: ['kit/button'] },
  { layer: 'mmerp', path: 'mmerp/activity-feed', name: 'ActivityFeed', source: '@/shared/document', related: ['templates/record', 'templates/document'] },
  { layer: 'mmerp', path: 'mmerp/approval-panel', name: 'ApprovalPanel', source: '@/shared/document', related: ['patterns/document-approval', 'mmerp/document-status'] },
  { layer: 'mmerp', path: 'mmerp/bulk-bar', name: 'BulkBar', source: '@/shared/ui/BulkBar', related: ['patterns/bulk', 'mmerp/data-table'] },
  { layer: 'mmerp', path: 'mmerp/cell-grid', name: 'CellGrid', source: '@/shared/ui/CellGrid', related: ['mmerp/lines-table'] },
  { layer: 'mmerp', path: 'mmerp/data-table', name: 'DataTable', source: '@/shared/ui/DataTable', related: ['templates/list', 'patterns/filter-search', 'patterns/bulk', 'mmerp/query-view'] },
  { layer: 'mmerp', path: 'mmerp/document-preview', name: 'DocumentPreview', source: '@/shared/document', related: ['templates/inbox', 'mmerp/side-panel'] },
  { layer: 'mmerp', path: 'mmerp/document-status', name: 'DocumentStatus', source: '@/shared/document', related: ['mmerp/status-badge', 'patterns/document-approval'] },
  { layer: 'mmerp', path: 'mmerp/editable-panel', name: 'EditablePanel', source: '@/shared/ui/form', related: ['mmerp/inline-edit-field', 'patterns/create-edit', 'templates/record'] },
  { layer: 'mmerp', path: 'mmerp/filters', name: 'Filters', source: '@/shared/ui/form', exports: ['SearchInput', 'FilterChip', 'OrgUnitFilter', 'LegalEntityFilter', 'MonthFilter', 'YearFilter'], related: ['patterns/filter-search', 'templates/list'] },
  { layer: 'mmerp', path: 'mmerp/form', name: 'Form · fields', source: '@/shared/ui/form', exports: ['Form', 'FormSection', 'FormFields', 'FormActions', 'TextField'], related: ['mmerp/form-modal', 'mmerp/editable-panel', 'patterns/create-edit'] },
  { layer: 'mmerp', path: 'mmerp/form-modal', name: 'FormModal', source: '@/shared/ui/page', related: ['mmerp/form', 'patterns/create-edit', 'mmerp/use-confirm'] },
  { layer: 'mmerp', path: 'mmerp/inline-edit-field', name: 'InlineEditField', source: '@/shared/ui/InlineEditField', related: ['mmerp/editable-panel'] },
  { layer: 'mmerp', path: 'mmerp/lines-table', name: 'LinesTable', source: '@/shared/ui/form', related: ['mmerp/form', 'mmerp/cell-grid', 'templates/document'] },
  { layer: 'mmerp', path: 'mmerp/more-actions', name: 'MoreActions', source: '@/shared/ui/MoreActions', related: ['patterns/confirm-delete', 'kit/button'] },
  { layer: 'mmerp', path: 'mmerp/notice', name: 'Notice', source: '@/shared/ui/Notice', related: ['mmerp/notify', 'mmerp/use-action'] },
  { layer: 'mmerp', path: 'mmerp/notify', name: 'notify', source: '@/shared/ui/notify', exports: ['notifySuccess', 'notifyError', 'notifyUndo'], related: ['mmerp/notice', 'mmerp/use-action'] },
  { layer: 'mmerp', path: 'mmerp/person-avatar', name: 'PersonAvatar', source: '@/shared/ui/avatar', related: ['templates/record'] },
  { layer: 'mmerp', path: 'mmerp/query-view', name: 'QueryView', source: '@/shared/ui/QueryView', related: ['mmerp/states', 'patterns/loading-empty-error'] },
  { layer: 'mmerp', path: 'mmerp/section', name: 'Section', source: '@/shared/ui/Section', related: ['kit/panel', 'mmerp/form'] },
  { layer: 'mmerp', path: 'mmerp/side-panel', name: 'SidePanel', source: '@/shared/ui/SidePanel', related: ['patterns/bulk', 'mmerp/document-preview'] },
  { layer: 'mmerp', path: 'mmerp/states', name: 'States', source: '@/shared/ui/states', exports: ['EmptyState', 'ErrorState', 'PageSkeleton', 'ContentSkeleton', 'ReadOnlyBanner', 'NotFoundPage', 'ForbiddenPage'], related: ['mmerp/query-view', 'patterns/loading-empty-error'] },
  { layer: 'mmerp', path: 'mmerp/status-badge', name: 'StatusBadge', source: '@/shared/ui/StatusBadge', related: ['mmerp/document-status'] },
  { layer: 'mmerp', path: 'mmerp/tiles', name: 'Tiles', source: '@/shared/ui/Tile', exports: ['TileGrid', 'StatTile', 'LinkTile'], related: ['templates/page'] },
  { layer: 'mmerp', path: 'mmerp/tree-list', name: 'TreeList', source: '@/shared/ui/TreeList', related: ['mmerp/data-table'] },
  { layer: 'mmerp', path: 'mmerp/use-action', name: 'useAction', source: '@/shared/ui/useAction', related: ['mmerp/use-confirm', 'patterns/confirm-delete'] },
  { layer: 'mmerp', path: 'mmerp/attachment-panel', name: 'useAttachmentSection', source: '@/shared/document', related: ['templates/document'] },
  { layer: 'mmerp', path: 'mmerp/use-confirm', name: 'useConfirm', source: '@/shared/ui/confirm', related: ['mmerp/use-action', 'patterns/confirm-delete'] },
  { layer: 'patterns', path: 'patterns/filter-search', label: 'shared.devui.pattern_filter_search', related: ['mmerp/filters', 'mmerp/data-table', 'mmerp/states', 'templates/list'] },
  { layer: 'patterns', path: 'patterns/create-edit', label: 'shared.devui.pattern_create_edit', related: ['mmerp/form', 'mmerp/form-modal', 'mmerp/editable-panel'] },
  { layer: 'patterns', path: 'patterns/confirm-delete', label: 'shared.devui.pattern_confirm_delete', related: ['mmerp/more-actions', 'mmerp/use-action', 'mmerp/use-confirm', 'mmerp/notice'] },
  { layer: 'patterns', path: 'patterns/loading-empty-error', label: 'shared.devui.pattern_loading_empty_error', related: ['mmerp/query-view', 'mmerp/states'] },
  { layer: 'patterns', path: 'patterns/bulk', label: 'shared.devui.pattern_bulk', related: ['mmerp/data-table', 'mmerp/bulk-bar', 'mmerp/side-panel'] },
  { layer: 'patterns', path: 'patterns/document-approval', label: 'shared.devui.pattern_document_approval', related: ['mmerp/document-status', 'mmerp/approval-panel', 'mmerp/activity-feed', 'templates/document'] },
  { layer: 'templates', path: 'templates/auth', name: 'AuthPage', source: '@/shared/ui/page', related: ['kit/button'] },
  { layer: 'templates', path: 'templates/document', name: 'DocumentPage', source: '@/shared/ui/page', related: ['mmerp/lines-table', 'mmerp/document-status', 'mmerp/approval-panel', 'mmerp/attachment-panel', 'mmerp/activity-feed'] },
  { layer: 'templates', path: 'templates/inbox', name: 'InboxPage', source: '@/shared/ui/page', related: ['mmerp/document-preview', 'mmerp/states'] },
  { layer: 'templates', path: 'templates/list', name: 'ListPage', source: '@/shared/ui/page', related: ['mmerp/data-table', 'mmerp/filters', 'mmerp/bulk-bar', 'mmerp/side-panel', 'mmerp/query-view'] },
  { layer: 'templates', path: 'templates/page', name: 'Page', source: '@/shared/ui/page', related: ['kit/button', 'kit/key-facts', 'mmerp/person-avatar', 'mmerp/status-badge'] },
  { layer: 'templates', path: 'templates/record', name: 'RecordPage', source: '@/shared/ui/page', related: ['kit/key-facts', 'mmerp/editable-panel', 'kit/field-list', 'kit/panel', 'mmerp/activity-feed'] },
] as const satisfies readonly Entry[]

export type Path = (typeof entries)[number]['path']

export const entryByPath = (path: string): Entry => {
  const e = (entries as readonly Entry[]).find((x) => x.path === path)
  if (!e) throw new Error(`devui: no entry ${path}`)
  return e
}

// key names an entry in i18n keys: the last path segment, `-` as `_` (mmerp/data-table → data_table).
export const key = (entry: Entry) => entry.path.slice(entry.path.indexOf('/') + 1).replaceAll('-', '_')

// Entries whose name is one identifier, longest first, so a text can link the components it names.
const named = (entries as readonly Entry[]).filter((e) => e.name && /^\w+$/.test(e.name)).sort((a, b) => b.name!.length - a.name!.length)
const nameRe = new RegExp(`\\b(${named.map((e) => e.name).join('|')})\\b`, 'g')

// splitNames cuts a text at every catalogued component name, giving that name's entry in its place.
export function splitNames(text: string): (string | Entry)[] {
  const out: (string | Entry)[] = []
  let at = 0
  for (const m of text.matchAll(nameRe)) {
    if (m.index > at) out.push(text.slice(at, m.index))
    out.push(named.find((e) => e.name === m[0])!)
    at = m.index + m[0].length
  }
  if (at < text.length) out.push(text.slice(at))
  return out
}
