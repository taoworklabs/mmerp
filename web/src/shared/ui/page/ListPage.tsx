import { Button, Drawer, Group, Pagination, Select, Stack, Text } from '@mantine/core'
import { useDisclosure, useMediaQuery } from '@mantine/hooks'
import { IconFilter } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { pageSizes } from '@/shared/url/listParams'
import { icon } from '../theme'
import classes from './ListPage.module.css'
import { Page, type PageHeaderProps } from './Page'

type Paging = { page: number; pageSize: number; total: number; onPage: (page: number) => void; onPageSize: (size: number) => void }

// ListPage: title and the create button, a filter bar, the table, then paging.
// Below 1024px the filter bar moves into a drawer behind a "Filter" button. tools (saved views,
// columns) sit at the bar's right end; bulk, while rows are selected, takes the whole bar's place.
export function ListPage({
  action,
  filters,
  tools,
  bulk,
  paging,
  children,
  ...header
}: Omit<PageHeaderProps, 'actions' | 'leading'> & {
  action?: ReactNode
  filters?: ReactNode
  tools?: ReactNode
  bulk?: ReactNode
  paging?: Paging
  children: ReactNode
}) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const [open, { toggle, close }] = useDisclosure()
  return (
    <Page {...header} actions={action}>
      {wide && bulk ? (
        bulk
      ) : wide && (filters || tools) ? (
        <Group gap="xs" align="flex-end" justify="space-between" wrap="nowrap">
          <Group gap="xs" align="flex-end" role="search" className={classes.filters}>
            {filters}
          </Group>
          {tools}
        </Group>
      ) : (
        filters && (
          <>
            <Group>
              <Button variant="default" leftSection={<IconFilter {...icon.button} />} onClick={toggle}>
                {t('shared.list.filter')}
              </Button>
            </Group>
            <Drawer opened={open} onClose={close} title={t('shared.list.filter')} position="bottom">
              <Stack gap="sm" role="search" className={classes.filters}>
                {filters}
                <Button onClick={close}>{t('shared.list.apply')}</Button>
              </Stack>
            </Drawer>
          </>
        )
      )}
      {children}
      {paging && paging.total > 0 && <PagingBar {...paging} />}
    </Page>
  )
}

function PagingBar({ page, pageSize, total, onPage, onPageSize }: Paging) {
  const { t } = useTranslation()
  return (
    <div className={classes.paging}>
      <Text size="sm" c="dimmed">
        {t('shared.list.total', { total: formatNumber(total) })}
      </Text>
      <Pagination
        size="input-sm"
        total={Math.ceil(total / pageSize)}
        value={page}
        onChange={onPage}
        classNames={{ control: `${classes.ghost} ${classes.page}` }}
        getControlProps={(control) => ({ 'aria-label': t(`shared.list.${control}`) })}
        getItemProps={(p) => ({ 'aria-label': t('shared.list.page', { page: p }) })}
      />
      <Select
        aria-label={t('shared.list.page_size')}
        data={pageSizes.map((n) => ({ value: String(n), label: t('shared.list.per_page', { n }) }))}
        value={String(pageSize)}
        onChange={(v) => v && onPageSize(Number(v))}
        allowDeselect={false}
        w={130}
        className={classes.pageSize}
        classNames={{ input: classes.ghost }}
      />
    </div>
  )
}
