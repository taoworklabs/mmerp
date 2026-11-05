import type { ColorName } from '../tokens/tokens.ts'

// Colour as meaning. A component offering fewer tones takes a subset of this union.
export type Tone = 'neutral' | 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'discovery'

// The neutral tone reads the gray ramp; every other tone reads the ramp of its own name.
export function ramp(tone: Tone): ColorName {
  return tone === 'neutral' ? 'gray' : tone
}

// Text colour for a tone: none keeps the fallback, `neutral` is the main text colour, `dimmed`
// reads the dimmed token, every other tone reads its ramp's "-light-color" step.
export function toneColor(tone: Tone | 'dimmed' | undefined, fallback: string): string {
  if (tone === undefined) return fallback
  if (tone === 'neutral') return 'var(--mmerp-color-text)'
  if (tone === 'dimmed') return 'var(--mmerp-color-dimmed)'
  return `var(--mmerp-color-${tone}-light-color)`
}
