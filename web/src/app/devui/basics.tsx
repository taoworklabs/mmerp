import { ActionIcon, Anchor, Badge, Breadcrumbs, Button, Checkbox, Group, Menu, Popover, SegmentedControl, Stack, Tabs, Text, TextInput, Tooltip } from '@mantine/core'
import {
  IconChevronRight,
  IconColumns3,
  IconDownload,
  IconPencil,
  IconPlus,
  IconTrash,
  IconUsersGroup,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { DocumentStatus, type DocStatus } from '@/shared/document'
import { PersonAvatar, PersonName } from '@/shared/ui/avatar'
import { FieldList } from '@/shared/ui/FieldList'
import { MenuAction } from '@/shared/ui/MenuAction'
import { MoreActions } from '@/shared/ui/MoreActions'
import { Notice } from '@/shared/ui/Notice'
import { OrgUnitName } from '@/shared/ui/OrgUnitName'
import { Section } from '@/shared/ui/Section'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { LinkTile, StatTile, TileGrid } from '@/shared/ui/Tile'
import { TreeList } from '@/shared/ui/TreeList'
import { Case, Demo } from './Demo'

export function Buttons() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.buttons')} description={t('shared.devui.buttons_hint')}>
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
    </Demo>
  )
}

export function IconButtons() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.icon_buttons')} description={t('shared.devui.icon_buttons_hint')}>
      <Case label={t('shared.devui.case_icon_beside')}>
        <Group gap="xs">
          <TextInput aria-label={t('shared.devui.name')} placeholder={t('shared.devui.name')} />
          <Tooltip label={t('shared.devui.export')}>
            <ActionIcon variant="subtle" color="gray" size="input-sm" aria-label={t('shared.devui.export')}>
              <IconDownload {...icon.button} />
            </ActionIcon>
          </Tooltip>
          <Button variant="default">{t('shared.devui.secondary')}</Button>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_icon_alone')}>
        <Group gap="xs">
          <Tooltip label={t('shared.devui.edit')}>
            <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.devui.edit')}>
              <IconPencil {...icon.button} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_icon_remove')}>
        <Group gap="xs">
          <Tooltip label={t('shared.devui.remove')}>
            <ActionIcon variant="subtle" color="gray" size="input-sm" aria-label={t('shared.devui.remove')}>
              <IconTrash {...icon.button} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Case>
    </Demo>
  )
}

export function Menus() {
  const { t } = useTranslation()
  const items = (
    <>
      <MenuAction icon={IconDownload} onClick={() => {}}>
        {t('shared.devui.export')}
      </MenuAction>
      <MenuAction icon={IconPencil} onClick={() => {}}>
        {t('shared.devui.edit')}
      </MenuAction>
      <Menu.Divider />
      <MenuAction icon={IconTrash} danger onClick={() => {}}>
        {t('shared.devui.danger')}
      </MenuAction>
    </>
  )
  return (
    <Demo title={t('shared.devui.menus')} description={t('shared.devui.menus_hint')}>
      <Case label={t('shared.devui.case_menu_page')}>
        <Group gap="xs">
          <Button variant="default">{t('shared.devui.secondary')}</Button>
          <MoreActions place="page">{items}</MoreActions>
          <Button>{t('shared.devui.primary')}</Button>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_menu_row')}>
        <Group gap="xs">
          <MoreActions>{items}</MoreActions>
        </Group>
      </Case>
    </Demo>
  )
}

// Sample values the links point at; data, not interface text.
const sample = { person: 'NV0019 · Lâm Đức Thịnh', phone: '0912 345 678', email: 'thinh.lam@demo.vn' }

export function Links() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.links')} description={t('shared.devui.links_hint')}>
      <Case label={t('shared.devui.case_link_record')}>
        <Group>
          <Anchor component={Link} to="/dev/ui/links" size="sm">
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
          <Anchor component="button" type="button" size="sm">
            {t('shared.devui.show_more')}
          </Anchor>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_breadcrumbs')}>
        <Breadcrumbs separator={<IconChevronRight {...icon.text} />} separatorMargin="xxs">
          <Anchor component={Link} to="/dev/ui/links" size="sm" c="dimmed">
            {t('shared.devui.title')}
          </Anchor>
          <Anchor component={Link} to="/dev/ui/links" size="sm" c="dimmed">
            {t('shared.devui.group_navigation')}
          </Anchor>
        </Breadcrumbs>
      </Case>
    </Demo>
  )
}

const docStatuses: DocStatus[] = ['draft', 'pending_approval', 'posted', 'cancelled']

export function Badges() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.badges')} description={t('shared.devui.badges_hint')}>
      <Case label={t('shared.devui.case_doc_status')}>
        <Group>
          {docStatuses.map((s) => (
            <DocumentStatus key={s} docType="shared" status={s} />
          ))}
        </Group>
      </Case>
      <Case label={t('shared.devui.case_record_status')}>
        <Group>
          <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
          <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
          <StatusBadge tone="info">{t('shared.devui.status_info')}</StatusBadge>
          <StatusBadge tone="warning">{t('shared.devui.status_warning')}</StatusBadge>
          <StatusBadge tone="negative">{t('shared.devui.status_negative')}</StatusBadge>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_count')}>
        <Group gap="xs">
          <Text size="sm">{t('shared.devui.section')}</Text>
          <Badge color="danger" size="sm" circle aria-label={t('shared.devui.count_label', { count: 2 })}>
            2
          </Badge>
        </Group>
      </Case>
    </Demo>
  )
}

export function Alerts() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.alerts')} description={t('shared.devui.alerts_hint')}>
      <Notice tone="info">{t('shared.devui.alert_info')}</Notice>
      <Notice tone="success">{t('shared.devui.alert_success')}</Notice>
      <Notice tone="warning">{t('shared.devui.alert_warning')}</Notice>
      <Notice tone="danger">{t('shared.devui.alert_danger')}</Notice>
      <Notice tone="discovery">{t('shared.devui.alert_discovery')}</Notice>
    </Demo>
  )
}

export function TabsDemo() {
  const { t } = useTranslation()
  const tabs = [1, 2, 3].map((n) => `${t('shared.devui.tab')} ${n}`)
  return (
    <Demo title={t('shared.devui.tabs')} description={t('shared.devui.tabs_hint')}>
      <Case label={t('shared.devui.case_tabs')}>
        <Tabs defaultValue={tabs[0]}>
          <Tabs.List>
            {tabs.map((tab) => (
              <Tabs.Tab key={tab} value={tab}>
                {tab}
              </Tabs.Tab>
            ))}
          </Tabs.List>
        </Tabs>
      </Case>
      <Case label={t('shared.devui.case_switch')}>
        <SegmentedControl w="fit-content" data={tabs} />
      </Case>
    </Demo>
  )
}

export function Overlays() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.overlays')} description={t('shared.devui.overlays_hint')}>
      <Case label={t('shared.devui.case_tooltip')}>
        <Group>
          <Tooltip label={t('shared.devui.export')}>
            <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.devui.export')}>
              <IconDownload {...icon.button} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Case>
      <Case label={t('shared.devui.case_popover')}>
        <Group>
          <Popover position="bottom-start">
            <Popover.Target>
              <Button variant="default" leftSection={<IconColumns3 {...icon.button} />}>
                {t('shared.views.columns')}
              </Button>
            </Popover.Target>
            <Popover.Dropdown>
              <Stack gap="xs">
                <Text size="xs" fw={600} c="dimmed">
                  {t('shared.views.columns_hint')}
                </Text>
                <Checkbox label={t('shared.devui.name')} defaultChecked />
                <Checkbox label={t('shared.devui.org_unit')} defaultChecked />
                <Checkbox label={t('shared.devui.amount')} />
              </Stack>
            </Popover.Dropdown>
          </Popover>
        </Group>
      </Case>
    </Demo>
  )
}

export function Avatars() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.avatars')} description={t('shared.devui.avatars_hint')}>
      <Case label={t('shared.devui.case_person')}>
        <Group>
          <PersonName name="Nguyễn Văn An" />
          <PersonName name="Trần Thị Bình" />
          <PersonName name="Lê Văn Cường" />
        </Group>
      </Case>
      <Case label={t('shared.devui.case_avatar_sizes')}>
        <Group>
          <PersonAvatar name="Trần Thị Bình" size="sm" />
          <PersonAvatar name="Trần Thị Bình" size="md" />
          <PersonAvatar name="Trần Thị Bình" size="lg" />
        </Group>
      </Case>
    </Demo>
  )
}

export function FieldLists() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.field_list')} description={t('shared.devui.field_list_hint')}>
      <FieldList
        rows={[
          [
            'Nhân viên',
            <Anchor key="p" component={Link} to="/dev/ui/field-list" size="sm" inherit>
              {sample.person}
            </Anchor>,
          ],
          ['Thời gian nghỉ', '12/10/2026 – 16/10/2026'],
          ['Trạng thái', <StatusBadge key="s" tone="positive">{t('shared.devui.status_positive')}</StatusBadge>],
          ['Nơi làm việc', t('shared.devui.long_value')],
          ['Ghi chú', '—'],
        ]}
      />
    </Demo>
  )
}

export function Tree() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.tree')} description={t('shared.devui.tree_hint')}>
      <TreeList
        label={t('shared.devui.tree')}
        nodes={[
          {
            key: 1,
            item: { kind: 'company' as const, name: 'Công ty CP Demo' },
            children: [
              { key: 2, item: { kind: 'department' as const, name: 'Kế toán' }, children: [] },
              { key: 3, item: { kind: 'branch' as const, name: 'Chi nhánh Hà Nội' }, children: [{ key: 4, item: { kind: 'department' as const, name: 'Kinh doanh' }, children: [] }] },
            ],
          },
        ]}
        render={(unit) => <OrgUnitName unit={unit} />}
        meta={() => 'Meta'}
        toggleLabel={(unit) => unit.name}
      />
    </Demo>
  )
}

export function Tiles() {
  const { t } = useTranslation()
  return (
    <Demo title={t('shared.devui.tiles')} description={t('shared.devui.tiles_hint')}>
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
    <Demo title={t('shared.devui.section')} description={t('shared.devui.sections_hint')}>
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
