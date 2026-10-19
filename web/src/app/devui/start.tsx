import { Anchor, Group, Stack, Text } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { Page } from '@/shared/ui/page'
import { Section } from '@/shared/ui/Section'
import { type Entry, entries, layers } from './catalogue'
import { useEntryTitle } from './DocPage'

// Overview explains the layers, each with its entries, and points to where the written rules live.
export function Overview() {
  const { t } = useTranslation()
  const title = useEntryTitle()
  return (
    <Page centered title={t('shared.devui.overview')} description={t('shared.devui.overview_hint')} breadcrumbs={[{ label: t('shared.devui.title'), to: '/dev/ui' }]}>
      <Stack gap="xl">
        {layers
          .filter((l) => l !== 'start')
          .map((l) => (
            <Section key={l} title={`${layers.indexOf(l) + 1} · ${t(`shared.devui.layer_${l}`)}`} description={t(`shared.devui.layer_${l}_note`)}>
              <Group gap="md">
                {(entries as readonly Entry[])
                  .filter((e) => e.layer === l)
                  .map((e) => (
                    <Anchor key={e.path} component={Link} to={`/dev/ui/${e.path}`} size="sm">
                      {title(e)}
                    </Anchor>
                  ))}
              </Group>
            </Section>
          ))}
        <Section title={t('shared.devui.overview_rules')}>
          <Text size="sm">{t('shared.devui.overview_rules_text')}</Text>
        </Section>
      </Stack>
    </Page>
  )
}
