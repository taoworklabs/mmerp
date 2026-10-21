// Screens/Employees: the HRM employee list, record and new-employee screens, built only from
// @mmerp/ui — proof the kit can show a real mmerp screen end to end. Routing between the three is
// local React state, as a real area would do it with the router.
import type { Meta, StoryObj } from '@storybook/react-vite'
import {
  IconAddressBook,
  IconArrowsExchange,
  IconBeach,
  IconBell,
  IconBuildingSkyscraper,
  IconCalendar,
  IconCash,
  IconChevronDown,
  IconCircleDot,
  IconClock,
  IconFileImport,
  IconFileText,
  IconHourglass,
  IconInbox,
  IconLayoutSidebarLeftCollapse,
  IconLayoutSidebarLeftExpand,
  IconLock,
  IconLogout,
  IconPencil,
  IconPlus,
  IconUser,
  IconUserOff,
  IconUsers,
} from '@tabler/icons-react'
import { useState, type ReactNode } from 'react'
import {
  ActionIcon,
  Anchor,
  AppShell,
  Avatar,
  Badge,
  Burger,
  Button,
  Combobox,
  DataTable,
  DateInput,
  FieldList,
  FilterBar,
  FilterChip,
  FormBar,
  FormFields,
  FormSection,
  Group,
  Indicator,
  KeyFacts,
  Menu,
  MoreActions,
  NavLink,
  Page,
  PageHeader,
  PagingBar,
  Panel,
  RecordColumns,
  Select,
  Stack,
  Tabs,
  Text,
  TextInput,
  ThemeIcon,
  Tooltip,
  addDays,
  icon,
  useDisclosure,
  useMediaQuery,
  type CalendarLocale,
  type Column,
  type DecorativeColor,
} from '../index.ts'
import { employees, units, type Employee } from './samples.ts'

// ---- dates: the app's own locale, parse and format, kept local since a screen imports only the
// kit, react, tabler icons and its own samples -----------------------------------------------

const viLocale: CalendarLocale = {
  weekdays: ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'],
  months: Array.from({ length: 12 }, (_, i) => `Tháng ${i + 1}`),
  firstDayOfWeek: 1,
}

const monthLabels = { previousMonth: 'Tháng trước', nextMonth: 'Tháng sau' }

function parseDate(text: string): string | null {
  const match = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(text)
  if (!match) return null
  const [, d = '', m = '', y = ''] = match
  const iso = `${y}-${m.padStart(2, '0')}-${d.padStart(2, '0')}`
  return addDays(iso, 0) === iso ? iso : null
}

function formatDate(iso: string): string {
  const [y, m, d] = iso.split('-')
  return `${d}/${m}/${y}`
}

const dateProps = { parse: parseDate, format: formatDate, locale: viLocale, labels: monthLabels, placeholder: 'dd/mm/yyyy', leftSection: <IconCalendar {...icon.text} aria-hidden /> }

// ---- small pieces shared by the three screens -----------------------------------------------

const avatarColors: DecorativeColor[] = ['blue', 'teal', 'violet', 'orange', 'pink', 'cyan', 'grape', 'indigo']

function initials(name: string) {
  const words = name.trim().split(/\s+/).filter(Boolean)
  const first = words[0]?.[0] ?? ''
  const last = words.length > 1 ? (words.at(-1)?.[0] ?? '') : ''
  return (first + last).toUpperCase()
}

// PersonAvatar shows a person's initials on a colour derived from the name, so it stays stable.
function PersonAvatar({ name, size = 30 }: { name: string; size?: number }) {
  let h = 0
  for (const ch of name) h = (h * 31 + ch.codePointAt(0)!) >>> 0
  return (
    <Avatar alt="" size={size} color={avatarColors[h % avatarColors.length]} aria-hidden>
      {initials(name)}
    </Avatar>
  )
}

function PersonName({ name }: { name: string }) {
  return (
    <Group gap="xs" wrap={false}>
      <PersonAvatar name={name} size={24} />
      <Text truncate>{name}</Text>
    </Group>
  )
}

function Status({ status }: { status: Employee['status'] }) {
  return status === 'active' ? <Badge color="success">Đang làm việc</Badge> : <Badge color="gray">Đã nghỉ việc</Badge>
}

// MissingValue is an empty field's placeholder: "+ Bổ sung" where a value would otherwise sit.
function MissingValue() {
  return (
    <Anchor render={<button type="button" />}>
      + Bổ sung
    </Anchor>
  )
}

// ---- shell: header bar, grouped nav with a rail toggle ------------------------------------------

type Route = { page: 'list' } | { page: 'detail'; id: number } | { page: 'new' }

const nav: { group?: string; label: string; icon: typeof IconUsers; page?: Route['page'] }[] = [
  { label: 'Tổng quan', icon: IconBuildingSkyscraper },
  { group: 'Hồ sơ', label: 'Nhân viên', icon: IconUsers, page: 'list' },
  { group: 'Hồ sơ', label: 'Hợp đồng', icon: IconFileText },
  { group: 'Chấm công', label: 'Bảng công', icon: IconClock },
  { group: 'Chấm công', label: 'Nghỉ phép', icon: IconBeach },
  { group: 'Lương', label: 'Bảng lương', icon: IconCash },
]

function Shell({ children }: { children: ReactNode }) {
  const [drawer, { toggle: toggleDrawer }] = useDisclosure()
  const [collapsed, setCollapsed] = useState(false)
  const mobile = useMediaQuery('(max-width: 63.99em)')
  const narrow = collapsed && !mobile
  const groups = [...new Set(nav.map((n) => n.group))]
  return (
    <AppShell
      navbarWidth={240}
      navbarCollapsed={collapsed}
      navbarCollapsedWidth={56}
      navbarOpened={drawer}
      header={
        <Group fill justify="space-between" wrap={false}>
          <Group gap="xs" wrap={false}>
            {mobile && <Burger opened={drawer} onClick={toggleDrawer} aria-label="Trình đơn" />}
            <ThemeIcon size={28} aria-hidden>
              <IconBuildingSkyscraper {...icon.text} />
            </ThemeIcon>
            <Text weight={600}>Nhân sự</Text>
          </Group>
          <Group gap="xs" wrap={false}>
            <Tooltip label="Thông báo">
              <Indicator label={3}>
                <ActionIcon variant="subtle" color="gray" aria-label="Thông báo, 3 chưa đọc">
                  <IconBell {...icon.button} aria-hidden />
                </ActionIcon>
              </Indicator>
            </Tooltip>
            <Tooltip label="Hộp duyệt">
              <ActionIcon variant="subtle" color="gray" aria-label="Hộp duyệt">
                <IconInbox {...icon.button} aria-hidden />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Tác vụ nền">
              <ActionIcon variant="subtle" color="gray" aria-label="Tác vụ nền">
                <IconHourglass {...icon.button} aria-hidden />
              </ActionIcon>
            </Tooltip>
            <Menu.Root>
              {/* No aria-label here: the avatar is aria-hidden, so the button's accessible name is
                  already just the visible "Quản trị" text, matching it exactly. */}
              <Menu.Trigger render={<Button variant="subtle" color="gray" rightSection={<IconChevronDown {...icon.text} aria-hidden />} />}>
                <Group gap="xs" wrap={false}>
                  <PersonAvatar name="Quản trị" size={24} />
                  Quản trị
                </Group>
              </Menu.Trigger>
              <Menu.Popup align="end">
                <Menu.Label>Ngôn ngữ</Menu.Label>
                <Menu.Item>Tiếng Việt</Menu.Item>
                <Menu.Item>English</Menu.Item>
                <Menu.Separator />
                <Menu.Item leftSection={<IconLogout {...icon.text} aria-hidden />}>Đăng xuất</Menu.Item>
              </Menu.Popup>
            </Menu.Root>
          </Group>
        </Group>
      }
      navbar={
        <Stack gap={0}>
          {groups.map((g) => (
            <Stack key={g ?? ''} gap={0} role="group" aria-label={g}>
              {g && !narrow && <NavLink.GroupLabel>{g}</NavLink.GroupLabel>}
              {nav
                .filter((n) => n.group === g)
                .map((n) => {
                  const link = (
                    <NavLink
                      key={n.label}
                      href="#"
                      compact
                      label={narrow ? '' : n.label}
                      aria-label={n.label}
                      leftSection={<n.icon {...icon.text} aria-hidden />}
                      active={n.page === 'list'}
                      onClick={(e) => e.preventDefault()}
                    />
                  )
                  return narrow ? (
                    <Tooltip key={n.label} label={n.label} side="right">
                      {link}
                    </Tooltip>
                  ) : (
                    link
                  )
                })}
            </Stack>
          ))}
        </Stack>
      }
      navbarFooter={
        <Tooltip label={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'} side="right">
          <ActionIcon variant="subtle" color="gray" aria-label={collapsed ? 'Mở rộng menu' : 'Thu gọn menu'} onClick={() => setCollapsed(!collapsed)}>
            {collapsed ? <IconLayoutSidebarLeftExpand {...icon.button} aria-hidden /> : <IconLayoutSidebarLeftCollapse {...icon.button} aria-hidden />}
          </ActionIcon>
        </Tooltip>
      }
    >
      {children}
    </AppShell>
  )
}

// ---- list ----------------------------------------------------------------------------------

const statusOptions = [
  { value: 'active', label: 'Đang làm việc' },
  { value: 'terminated', label: 'Đã nghỉ việc' },
]

const dataTableLabels = {
  actions: 'Thao tác khác',
  selectAll: 'Chọn tất cả',
  selectRow: 'Chọn dòng',
  resize: (column: string) => `Đổi độ rộng cột ${column}`,
}

const pagingLabels = { previous: 'Trang trước', next: 'Trang sau', page: (p: number) => `Trang ${p}` }

function ListScreen({ go }: { go: (route: Route) => void }) {
  const [q, setQ] = useState('')
  const [status, setStatus] = useState<string | null>(null)
  const [sort, setSort] = useState('name')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [selected, setSelected] = useState<ReadonlySet<string | number>>(new Set())

  const term = q.trim().toLowerCase()
  const filtered = employees.filter(
    (e) => (!term || e.name.toLowerCase().includes(term) || e.code.toLowerCase().includes(term)) && (!status || e.status === status),
  )
  const key = (sort.startsWith('-') ? sort.slice(1) : sort) as 'name' | 'code' | 'hire'
  const direction = sort.startsWith('-') ? -1 : 1
  const sorted = filtered.sort((a, b) => direction * a[key].localeCompare(b[key]))
  const rows = sorted.slice((page - 1) * pageSize, page * pageSize)

  const columns: Column<Employee>[] = [
    { key: 'name', header: 'Họ tên', role: 'title', kind: 'person', sortable: true, render: (e) => <PersonName name={e.name} /> },
    { key: 'code', header: 'Mã', role: 'meta', kind: 'code', sortable: true, render: (e) => e.code },
    { key: 'unit', header: 'Phòng ban', role: 'meta', render: (e) => e.unit },
    { key: 'hire', header: 'Ngày vào làm', role: 'hidden', kind: 'date', sortable: true, render: (e) => formatDate(e.hire) },
    { key: 'status', header: 'Trạng thái', role: 'status', kind: 'status', render: (e) => <Status status={e.status} /> },
  ]

  return (
    <Page
      variant="fill"
      header={
        <PageHeader
          title="Nhân viên"
          description="Hồ sơ nhân viên của công ty"
          actions={
            <Group gap="xs" wrap={false}>
              <Button variant="default" leftSection={<IconFileImport {...icon.button} aria-hidden />}>
                Nhập số dư phép
              </Button>
              <Button leftSection={<IconPlus {...icon.button} aria-hidden />} onClick={() => go({ page: 'new' })}>
                Thêm nhân viên
              </Button>
            </Group>
          }
        />
      }
    >
      <Page.Content>
        <FilterBar
          label="Bộ lọc nhân viên"
          filters={
            <>
              <FilterBar.Search
                aria-label="Tìm nhân viên"
                placeholder="Tìm theo tên, mã"
                value={q}
                onChange={(e) => {
                  setQ(e.currentTarget.value)
                  setPage(1)
                }}
              />
              <FilterChip
                label="Trạng thái"
                icon={<IconCircleDot {...icon.button} aria-hidden />}
                data={statusOptions}
                value={status}
                onChange={(v) => {
                  setStatus(v)
                  setPage(1)
                }}
                valueText={(label, value) => `${label}: ${value}`}
                clearLabel="Bỏ lọc trạng thái"
                nothingFound="Không có kết quả"
              />
              {(q || status) && (
                <Button
                  variant="default"
                  onClick={() => {
                    setQ('')
                    setStatus(null)
                    setPage(1)
                  }}
                >
                  Xoá bộ lọc
                </Button>
              )}
            </>
          }
        />
        <DataTable
          columns={columns}
          rows={rows}
          rowKey={(e) => e.id}
          label="Nhân viên"
          labels={dataTableLabels}
          maxHeight="60dvh"
          renderLink={(row, children) => (
            <a
              href="#"
              onClick={(e) => {
                e.preventDefault()
                go({ page: 'detail', id: row.id })
              }}
            >
              {children}
            </a>
          )}
          selection={{ selected, onChange: setSelected }}
          sort={sort}
          onSort={setSort}
          menu={() => (
            <>
              <Menu.Item leftSection={<IconArrowsExchange {...icon.text} aria-hidden />}>Điều chuyển</Menu.Item>
              <Menu.Item color="danger" leftSection={<IconUserOff {...icon.text} aria-hidden />}>
                Cho nghỉ việc
              </Menu.Item>
            </>
          )}
        />
        <PagingBar
          total={sorted.length}
          page={page}
          pageSize={pageSize}
          pageSizes={[10, 20, 50]}
          onPage={setPage}
          onPageSize={(size) => {
            setPageSize(size)
            setPage(1)
          }}
          totalText={`${sorted.length} nhân viên`}
          pageSizeText={(n) => `${n} / trang`}
          pageSizeLabel="Số dòng mỗi trang"
          labels={pagingLabels}
          navigationLabel="Phân trang nhân viên"
        />
      </Page.Content>
    </Page>
  )
}

// ---- record ----------------------------------------------------------------------------------

// MainInfoPanel is the one card of the three that switches to an in-place form on Edit; the other
// two stay read-only, since the API's "allowed_actions" would decide that for real, not this story.
function MainInfoPanel({ employee: e }: { employee: Employee }) {
  const [editing, setEditing] = useState(false)
  const [unit, setUnit] = useState<string | null>(e.unit)
  const [hire, setHire] = useState<string | null>(e.hire)
  return (
    <Panel
      title="Thông tin chính"
      icon={<IconUser {...icon.text} aria-hidden />}
      actions={
        !editing && (
          <Button variant="subtle" leftSection={<IconPencil {...icon.button} aria-hidden />} onClick={() => setEditing(true)}>
            Sửa
          </Button>
        )
      }
    >
      {editing ? (
        <Stack gap="md">
          <FormFields>
            <TextInput label="Họ tên" required maxLength={200} defaultValue={e.name} />
            <TextInput label="Mã nhân viên" readOnly defaultValue={e.code} />
            <Combobox label="Phòng ban" required data={units.map((u) => ({ value: u, label: u }))} value={unit} onChange={setUnit} nothingFound="Không có kết quả" />
            <DateInput label="Ngày vào làm" required value={hire} onChange={setHire} {...dateProps} />
          </FormFields>
          <FormBar>
            <Button variant="subtle" color="gray" onClick={() => setEditing(false)}>
              Hủy
            </Button>
            <Button onClick={() => setEditing(false)}>Lưu</Button>
          </FormBar>
        </Stack>
      ) : (
        <FieldList
          stacked
          rows={[
            ['Họ tên', e.name],
            ['Mã nhân viên', e.code],
            ['Phòng ban', e.unit],
            ['Ngày vào làm', formatDate(e.hire)],
            ['Quản lý trực tiếp', e.manager ? <Anchor href="#">{e.manager}</Anchor> : <MissingValue />],
            ['Tài khoản đăng nhập', e.login ?? <MissingValue />],
          ]}
        />
      )}
    </Panel>
  )
}

function MainColumn({ employee: e }: { employee: Employee }) {
  return (
    <Stack gap="md">
      <MainInfoPanel employee={e} />
      <Panel title="Thông tin cá nhân" icon={<IconAddressBook {...icon.text} aria-hidden />}>
        <FieldList
          stacked
          rows={[
            ['Ngày sinh', <MissingValue key="birth" />],
            ['Giới tính', <MissingValue key="gender" />],
            ['Điện thoại', e.phone ? <Anchor href={`tel:${e.phone.replace(/\s/g, '')}`}>{e.phone}</Anchor> : <MissingValue key="phone" />],
            ['Email', e.email ? <Anchor href={`mailto:${e.email}`}>{e.email}</Anchor> : <MissingValue key="email" />],
            ['Địa chỉ', <MissingValue key="address" />, 'wide'],
          ]}
        />
      </Panel>
      <Panel title="Thông tin bảo mật" icon={<IconLock {...icon.text} aria-hidden />}>
        <FieldList
          stacked
          rows={[
            ['Số CCCD', '•••• 1234'],
            ['Số sổ BHXH', '•••• 5678'],
            ['Mã số thuế', <MissingValue key="tax" />],
            ['Tài khoản ngân hàng', <MissingValue key="bank" />],
          ]}
        />
      </Panel>
    </Stack>
  )
}

// A plain digest of what the record's other tabs hold; only the three panels' shape matters here.
function SideColumn() {
  return (
    <>
      <Panel plain title="Hợp đồng hiện hành" actions={<Anchor href="#">Xem tất cả</Anchor>}>
        <Stack gap="xxs">
          <Group justify="space-between" wrap={false}>
            <Text weight={600} render={<span />}>
              <Anchor href="#">HĐ-2024-0012</Anchor>
            </Text>
            <Badge color="success">Đã ghi sổ</Badge>
          </Group>
          <Text size="sm" tone="dimmed">
            Không xác định thời hạn, từ 01/03/2024
          </Text>
        </Stack>
      </Panel>
      <Panel plain title="Nghỉ phép 2026" actions={<Anchor href="#">Xem tất cả</Anchor>}>
        <Text weight={600}>Còn 8,5 ngày</Text>
      </Panel>
      <Panel plain title="Hoạt động gần đây" actions={<Anchor href="#">Xem tất cả</Anchor>}>
        <Text size="sm" tone="dimmed">
          (Hoạt động gần đây sẽ hiện ở đây)
        </Text>
      </Panel>
    </>
  )
}

function DetailScreen({ id, go }: { id: number; go: (route: Route) => void }) {
  const e = employees.find((x) => x.id === id) ?? employees[0]!
  const [tab, setTab] = useState('overview')

  const tabs = [
    { value: 'overview', label: 'Tổng quan', content: <RecordColumns main={<MainColumn employee={e} />} side={<SideColumn />} /> },
    { value: 'contracts', label: 'Hợp đồng (2)', content: <Text tone="dimmed">Nội dung hợp đồng sẽ hiện ở đây.</Text> },
    { value: 'leave', label: 'Nghỉ phép', content: <Text tone="dimmed">Nội dung nghỉ phép sẽ hiện ở đây.</Text> },
    { value: 'dependents', label: 'Người phụ thuộc', content: <Text tone="dimmed">Nội dung người phụ thuộc sẽ hiện ở đây.</Text> },
    { value: 'files', label: 'Tệp đính kèm', content: <Text tone="dimmed">Nội dung tệp đính kèm sẽ hiện ở đây.</Text> },
    { value: 'activity', label: 'Hoạt động', content: <Text tone="dimmed">Nội dung hoạt động sẽ hiện ở đây.</Text> },
  ]

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <Page
        variant="detail"
        header={
          <PageHeader
            title={e.name}
            current
            breadcrumbsLabel="Đường dẫn"
            breadcrumbs={[
              {
                label: 'Nhân viên',
                render: (
                  <a
                    href="#"
                    onClick={(ev) => {
                      ev.preventDefault()
                      go({ page: 'list' })
                    }}
                  />
                ),
              },
            ]}
            back={<PageHeader.BackButton label="Quay lại danh sách nhân viên" onClick={() => go({ page: 'list' })} />}
            leading={<PersonAvatar name={e.name} size={48} />}
            status={<Status status={e.status} />}
            description={[e.code, e.unit, e.login].filter(Boolean).join(' · ')}
            actions={
              <Group gap="xs" wrap={false}>
                <Button variant="default" leftSection={<IconArrowsExchange {...icon.button} aria-hidden />}>
                  Điều chuyển
                </Button>
                <MoreActions label="Thao tác khác" place="page">
                  <Menu.Item color="danger" leftSection={<IconUserOff {...icon.text} aria-hidden />}>
                    Cho nghỉ việc
                  </Menu.Item>
                </MoreActions>
              </Group>
            }
            summary={
              <KeyFacts
                facts={[
                  ['Quản lý trực tiếp', e.manager ? <Anchor href="#">{e.manager}</Anchor> : '—'],
                  ['Thâm niên', '3 năm 2 tháng'],
                  ['Loại hợp đồng', 'Không xác định thời hạn'],
                  ['Phép còn lại 2026', '8,5 ngày'],
                ]}
              />
            }
            tabs={
              <Tabs.List onRule>
                {tabs.map((t) => (
                  <Tabs.Tab key={t.value} value={t.value}>
                    {t.label}
                  </Tabs.Tab>
                ))}
              </Tabs.List>
            }
          />
        }
      >
        {tabs.map((t) => (
          <Tabs.Panel key={t.value} value={t.value}>
            {t.content}
          </Tabs.Panel>
        ))}
      </Page>
    </Tabs>
  )
}

// ---- new -------------------------------------------------------------------------------------

function NewScreen({ go }: { go: (route: Route) => void }) {
  const [name, setName] = useState('')
  const [unit, setUnit] = useState<string | null>(null)
  const [manager, setManager] = useState<string | null>(null)
  const [hire, setHire] = useState<string | null>(null)
  const [birth, setBirth] = useState<string | null>(null)
  const [gender, setGender] = useState<string | null>(null)
  const [sensitiveOpen, setSensitiveOpen] = useState(false)
  const [tried, setTried] = useState(false)

  const mainErrors = tried ? [name, unit, hire].filter((v) => !v).length : 0

  return (
    <Page
      variant="centered"
      header={
        <PageHeader
          title="Thêm nhân viên"
          breadcrumbsLabel="Đường dẫn"
          breadcrumbs={[
            {
              label: 'Nhân viên',
              render: (
                <a
                  href="#"
                  onClick={(ev) => {
                    ev.preventDefault()
                    go({ page: 'list' })
                  }}
                />
              ),
            },
          ]}
        />
      }
    >
      <form
        noValidate
        onSubmit={(ev) => {
          ev.preventDefault()
          setTried(true)
        }}
      >
        <Stack gap="md">
          <FormSection
            title="Thông tin chính"
            description="Những gì mọi nhân viên cần có"
            icon={<IconUser {...icon.text} aria-hidden />}
            badge={mainErrors > 0 && <Badge color="danger" aria-label={`${mainErrors} trường còn lỗi`}>{mainErrors}</Badge>}
          >
            <FormSection.Row>
              <TextInput label="Họ tên" required maxLength={200} value={name} onChange={(e) => setName(e.currentTarget.value)} error={tried && !name ? 'Bắt buộc nhập' : undefined} />
            </FormSection.Row>
            <TextInput label="Mã nhân viên" description="Để trống để hệ thống tự cấp" />
            <Combobox
              label="Phòng ban"
              required
              data={units.map((u) => ({ value: u, label: u }))}
              value={unit}
              onChange={setUnit}
              nothingFound="Không có kết quả"
              error={tried && !unit ? 'Bắt buộc chọn' : undefined}
            />
            <DateInput label="Ngày vào làm" required value={hire} onChange={setHire} {...dateProps} error={tried && !hire ? 'Bắt buộc nhập' : undefined} />
            <Combobox
              label="Quản lý trực tiếp"
              data={employees.map((e) => ({ value: String(e.id), label: `${e.code} · ${e.name}` }))}
              value={manager}
              onChange={setManager}
              nothingFound="Không có kết quả"
              clearable
              clearLabel="Xoá"
            />
            <TextInput label="Tài khoản đăng nhập" description="Tên đăng nhập của người dùng đã có" />
          </FormSection>
          <FormSection title="Thông tin cá nhân" description="Liên hệ và nhân thân" icon={<IconAddressBook {...icon.text} aria-hidden />}>
            <FormSection.Short>
              <DateInput label="Ngày sinh" value={birth} onChange={setBirth} {...dateProps} clearable clearLabel="Xoá" />
            </FormSection.Short>
            <FormSection.Short>
              <Select
                label="Giới tính"
                data={[
                  { value: 'male', label: 'Nam' },
                  { value: 'female', label: 'Nữ' },
                  { value: 'other', label: 'Khác' },
                ]}
                value={gender}
                onChange={setGender}
                clearable
                clearLabel="Xoá"
              />
            </FormSection.Short>
            <FormSection.Short>
              <TextInput label="Điện thoại" type="tel" placeholder="0903 123 456" />
            </FormSection.Short>
            <TextInput label="Email" type="email" />
            <FormSection.Row>
              <TextInput label="Địa chỉ" />
            </FormSection.Row>
          </FormSection>
          <FormSection title="Thông tin bảo mật" description="CCCD, BHXH, thuế, ngân hàng; mỗi lần xem đều được ghi lại" icon={<IconLock {...icon.text} aria-hidden />}>
            {sensitiveOpen ? (
              <>
                <TextInput label="Số CCCD" autoFocus />
                <TextInput label="Số sổ BHXH" />
                <TextInput label="Mã số thuế" />
                <TextInput label="Tài khoản ngân hàng" />
              </>
            ) : (
              <FormSection.Row>
                <Button variant="light" leftSection={<IconPlus {...icon.button} aria-hidden />} onClick={() => setSensitiveOpen(true)}>
                  Thêm thông tin bảo mật
                </Button>
              </FormSection.Row>
            )}
          </FormSection>
          <FormBar>
            <Button variant="subtle" color="gray" onClick={() => go({ page: 'list' })}>
              Hủy
            </Button>
            <Button type="submit">Thêm nhân viên</Button>
          </FormBar>
        </Stack>
      </form>
    </Page>
  )
}

// ---- stories -----------------------------------------------------------------------------------

function App({ start }: { start: Route }) {
  const [route, go] = useState<Route>(start)
  return (
    <Shell>
      {route.page === 'list' && <ListScreen go={go} />}
      {route.page === 'detail' && <DetailScreen id={route.id} go={go} />}
      {route.page === 'new' && <NewScreen go={go} />}
    </Shell>
  )
}

const meta: Meta<typeof App> = {
  title: 'Screens/Employees',
  component: App,
  parameters: { layout: 'fullscreen' },
}

export default meta

type Story = StoryObj<typeof App>

export const List: Story = { args: { start: { page: 'list' } } }
export const Record: Story = { args: { start: { page: 'detail', id: 7 } } }
export const New: Story = { args: { start: { page: 'new' } } }
