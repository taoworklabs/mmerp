import { Anchor, Button, Divider, Group, List, Loader, Skeleton, Stack, Text, Tooltip } from '@mmerp/ui'
import { IconDownload, IconPlus } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { ContentSkeleton } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { Case } from './Demo'
import { DocPage, DoDont } from './DocPage'

// Sample values the links point at; data, not interface text.
const sample = { person: 'NV0019 · Lâm Đức Thịnh', phone: '0912 345 678', email: 'thinh.lam@demo.vn' }

// The props each kind takes; data, not interface text.
const code = {
  button: `<Button>                              // primary: one per place
<Button variant="default">            // secondary
<Button variant="subtle" color="gray"> // the way out beside a primary
<Button color="danger">               // destructive, after a confirmation`,
  anchor: `<Anchor render={<Link to={path} />}>  // a record or page
<Anchor href="tel:…">                 // a phone number or email
<Anchor render={<button />}>           // only for "show more" inside text`,
  layout: `<Group gap="xs">  <Stack gap="md">   // gaps are theme keys, never numbers
<Text size="sm" tone="dimmed">          // sizes and colours are theme keys too`,
  loader: `<Loader size="sm" label={t('shared.state.loading')} />   // beside the text of what is loading`,
  skeleton: `<Skeleton height="1em" width="4ch" />   // the shape of one small value`,
  tooltip: `<Tooltip label={t('…')}>
  <Button variant="subtle" color="gray" aria-label={t('…')}>{icon}</Button>
</Tooltip>`,
}

export function AnchorDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/anchor"
      example={
        <Case>
          <Group>
            <Anchor render={<Link to="/dev/ui/mantine/anchor" />} size="sm">
              {sample.person}
            </Anchor>
          </Group>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_link_record')}>
            <Group>
              <Anchor render={<Link to="/dev/ui/mantine/anchor" />} size="sm">
                {sample.person}
              </Anchor>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_link_contact')}>
            <Group>
              <Anchor href="tel:0912345678" size="sm">
                {sample.phone}
              </Anchor>
              <Anchor href="mailto:thinh.lam@demo.vn" size="sm">
                {sample.email}
              </Anchor>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_link_action')}>
            <Group>
              <Anchor render={<button />} type="button" size="sm">
                {t('shared.devui.show_more')}
              </Anchor>
            </Group>
          </Case>
        </>
      }
      code={code.anchor}
    />
  )
}

export function ButtonDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/button"
      example={
        <Case>
          <Group justify="end">
            <Button variant="subtle" color="gray">
              {t('shared.form.cancel')}
            </Button>
            <Button leftSection={<IconPlus {...icon.button} />}>{t('shared.devui.primary')}</Button>
          </Group>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_primary')}>
            <Group>
              <Button leftSection={<IconPlus {...icon.button} />}>{t('shared.devui.primary')}</Button>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_secondary')}>
            <Group>
              <Button variant="default">{t('shared.devui.secondary')}</Button>
              <Button variant="subtle">{t('shared.devui.add')}</Button>
              <Button variant="subtle" color="gray">
                {t('shared.form.cancel')}
              </Button>
              <Button variant="light" leftSection={<IconPlus {...icon.button} />}>
                {t('shared.devui.add_line')}
              </Button>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_destructive')}>
            <Group>
              <Button variant="subtle" color="danger">
                {t('shared.devui.danger')}
              </Button>
              <Button variant="outline" color="danger">
                {t('shared.devui.reject')}
              </Button>
              <Button color="danger">{t('shared.devui.danger')}</Button>
            </Group>
          </Case>
          <Case label={t('shared.devui.case_states')}>
            <Group>
              <Button loading>{t('shared.devui.primary')}</Button>
              <Button disabled>{t('shared.devui.primary')}</Button>
            </Group>
          </Case>
        </>
      }
      doDont={
        <DoDont
          good={
            <Group>
              <Button variant="default">{t('shared.devui.secondary')}</Button>
              <Button>{t('shared.devui.primary')}</Button>
            </Group>
          }
          bad={
            <>
              <Group>
                <Button>{t('shared.devui.secondary')}</Button>
                <Button>{t('shared.devui.primary')}</Button>
              </Group>
              <Text size="sm" tone="dimmed">
                {t('shared.devui.dont_two_primary')}
              </Text>
            </>
          }
        />
      }
      code={code.button}
    />
  )
}

export function LayoutTextDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/layout-text"
      example={
        <Case>
          <Stack gap="xs">
            <Text weight={600}>{t('shared.devui.section_staff')}</Text>
            <Text size="sm" tone="dimmed">
              {t('shared.devui.section_hint')}
            </Text>
          </Stack>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_group_stack')}>
            <Group gap="xs">
              <Button variant="default">{t('shared.devui.secondary')}</Button>
              <Button>{t('shared.devui.primary')}</Button>
            </Group>
            <Stack gap="xxs">
              <Text size="sm">{t('shared.devui.name')}</Text>
              <Text size="sm">{t('shared.devui.code')}</Text>
            </Stack>
          </Case>
          <Case label={t('shared.devui.case_divider')}>
            <Text size="sm">{t('shared.devui.section_company')}</Text>
            <Divider />
            <Text size="sm">{t('shared.devui.section_staff')}</Text>
          </Case>
          <Case label={t('shared.devui.case_list')}>
            <List spacing="xs">
              <List.Item>{t('shared.devui.name')}</List.Item>
              <List.Item>{t('shared.devui.code')}</List.Item>
            </List>
          </Case>
          <Case label={t('shared.devui.case_text')}>
            <Text size="sm">{t('shared.devui.page_content')}</Text>
            <Text size="sm" tone="dimmed">
              {t('shared.devui.hint')}
            </Text>
            <Text size="xs" tone="dimmed">
              {t('shared.devui.hint')}
            </Text>
          </Case>
        </>
      }
      code={code.layout}
    />
  )
}

export function LoaderDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/loader"
      example={
        <Case>
          <Group gap="xs">
            <Loader size="sm" label={t('shared.state.loading')} />
            <Text size="sm" tone="dimmed">
              {t('shared.devui.loading')}
            </Text>
          </Group>
        </Case>
      }
      doDont={
        <DoDont
          good={<ContentSkeleton />}
          bad={
            <>
              <Group justify="center">
                <Loader label={t('shared.state.loading')} />
              </Group>
              <Text size="sm" tone="dimmed">
                {t('shared.devui.dont_loader_region')}
              </Text>
            </>
          }
        />
      }
      code={code.loader}
    />
  )
}

export function SkeletonDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/skeleton"
      example={
        <Case>
          <Group gap="xs">
            <Text size="sm">{t('shared.devui.name')}</Text>
            <Skeleton height="1em" width="4ch" />
          </Group>
        </Case>
      }
      doDont={
        <DoDont
          good={<ContentSkeleton />}
          bad={
            <>
              <Stack gap="xs">
                <Skeleton height="1em" />
                <Skeleton height="1em" width="70%" />
                <Skeleton height="1em" width="40%" />
              </Stack>
              <Text size="sm" tone="dimmed">
                {t('shared.devui.dont_skeleton_stack')}
              </Text>
            </>
          }
        />
      }
      code={code.skeleton}
    />
  )
}

export function TooltipDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/tooltip"
      example={
        <Case>
          <Group>
            <Tooltip label={t('shared.devui.export')}>
              <Button variant="subtle" color="gray" aria-label={t('shared.devui.export')}>
                <IconDownload {...icon.button} />
              </Button>
            </Tooltip>
          </Group>
        </Case>
      }
      doDont={
        <DoDont
          good={
            <Group>
              <Tooltip label={t('shared.devui.export')}>
                <Button variant="subtle" color="gray" aria-label={t('shared.devui.export')}>
                  <IconDownload {...icon.button} />
                </Button>
              </Tooltip>
            </Group>
          }
          bad={
            <>
              <Group>
                <Tooltip label={t('shared.devui.export')}>
                  <Button variant="default">{t('shared.devui.export')}</Button>
                </Tooltip>
              </Group>
              <Text size="sm" tone="dimmed">
                {t('shared.devui.dont_tooltip_repeat')}
              </Text>
            </>
          }
        />
      }
      code={code.tooltip}
    />
  )
}
