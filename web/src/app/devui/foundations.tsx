import { Box, Button, Code, Grid, Group, List, Paper, SimpleGrid, Stack, Text, TextInput, Title } from '@mmerp/ui'
import { useMantineTheme } from '@mantine/core'
import { IconCheck, IconInbox, IconPlus, IconX } from '@tabler/icons-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Section } from '@/shared/ui/Section'
import { icon } from '@/shared/ui/theme'
import { Demo } from './Demo'
import devui from './DevUi.module.css'

// Values are read from the live theme, so this page never drifts from theme.ts.
const px = (v: string) => {
  const m = /([\d.]+)rem/.exec(v)
  return m ? `${Number(m[1]) * 16}px` : v
}

// CssValue shows what a CSS variable resolves to once the theme has set it on the root.
function CssValue({ name }: { name: string }) {
  const [value, setValue] = useState('')
  useEffect(() => setValue(px(getComputedStyle(document.documentElement).getPropertyValue(name).trim())), [name])
  return <Code>{value}</Code>
}

// Spec is one token: what it looks like, its name and value, and what it is for.
function Spec({ sample, name, value, use }: { sample: ReactNode; name: string; value: ReactNode; use: string }) {
  return (
    <Grid gap="md" className={devui.spec}>
      <Grid.Col span={{ base: 12, sm: 4 }}>{sample}</Grid.Col>
      <Grid.Col span={{ base: 12, sm: 3 }}>
        <Stack gap="xxxs" align="start">
          <Code>{name}</Code>
          <Text size="xs" tone="dimmed">
            {value}
          </Text>
        </Stack>
      </Grid.Col>
      <Grid.Col span={{ base: 12, sm: 5 }}>
        <Text size="sm">{use}</Text>
      </Grid.Col>
    </Grid>
  )
}

const designPrinciples = ['dense', 'flat', 'colour', 'radius', 'one_way', 'backend']
const uxPrinciples = ['one_page', 'one_primary', 'undo', 'errors', 'url', 'server']
const voice = ['short', 'imperative', 'sentence', 'plain']
const examples = ['button', 'toast', 'field_error']

export function Principles() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.principles')} description={t('shared.devui.principles_hint')}>
      <Section title={t('shared.devui.principles_design')}>
        <List spacing="xs">
          {designPrinciples.map((k) => (
            <List.Item key={k}>{t(`shared.devui.p_${k}`)}</List.Item>
          ))}
        </List>
      </Section>
      <Section title={t('shared.devui.principles_ux')}>
        <List spacing="xs">
          {uxPrinciples.map((k) => (
            <List.Item key={k}>{t(`shared.devui.p_${k}`)}</List.Item>
          ))}
        </List>
      </Section>
      <Section title={t('shared.devui.voice')}>
        <List spacing="xs">
          {voice.map((k) => (
            <List.Item key={k}>{t(`shared.devui.voice_${k}`)}</List.Item>
          ))}
        </List>
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          {examples.flatMap((k) => [
            <Group key={`${k}-do`} gap="xs" wrap={false}>
              <IconCheck {...icon.text} color="var(--mmerp-color-success-6)" aria-label={t('shared.devui.do')} />
              <Text size="sm">{t(`shared.devui.example_${k}_do`)}</Text>
            </Group>,
            <Group key={`${k}-dont`} gap="xs" wrap={false}>
              <IconX {...icon.text} color="var(--mmerp-color-danger-6)" aria-label={t('shared.devui.dont')} />
              <Text size="sm" tone="dimmed">
                {t(`shared.devui.example_${k}_dont`)}
              </Text>
            </Group>,
          ])}
        </SimpleGrid>
      </Section>
    </Demo>
  )
}

const ramps = ['primary', 'success', 'warning', 'danger', 'discovery', 'gray']

// kind says how a semantic token shows: as text, as an edge, or as a fill.
const semantic: { name: string; kind: 'text' | 'border' | 'fill' }[] = [
  { name: '--mmerp-color-text', kind: 'text' },
  { name: '--mmerp-color-dimmed', kind: 'text' },
  { name: '--mmerp-color-placeholder', kind: 'text' },
  { name: '--mmerp-color-error', kind: 'text' },
  { name: '--mmerp-color-border', kind: 'border' },
  { name: '--mmerp-color-input-border', kind: 'border' },
  { name: '--mmerp-color-elevation-surface', kind: 'fill' },
  { name: '--mmerp-color-elevation-surface-sunken', kind: 'fill' },
  { name: '--mmerp-color-table-header', kind: 'fill' },
  { name: '--mmerp-color-disabled', kind: 'fill' },
  { name: '--mmerp-color-primary-light', kind: 'fill' },
]

// Swatch is a framed patch of one colour, so the palest steps still show on white.
function Swatch({ bg }: { bg: string }) {
  return <Patch background={bg} height="var(--mmerp-space-xl)" radius="var(--mmerp-radius-md)" border="1px solid var(--mmerp-color-border)" />
}

// Patch paints a sample with values chosen at run time, which no class can name.
function Patch({ background, width, height, radius, border }: { background?: string; width?: string; height: string; radius?: string; border?: string }) {
  // eslint-disable-next-line no-restricted-syntax -- the whole point of a sample is its run-time value
  return <Box style={{ background, width, height, borderRadius: radius, border }} />
}

export function Colour() {
  const { t } = useTranslation()
  const theme = useMantineTheme()
  return (
    <Demo title={t('shared.devui.colour')} description={t('shared.devui.colour_hint')}>
      <Section title={t('shared.devui.colour_ramps')} description={t('shared.devui.colour_steps')}>
        {ramps.map((name) => (
          <Stack key={name} gap="xxs">
            <Group gap="xs">
              <Code>{name}</Code>
              <Text size="sm" tone="dimmed">
                {t(`shared.devui.ramp_${name}`)}
              </Text>
            </Group>
            <SimpleGrid cols={{ base: 5, sm: 10 }} spacing="xxs">
              {theme.colors[name]!.map((hex, i) => (
                <Stack key={i} gap="xxxs">
                  <Swatch bg={`var(--mmerp-color-${name}-${i})`} />
                  <Text size="xs" tone="dimmed" truncate>
                    {`${i} · ${hex}`}
                  </Text>
                </Stack>
              ))}
            </SimpleGrid>
          </Stack>
        ))}
      </Section>
      <Section title={t('shared.devui.colour_semantic')}>
        {semantic.map(({ name, kind }) => (
          <Spec
            key={name}
            name={name}
            value={<CssValue name={name} />}
            use={t(`shared.devui.token${name.replaceAll('-', '_')}`)}
            sample={
              kind === 'text' ? (
                // eslint-disable-next-line no-restricted-syntax -- the sample is painted with the token under test
                <Text weight={600} style={{ color: `var(${name})` }}>
                  {t('shared.devui.type_sample')}
                </Text>
              ) : kind === 'border' ? (
                <Patch height="var(--mmerp-space-xl)" radius="var(--mmerp-radius-md)" border={`1px solid var(${name})`} />
              ) : (
                <Swatch bg={`var(${name})`} />
              )
            }
          />
        ))}
      </Section>
    </Demo>
  )
}

const textSizes = ['xs', 'sm', 'lg', 'xl'] as const
const headings = ['h1', 'h2', 'h3'] as const
const weights = [400, 500, 600, 650] as const

export function Typography() {
  const { t } = useTranslation()
  const theme = useMantineTheme()
  return (
    <Demo title={t('shared.devui.typography')} description={t('shared.devui.typography_hint')}>
      <Section title={t('shared.devui.type_headings')}>
        {headings.map((h, i) => (
          <Spec
            key={h}
            name={h}
            value={`${px(theme.headings.sizes[h].fontSize)} / ${theme.headings.sizes[h].lineHeight} · ${theme.headings.fontWeight}`}
            use={t(`shared.devui.type_${h}`)}
            sample={<Title order={(i + 1) as 1 | 2 | 3}>{t('shared.devui.type_sample')}</Title>}
          />
        ))}
      </Section>
      <Section title={t('shared.devui.type_text')}>
        {textSizes.map((s) => (
          <Spec
            key={s}
            name={s}
            value={`${px(theme.fontSizes[s])} / ${theme.lineHeights[s]}`}
            use={t(`shared.devui.type_${s}`)}
            sample={<Text size={s}>{t('shared.devui.type_sample')}</Text>}
          />
        ))}
      </Section>
      <Section title={t('shared.devui.type_weights')}>
        {weights.map((w) => (
          <Spec key={w} name={String(w)} value={theme.fontFamily ?? ''} use={t(`shared.devui.weight_${w}`)} sample={<Text weight={w}>{t('shared.devui.type_sample')}</Text>} />
        ))}
      </Section>
    </Demo>
  )
}

const sizes = ['--mmerp-control-height', '--mmerp-side-width', '--mmerp-aside-width', '--mmerp-content-width', '--mmerp-detail-width']

export function Spacing() {
  const { t } = useTranslation()
  const theme = useMantineTheme()
  return (
    <Demo title={t('shared.devui.spacing')} description={t('shared.devui.spacing_hint')}>
      <Section title={t('shared.devui.spacing')}>
        {Object.entries(theme.spacing).map(([k, v]) => (
          <Spec key={k} name={k} value={px(v)} use={t(`shared.devui.space_${k}`)} sample={<Patch background="var(--mmerp-color-primary-2)" height="var(--mmerp-space-xs)" width={`var(--mmerp-space-${k})`} />} />
        ))}
      </Section>
      <Section title={t('shared.devui.sizes')}>
        <Group gap="xs">
          <TextInput aria-label={t('shared.devui.name')} placeholder={t('shared.devui.name')} />
          <Button leftSection={<IconPlus {...icon.button} />}>{t('shared.devui.add')}</Button>
          <Button variant="default">{t('shared.devui.export')}</Button>
        </Group>
        {sizes.map((name) => (
          <Spec key={name} name={name} value={<CssValue name={name} />} use={t(`shared.devui.token${name.replaceAll('-', '_')}`)} sample={null} />
        ))}
      </Section>
    </Demo>
  )
}

const radii: { key: string; value: string }[] = [
  { key: 'sm', value: 'sm' },
  { key: 'md', value: 'md' },
  { key: 'lg', value: 'lg' },
  { key: 'round', value: '100%' },
]

export function Surfaces() {
  const { t } = useTranslation()
  const theme = useMantineTheme()
  return (
    <Demo title={t('shared.devui.surfaces')} description={t('shared.devui.surfaces_hint')}>
      <Section title={t('shared.devui.radius')}>
        {radii.map(({ key, value }) => (
          <Spec
            key={key}
            name={key}
            value={key === 'round' ? value : px(theme.radius[key as keyof typeof theme.radius] ?? '')}
            use={t(`shared.devui.radius_${key}`)}
            sample={<Patch width="var(--mmerp-space-xxl)" height="var(--mmerp-space-xxl)" radius={value} background="var(--mmerp-color-elevation-surface-sunken)" border="1px solid var(--mmerp-color-border)" />}
          />
        ))}
      </Section>
      <Section title={t('shared.devui.elevation')}>
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="lg">
          <Paper padding="md" className={devui.sunken}>
            <Text size="sm">{t('shared.devui.elevation_sunken')}</Text>
          </Paper>
          <Paper padding="md" withBorder>
            <Text size="sm">{t('shared.devui.elevation_flat')}</Text>
          </Paper>
          <Paper padding="md" className={devui.overlay}>
            <Text size="sm">{t('shared.devui.elevation_overlay')}</Text>
          </Paper>
          <Paper padding="md" className={devui.footer}>
            <Text size="sm">{t('shared.devui.elevation_footer')}</Text>
          </Paper>
        </SimpleGrid>
      </Section>
    </Demo>
  )
}

const iconSizes = ['text', 'button', 'empty'] as const

export function Icons() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.icons')} description={t('shared.devui.icons_hint')}>
      <Section title={t('shared.devui.icons')}>
        {iconSizes.map((k) => (
          <Spec
            key={k}
            name={`icon.${k}`}
            value={`${icon[k].size}px · stroke ${icon[k].stroke}`}
            use={t(`shared.devui.icon_${k}`)}
            sample={<IconInbox {...icon[k]} aria-hidden />}
          />
        ))}
      </Section>
    </Demo>
  )
}
