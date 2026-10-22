export type Tone = 'default' | 'dimmed' | 'danger' | 'success' | 'warning' | 'primary'

// `default` keeps the fallback colour (plain text, or a primitive's own brand colour);
// `dimmed` reads the dimmed token; every other tone reads its ramp's "-light-color" step.
export function toneColor(tone: Tone | undefined, fallback: string): string {
  if (tone === undefined) return fallback
  if (tone === 'default') return 'var(--mmerp-color-text)'
  if (tone === 'dimmed') return 'var(--mmerp-color-dimmed)'
  return `var(--mmerp-color-${tone}-light-color)`
}
