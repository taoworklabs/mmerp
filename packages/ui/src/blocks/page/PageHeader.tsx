import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { ArrowLeftIcon, ChevronRightIcon } from '../../internal/icons.tsx'
import { ActionIcon, type ActionIconProps } from '../../primitives/input/ActionIcon.tsx'
import { Breadcrumbs } from '../../primitives/navigation/Breadcrumbs.tsx'
import { Tooltip } from '../../primitives/overlay/Tooltip.tsx'
import { Anchor } from '../../primitives/typography/Anchor.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { Title } from '../../primitives/typography/Title.tsx'
import { icon } from '../../tokens/tokens.ts'
import styles from './PageHeader.module.css'

export interface PageHeaderBreadcrumb {
  label: string
  // An <a>/<Link> element, without children: PageHeader fills them in with the label.
  render: React.ReactElement
}

// Either no breadcrumbs, or breadcrumbs with the nav's accessible name: never an unnamed landmark.
type PageHeaderBreadcrumbsProps = { breadcrumbs?: undefined; breadcrumbsLabel?: undefined } | { breadcrumbs: PageHeaderBreadcrumb[]; breadcrumbsLabel: string }

export type PageHeaderProps = Omit<React.ComponentPropsWithRef<'div'>, 'title'> &
  PageHeaderBreadcrumbsProps & {
    title: string
    // The breadcrumb ends with the page's own title.
    current?: boolean
    // The back control, built by the caller: history logic (where "back" goes) is theirs.
    back?: React.ReactElement
    leading?: React.ReactNode
    status?: React.ReactNode
    description?: React.ReactNode
    actions?: React.ReactNode
    summary?: React.ReactNode
    // A Tabs.List (inside a Tabs around the page) that sits on the header's rule.
    tabs?: React.ReactNode
    // No rule under the header: the content draws its own frame.
    plain?: boolean
  }

function PageHeaderRoot({
  title,
  breadcrumbs,
  breadcrumbsLabel,
  current,
  back,
  leading,
  status,
  description,
  actions,
  summary,
  tabs,
  plain,
  className,
  ...props
}: PageHeaderProps) {
  return (
    // data-page-header: a list page drops the header's rule, its table's column headers taking over.
    <div className={cx(styles.root, className)} data-page-header="" data-plain={plain ? '' : undefined} data-tabs={tabs ? '' : undefined} {...props}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs aria-label={breadcrumbsLabel} currentLast={Boolean(current)} separator={<ChevronRightIcon />}>
          {breadcrumbs.map((crumb, index) => (
            <Anchor key={index} tone="dimmed" render={crumb.render}>
              {crumb.label}
            </Anchor>
          ))}
          {current && (
            <Text size="sm" weight={500} truncate render={<span />}>
              {title}
            </Text>
          )}
        </Breadcrumbs>
      )}
      <div className={styles.topRow}>
        <div className={styles.titleGroup}>
          {back}
          {leading}
          <div className={styles.titleColumn}>
            <div className={styles.titleRow}>
              <Title order={1} className={styles.title}>
                {title}
              </Title>
              {status}
            </div>
            {description && (
              <Text size="sm" tone="dimmed" render={<div />}>
                {description}
              </Text>
            )}
          </div>
        </div>
        {actions && <div className={styles.actions}>{actions}</div>}
      </div>
      {summary}
      {tabs && <div className={styles.tabs}>{tabs}</div>}
    </div>
  )
}

export interface PageHeaderBackButtonProps extends Omit<ActionIconProps, 'aria-label' | 'render'> {
  // Doubles as the accessible name; a plain ActionIconProps aria-label would be redundant.
  label: string
  render?: React.ReactElement
}

function BackButton({ label, render, className, ...props }: PageHeaderBackButtonProps) {
  return (
    <Tooltip label={label}>
      <ActionIcon variant="default" aria-label={label} render={render} className={cx(styles.backButton, className)} {...props}>
        <ArrowLeftIcon size={icon.button.size} />
      </ActionIcon>
    </Tooltip>
  )
}

export const PageHeader = Object.assign(PageHeaderRoot, { BackButton })
