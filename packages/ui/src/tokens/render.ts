import { tokens, type ColorName } from './tokens.ts'

const rampNames: ColorName[] = ['primary', 'success', 'warning', 'danger', 'info', 'discovery', 'gray']

// Every px size becomes rem relative to a 16px root, except touch-target (stays px).
function rem(px: number): string {
  return `${px / 16}rem`
}

// filterWidth keys like searchMin need a CSS-friendly name.
function kebab(key: string): string {
  return key.replace(/[A-Z]/g, (letter) => `-${letter.toLowerCase()}`)
}

function colorLines(): string[] {
  const lines: string[] = []
  for (const name of rampNames) {
    const ramp = tokens.color.ramp[name]
    for (let step = 0; step < ramp.length; step++) {
      lines.push(`--mmerp-color-${name}-${step}: ${ramp[step]!};`)
    }
    const offset = tokens.color.tint[name]
    lines.push(`--mmerp-color-${name}-light: ${ramp[offset]!};`)
    lines.push(`--mmerp-color-${name}-light-hover: ${ramp[offset + 1]!};`)
    lines.push(`--mmerp-color-${name}-light-color: ${ramp[7]!};`)
  }
  for (const [key, value] of Object.entries(tokens.color.semantic)) {
    lines.push(`--mmerp-color-${key}: ${value};`)
  }
  for (const [name, { bg, text }] of Object.entries(tokens.color.decorative)) {
    lines.push(`--mmerp-color-${name}-light: ${bg};`)
    lines.push(`--mmerp-color-${name}-light-color: ${text};`)
  }
  return lines
}

function sizeLines(): string[] {
  const lines: string[] = []
  lines.push(`--mmerp-font-family: ${tokens.fontFamily};`)
  for (const [key, px] of Object.entries(tokens.fontSize)) {
    lines.push(`--mmerp-font-size-${key}: ${rem(px)};`)
  }
  for (const [key, value] of Object.entries(tokens.lineHeight)) {
    lines.push(`--mmerp-line-height-${key}: ${value};`)
  }
  for (const [key, value] of Object.entries(tokens.fontWeight)) {
    lines.push(`--mmerp-font-weight-${key}: ${value};`)
  }
  for (const [key, heading] of Object.entries(tokens.heading)) {
    lines.push(`--mmerp-heading-${key}-size: ${rem(heading.size)};`)
    lines.push(`--mmerp-heading-${key}-line-height: ${heading.lineHeight};`)
  }
  for (const [key, px] of Object.entries(tokens.space)) {
    lines.push(`--mmerp-space-${key}: ${rem(px)};`)
  }
  for (const [key, px] of Object.entries(tokens.radius)) {
    lines.push(`--mmerp-radius-${key}: ${rem(px)};`)
  }
  for (const [key, value] of Object.entries(tokens.shadow)) {
    lines.push(`--mmerp-shadow-${key}: ${value};`)
  }
  lines.push(`--mmerp-control-height: ${rem(tokens.controlHeight.sm)};`)
  lines.push(`--mmerp-control-height-lg: ${rem(tokens.controlHeight.lg)};`)
  lines.push(`--mmerp-content-width: ${rem(tokens.layoutWidth.content)};`)
  lines.push(`--mmerp-side-width: ${rem(tokens.layoutWidth.side)};`)
  lines.push(`--mmerp-aside-width: ${rem(tokens.layoutWidth.aside)};`)
  lines.push(`--mmerp-detail-width: ${rem(tokens.layoutWidth.detail)};`)
  for (const [key, px] of Object.entries(tokens.filterWidth)) {
    lines.push(`--mmerp-filter-width-${kebab(key)}: ${rem(px)};`)
  }
  lines.push(`--mmerp-motion-duration: ${tokens.motionDuration}ms;`)
  lines.push(`--mmerp-touch-target: ${tokens.touchTarget}px;`)
  return lines
}

export function renderTokensCss(): string {
  const lines = [...colorLines(), ...sizeLines()].map((line) => `  ${line}`).join('\n')
  return `/* Generated from tokens.ts by scripts/gen-tokens.ts. Do not edit. */\n:root {\n${lines}\n}\n`
}
