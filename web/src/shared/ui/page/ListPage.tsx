import { Button, Group, Pagination, Select, Text } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { pageSizes } from '@/shared/url/listParams'
import classes from './ListPage.module.css'
import { Page, type PageHeaderProps } from './Page'

type Paging = { page: number; pageSize: number; total: number; onPage: (page: number) => void; onPageSize: (size: number) => void }

// ListPage: title and the create button, optional view tabs in place of the header's rule, a filter bar, the table, then paging. The filter bar's
// chips wrap onto more lines when they do not fit; tools (saved views, columns) sit at its right end
// from 1024px; bulk, while rows are selected, takes the whole bar's place. From 1024px the page fills
// the window and only its content scrolls.
export function ListPage({
  action,
  tabs,
  filters,
  tools,
  bulk,
  paging,
  onClearFilters,
  children,
  ...header
}: Omit<PageHeaderProps, 'actions' | 'leading'> & {
  action?: ReactNode
  // Ways to view the same list (tree, table): a Tabs element whose list replaces the header's rule.
  tabs?: ReactNode
  filters?: ReactNode
  // Given while any filter is set: a "Clear filters" button ends the bar.
  onClearFilters?: () => void
  tools?: ReactNode
  bulk?: ReactNode
  paging?: Paging
  children: ReactNode
}) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  return (
    <Page {...header} actions={action} divider={!tabs} className={wide ? classes.frame : undefined}>
      {tabs}
      {wide && bulk
        ? bulk
        : (filters || (wide && tools)) && (
            <Group gap="xs" align="flex-start" justify="space-between" wrap="nowrap">
              <Group gap="xs" align="flex-end" flex={1} miw={0} role="search" className={classes.filters}>
                {filters}
                {onClearFilters && (
                  <Button variant="default" onClick={onClearFilters}>
                    {t('shared.list.clear_filters')}
                  </Button>
                )}
              </Group>
              {wide && tools}
            </Group>
          )}
      <div className={classes.content}>{children}</div>
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
