import { ActionIcon, Alert, Button, Group, Menu, SegmentedControl, Tabs, Text, Tooltip } from '@mantine/core'
import { IconAlertTriangle, IconCircleCheck, IconDownload, IconInfoCircle, IconPlus, IconSparkles, IconTrash, IconUsersGroup } from '@tabler/icons-react'
import { DocumentStatus, type DocStatus } from '@/shared/document'
import { useTranslation } from 'react-i18next'
import { PersonAvatar, PersonName } from '@/shared/ui/avatar'
import { FieldList } from '@/shared/ui/FieldList'
import { MoreActions } from '@/shared/ui/MoreActions'
import { OrgUnitName } from '@/shared/ui/OrgUnitName'
import { Section } from '@/shared/ui/Section'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { LinkTile, StatTile, TileGrid } from '@/shared/ui/Tile'
import { TreeList } from '@/shared/ui/TreeList'
import { Demo } from './Demo'

export function Buttons() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.buttons')} description={t('shared.devui.buttons_hint')}>
      <Group>
        <Button leftSection={<IconPlus {...icon.button} />}>{t('shared.devui.primary')}</Button>
        <Button variant="default">{t('shared.devui.secondary')}</Button>
        <Button variant="subtle">{t('shared.devui.add')}</Button>
        <Button variant="subtle" color="gray">
          {t('shared.form.cancel')}
        </Button>
        <Button variant="light" leftSection={<IconPlus {...icon.button} />}>
          {t('shared.devui.add_line')}
        </Button>
      </Group>
      <Group>
        <Button variant="subtle" color="danger">
          {t('shared.devui.danger')}
        </Button>
        <Button variant="outline" color="danger">
          {t('shared.devui.reject')}
        </Button>
        <Button color="danger">{t('shared.devui.danger')}</Button>
      </Group>
      <Group>
        <Button loading>{t('shared.devui.primary')}</Button>
        <Button disabled>{t('shared.devui.primary')}</Button>
        <Tooltip label={t('shared.devui.remove')}>
          <ActionIcon variant="subtle" color="gray" size="input-sm" aria-label={t('shared.devui.remove')}>
            <IconTrash {...icon.button} />
          </ActionIcon>
        </Tooltip>
        <MoreActions place="page">
          <Menu.Item leftSection={<IconDownload {...icon.text} />}>{t('shared.devui.export')}</Menu.Item>
        </MoreActions>
        <MoreActions>
          <Menu.Item leftSection={<IconDownload {...icon.text} />}>{t('shared.devui.export')}</Menu.Item>
        </MoreActions>
      </Group>
    </Demo>
  )
}

const docStatuses: DocStatus[] = ['draft', 'pending_approval', 'posted', 'cancelled']

// Signals are the ways a screen says something about its data: badges, banners, view switches.
export function Signals() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.signals')}>
      <Group>
        {docStatuses.map((s) => (
          <DocumentStatus key={s} docType="shared" status={s} />
        ))}
      </Group>
      <Alert color="info" icon={<IconInfoCircle {...icon.button} />}>
        {t('shared.devui.alert_info')}
      </Alert>
      <Alert color="success" icon={<IconCircleCheck {...icon.button} />}>
        {t('shared.devui.alert_success')}
      </Alert>
      <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />}>
        {t('shared.devui.alert_warning')}
      </Alert>
      <Alert color="danger" icon={<IconAlertTriangle {...icon.button} />} role="alert">
        {t('shared.devui.alert_danger')}
      </Alert>
      <Alert color="discovery" icon={<IconSparkles {...icon.button} />}>
        {t('shared.devui.alert_discovery')}
      </Alert>
      <Tabs defaultValue="a">
        <Tabs.List>
          <Tabs.Tab value="a">{`${t('shared.devui.tab')} 1`}</Tabs.Tab>
          <Tabs.Tab value="b">{`${t('shared.devui.tab')} 2`}</Tabs.Tab>
          <Tabs.Tab value="c">{`${t('shared.devui.tab')} 3`}</Tabs.Tab>
        </Tabs.List>
      </Tabs>
      <SegmentedControl w="fit-content" data={[1, 2, 3].map((n) => `${t('shared.devui.tab')} ${n}`)} />
    </Demo>
  )
}

export function People() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.people')}>
      <Group>
        <PersonName name="Nguyễn Văn An" />
        <OrgUnitName unit={{ kind: 'company', name: 'Công ty CP Demo' }} />
        <OrgUnitName unit={{ kind: 'department', name: 'Kế toán' }} depth={1} />
        <PersonAvatar name="Trần Thị Bình" size="lg" />
      </Group>
      <Group>
        <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
        <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
        <StatusBadge tone="info">{t('shared.devui.status_info')}</StatusBadge>
        <StatusBadge tone="warning">{t('shared.devui.status_warning')}</StatusBadge>
        <StatusBadge tone="negative">{t('shared.devui.status_negative')}</StatusBadge>
      </Group>
      <FieldList
        rows={[
          ['Nhân viên', 'NV0019 · Lâm Đức Thịnh'],
          ['Thời gian nghỉ', '12/10/2026 – 16/10/2026'],
          ['Lý do', 'Du lịch nước ngoài'],
        ]}
      />
      <TreeList
        label={t('shared.devui.people')}
        nodes={[
          {
            key: 1,
            item: 'Công ty CP Demo',
            children: [
              { key: 2, item: 'Kế toán', children: [] },
              { key: 3, item: 'Chi nhánh Hà Nội', children: [{ key: 4, item: 'Kinh doanh', children: [] }] },
            ],
          },
        ]}
        render={(name) => <Text inherit>{name}</Text>}
        meta={() => 'Meta'}
        toggleLabel={(name) => name}
      />
    </Demo>
  )
}

export function Tiles() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.tiles')}>
      <TileGrid>
        <StatTile label={t('shared.devui.name')} value={128} to="/dev/ui/tiles" />
        <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui/tiles" />
        <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui/tiles" error={t('shared.devui.error')} />
        <LinkTile icon={IconUsersGroup} title={t('shared.devui.name')} to="/dev/ui/tiles" />
      </TileGrid>
    </Demo>
  )
}

export function Sections() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.section')}>
      <Section
        title={t('shared.devui.section')}
        description={t('shared.devui.section_hint')}
        actions={
          <>
            <Button variant="default" leftSection={<IconPlus {...icon.button} />}>
              {t('shared.devui.add')}
            </Button>
            <MoreActions place="page">
              <Menu.Item leftSection={<IconDownload {...icon.text} />}>{t('shared.devui.export')}</Menu.Item>
            </MoreActions>
          </>
        }
      >
        <FieldList rows={[[t('shared.devui.name'), 'Công ty CP Demo']]} />
      </Section>
    </Demo>
  )
}
