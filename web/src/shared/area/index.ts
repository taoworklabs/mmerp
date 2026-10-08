import type { QueryKey } from '@tanstack/react-query'
import type { ComponentType } from 'react'
import type { Locale } from '@/shared/i18n'

type Bundle = () => Promise<{ default: Record<string, string> }>

export type NavItem = {
  label: string // i18n key
  path: string // relative to the area's basePath
  icon: ComponentType<{ size?: number; stroke?: number }>
  permission?: string | string[] // shown only to users who have it (or one of them) somewhere
  group?: string // i18n key of the item's group; ungrouped items come first
}

// RecordTypeEntry is what an area tells the rest of the app about one of its record types.
export type RecordTypeEntry = {
  path: (id: number) => string // relative to the area's basePath
  // Read-only quick view for the approval inbox; lazy.
  preview?: () => Promise<{ default: ComponentType<{ id: number }> }>
  // Query keys to refresh when a record of this type changes, wherever it changed.
  invalidate: (id: number) => QueryKey[]
}

// AreaManifest is the only thing app knows about an area.
export type AreaManifest = {
  product: string // 'core' is always on
  label: string // i18n key of the area's home tile and of the header title inside the area
  basePath: string
  routes?: () => Promise<{ default: ComponentType }> // lazy; renders the area's <Routes>
  nav: NavItem[]
  collapsibleGroups?: string[] // groups the user may fold; remembered per user
  // The area's tile on the home page; it opens the first nav item the user may see.
  homeIcon: ComponentType<{ size?: number; stroke?: number }>
  i18n: {
    meta: Record<Locale, Bundle> // nav labels; loaded at startup
    main?: Record<Locale, Bundle> // the rest; loaded with the routes
  }
  recordTypes?: Record<string, RecordTypeEntry>
}

export { RecordPreview, RecordTypesProvider, useInvalidateRecord, useRecordLink, useRecordLinks } from './records'
