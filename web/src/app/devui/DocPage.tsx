import { Anchor, Badge, Box, Code, Group, SimpleGrid, Stack, Text, Title } from '@mmerp/ui'
import { TableOfContents } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Page } from '@/shared/ui/page'
import { type Entry, entryByPath, key, type Path, splitNames } from './catalogue'
import classes from './DevUi.module.css'

// Linked shows a text with every catalogued component it names as a link to that component's page.
export function Linked({ text }: { text: string }) {
  return splitNames(text).map((part, i) =>
    typeof part === 'string' ? (
      part
    ) : (
      <Anchor key={i} render={<Link to={`/dev/ui/${part.path}`} />}>
        {part.name}
      </Anchor>
    ),
  )
}

export function useEntryTitle() {
  const { t } = useTranslation()
  return (e: Entry) => e.name ?? t(e.label ?? '')
}

function DocSection({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <Stack gap="sm" render={<section />} aria-labelledby={id}>
      <Title order={2} id={id} data-devui-section className={classes.heading}>
        {title}
      </Title>
      {children}
    </Stack>
  )
}

// DocPage is one catalogue page in the fixed order every page shares; a part with nothing to show
// is left out. The page passes only its examples and code; the rest comes from its entry.
export function DocPage({ path, example, variants, doDont, code }: { path: Path; example: ReactNode; variants?: ReactNode; doDont?: ReactNode; code?: string }) {
  const { t } = useTranslation()
  const title = useEntryTitle()
  const entry = entryByPath(path)
  const k = key(entry)
  const component = entry.layer === 'mantine' || entry.layer === 'mmerp'
  const imports = entry.source ? `import { ${(entry.exports ?? [entry.name]).join(', ')} } from '${entry.source}'` : ''
  const snippet = [imports, code].filter(Boolean).join('\n\n')
  const tag =
    entry.layer === 'mantine' ? (
      <Badge variant="light" color="gray">
        {t('shared.devui.source_mantine')}
      </Badge>
    ) : entry.layer === 'mmerp' ? (
      <Badge variant="light">{t('shared.devui.source_mmerp')}</Badge>
    ) : undefined
  const description = (
    <>
      {t(`shared.devui.about_${k}`)}
      {entry.layer === 'mantine' && !entry.kit && (
        <>
          {' '}
          <Anchor href={`https://mantine.dev/core/${(entry.covers?.[0] ?? entry.name ?? '').toLowerCase()}/`} target="_blank" rel="noreferrer">
            {t('shared.devui.mantine_docs')}
          </Anchor>
        </>
      )}
    </>
  )
  return (
    <Page centered title={title(entry)} status={tag} description={description} breadcrumbs={[{ label: t('shared.devui.title'), to: '/dev/ui' }]}>
      <Group align="start" wrap={false} gap="xl">
        <Stack gap="xl" fill>
          <DocSection id="example" title={t('shared.devui.sec_example')}>
            {example}
          </DocSection>
          <DocSection id="usage" title={t('shared.devui.sec_usage')}>
            <Text size="sm">
              <Text render={<span />} weight={600}>
                {t('shared.devui.use_when')}
              </Text>{' '}
              {t(`shared.devui.use_${k}`)}
            </Text>
            <Text size="sm">
              <Text render={<span />} weight={600}>
                {t('shared.devui.avoid_when')}
              </Text>{' '}
              <Linked text={t(`shared.devui.avoid_${k}`)} />
            </Text>
          </DocSection>
          {variants && (
            <DocSection id="variants" title={t('shared.devui.sec_variants')}>
              {variants}
            </DocSection>
          )}
          {doDont && (
            <DocSection id="do-dont" title={t('shared.devui.sec_do_dont')}>
              {doDont}
            </DocSection>
          )}
          {snippet && (
            <DocSection id="code" title={t('shared.devui.sec_code')}>
              <Code block>{snippet}</Code>
            </DocSection>
          )}
          {entry.related && entry.related.length > 0 && (
            <DocSection id="related" title={t(component ? 'shared.devui.sec_related' : 'shared.devui.sec_uses')}>
              <Group gap="md">
                {entry.related.map((p) => (
                  <Anchor key={p} render={<Link to={`/dev/ui/${p}`} />} size="sm">
                    {title(entryByPath(p))}
                  </Anchor>
                ))}
              </Group>
            </DocSection>
          )}
        </Stack>
        <Box className={classes.toc}>
          <Text size="xs" weight={600} tone="dimmed" className={classes.tocTitle}>
            {t('shared.devui.on_this_page')}
          </Text>
          <TableOfContents
            size="sm"
            variant="light"
            scrollSpyOptions={{ selector: '[data-devui-section]' }}
            getControlProps={({ data }) => ({ onClick: () => data.getNode().scrollIntoView({ behavior: 'smooth' }), children: data.value })}
          />
        </Box>
      </Group>
    </Page>
  )
}

// DoDont pairs the right way with the wrong one, each marked by its border and label.
export function DoDont({ good, bad }: { good: ReactNode; bad: ReactNode }) {
  const { t } = useTranslation()
  return (
    <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
      <Stack gap="xs" className={classes.do}>
        <Text size="sm" weight={600} tone="success">
          {t('shared.devui.do')}
        </Text>
        {good}
      </Stack>
      <Stack gap="xs" className={classes.dont}>
        <Text size="sm" weight={600} tone="danger">
          {t('shared.devui.dont')}
        </Text>
        {bad}
      </Stack>
    </SimpleGrid>
  )
}
