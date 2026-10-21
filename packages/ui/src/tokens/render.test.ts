import { describe, expect, it } from 'vitest'
import generated from './tokens.css?raw'
import { renderTokensCss } from './render'

describe('renderTokensCss', () => {
  const css = renderTokensCss()
  it('emits ramps, semantic colours and tints', () => {
    expect(css).toContain('--mmerp-color-primary-6: #1868DB;')
    expect(css).toContain('--mmerp-color-gray-0: #F8F8F8;')
    expect(css).toContain('--mmerp-color-text: #292A2E;')
    expect(css).toContain('--mmerp-color-dimmed: #505258;')
    expect(css).toContain('--mmerp-color-border: #0B120E24;')
    expect(css).toContain('--mmerp-color-error: #AE2E24;')
    expect(css).toContain('--mmerp-color-danger-light: #FFECEB;')
    expect(css).toContain('--mmerp-color-danger-light-color: #AE2E24;')
    expect(css).toContain('--mmerp-color-gray-light: #F0F1F2;') // neutral tint starts one step darker
  })
  it('emits decorative colours', () => {
    expect(css).toContain('--mmerp-color-teal-light: #12B8861A;')
    expect(css).toContain('--mmerp-color-teal-light-color: #12B886;')
  })
  it('emits sizes in rem from px', () => {
    expect(css).toContain('--mmerp-control-height: 1.875rem;')
    expect(css).toContain('--mmerp-control-height-lg: 2.5rem;')
    expect(css).toContain('--mmerp-space-md: 1rem;')
    expect(css).toContain('--mmerp-radius-lg: 0.5rem;')
    expect(css).toContain('--mmerp-font-size-xl: 1.25rem;')
    expect(css).toContain('--mmerp-content-width: 75rem;')
  })
  it('keeps breakpoints and icon sizes out of CSS', () => {
    expect(css).not.toContain('breakpoint')
    expect(css).not.toContain('--mmerp-icon')
  })
  it('matches the committed tokens.css', () => {
    expect(generated).toBe(css)
  })
})
