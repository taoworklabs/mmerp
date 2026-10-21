import type { Meta, StoryObj } from '@storybook/react-vite'
import { tokens } from './tokens'

const meta: Meta = {
  title: 'Foundations/Tokens',
}

export default meta

type Story = StoryObj

function Swatch({ name, value }: { name: string; value: string }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4, width: 160 }}>
      <div style={{ height: 48, borderRadius: tokens.radius.sm, background: value }} />
      <code style={{ fontSize: tokens.fontSize.xs }}>{name}</code>
    </div>
  )
}

export const Colours: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space.lg }}>
      {Object.entries(tokens.color.ramp).map(([ramp, steps]) => (
        <div key={ramp} style={{ display: 'flex', flexWrap: 'wrap', gap: tokens.space.xs }}>
          {steps.map((value, step) => (
            <Swatch key={step} name={`--mmerp-color-${ramp}-${step}`} value={value} />
          ))}
        </div>
      ))}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: tokens.space.xs }}>
        {Object.entries(tokens.color.semantic).map(([name, value]) => (
          <Swatch key={name} name={`--mmerp-color-${name}`} value={value} />
        ))}
      </div>
    </div>
  ),
}

export const Type: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space.sm }}>
      {Object.entries(tokens.fontSize).map(([key, size]) => (
        <div key={key} style={{ fontSize: size, lineHeight: tokens.lineHeight[key as keyof typeof tokens.lineHeight] }}>
          {`--mmerp-font-size-${key}: ${size}px`}
        </div>
      ))}
      {Object.entries(tokens.heading).map(([key, heading]) => (
        <div key={key} style={{ fontSize: heading.size, lineHeight: heading.lineHeight, fontWeight: tokens.fontWeight.heading }}>
          {`--mmerp-heading-${key}-size: ${heading.size}px`}
        </div>
      ))}
    </div>
  ),
}

export const Space: Story = {
  render: () => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: tokens.space.xs }}>
      {Object.entries(tokens.space).map(([key, size]) => (
        <div key={key} style={{ display: 'flex', alignItems: 'center', gap: tokens.space.xs }}>
          <div style={{ width: size, height: size, background: tokens.color.ramp.primary[6] }} />
          <code style={{ fontSize: tokens.fontSize.xs }}>{`--mmerp-space-${key}: ${size}px`}</code>
        </div>
      ))}
    </div>
  ),
}

export const RadiusStory: Story = {
  name: 'Radius',
  render: () => (
    <div style={{ display: 'flex', gap: tokens.space.sm }}>
      {Object.entries(tokens.radius).map(([key, size]) => (
        <div key={key} style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'center' }}>
          <div style={{ width: 48, height: 48, borderRadius: size, background: tokens.color.ramp.gray[2] }} />
          <code style={{ fontSize: tokens.fontSize.xs }}>{`--mmerp-radius-${key}`}</code>
        </div>
      ))}
    </div>
  ),
}

export const ShadowStory: Story = {
  name: 'Shadow',
  render: () => (
    <div style={{ display: 'flex', gap: tokens.space.lg, padding: tokens.space.lg, background: tokens.color.ramp.gray[0] }}>
      {Object.entries(tokens.shadow).map(([key, value]) => (
        <div key={key} style={{ display: 'flex', flexDirection: 'column', gap: 4, alignItems: 'center' }}>
          <div style={{ width: 64, height: 64, borderRadius: tokens.radius.md, background: tokens.color.semantic.body, boxShadow: value }} />
          <code style={{ fontSize: tokens.fontSize.xs }}>{`--mmerp-shadow-${key}`}</code>
        </div>
      ))}
    </div>
  ),
}
