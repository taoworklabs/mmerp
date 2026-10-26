import type { StyleVars } from '../../internal/style.ts'
import type { ColorName } from '../../tokens/tokens.ts'

export interface TableAppearance {
  borders?: 'rows' | 'columns' | 'grid' | 'none'
  withBorder?: boolean
  borderColor?: ColorName
  borderStyle?: 'solid' | 'dashed' | 'dotted'
  borderWidth?: 1 | 2
  headerColor?: ColorName | 'neutral' | 'plain'
  striped?: boolean
  stripeColor?: ColorName
  hoverColor?: ColorName
  density?: 'compact' | 'standard' | 'comfortable'
}

export function tableAppearanceVars(appearance: TableAppearance = {}, framed = false): StyleVars {
  const { borders = 'rows', withBorder = framed, borderColor, borderStyle = 'solid', borderWidth = 1,
    headerColor = 'neutral', stripeColor = 'gray', hoverColor } = appearance
  const border = `${borderWidth}px ${borderStyle} ${borderColor ? `var(--mmerp-color-${borderColor}-3)` : 'var(--mmerp-color-border)'}`
  return {
    '--mmerp-table-frame': withBorder ? border : 'none',
    '--mmerp-table-row-rule': borders === 'rows' || borders === 'grid' ? border : 'none',
    '--mmerp-table-column-rule': borders === 'columns' || borders === 'grid' ? border : 'none',
    '--mmerp-table-rule-color': borders === 'none' ? 'transparent' : borderColor ? `var(--mmerp-color-${borderColor}-3)` : 'var(--mmerp-color-border)',
    '--mmerp-table-header-bg': headerColor === 'neutral' ? 'var(--mmerp-color-table-header)' : headerColor === 'plain' ? 'var(--mmerp-color-body)' : `var(--mmerp-color-${headerColor}-light)`,
    '--mmerp-table-header-text': headerColor === 'neutral' || headerColor === 'plain' ? 'var(--mmerp-color-dimmed)' : `var(--mmerp-color-${headerColor}-light-color)`,
    '--mmerp-table-stripe-bg': `var(--mmerp-color-${stripeColor}-0)`,
    '--mmerp-table-hover-bg': hoverColor ? `var(--mmerp-color-${hoverColor}-1)` : 'var(--mmerp-color-elevation-surface-sunken)',
  }
}
