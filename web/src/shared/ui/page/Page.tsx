import { Page as KitPage, PageHeader as KitPageHeader } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useLocation, useNavigate } from 'react-router'

export type Crumb = { label: string; to: string }

export type PageHeaderProps = {
  title: string
  // Where this page sits: links to the pages above it, the page itself excluded.
  breadcrumbs?: Crumb[]
  // One line under the title: what the list holds, or the record's key facts.
  description?: ReactNode
  // Shown left of the title, e.g. a person's avatar.
  leading?: ReactNode
  // Right of the title: the record's status badge.
  status?: ReactNode
  actions?: ReactNode
  // Under the title, over the tabs: the record's key facts (KeyFacts).
  summary?: ReactNode
  // A Tabs.List (inside a Tabs around the page) that sits on the header's rule.
  tabs?: ReactNode
  // No rule under the header: the content draws its own frame.
  plain?: boolean
  // One record's page (DetailPage, DocumentPage): the breadcrumb ends with the page itself and a
  // back button leads the title.
  withBack?: boolean
  // The breadcrumb ends with the page itself, without the back button: a record page whose
  // breadcrumb already leads back.
  current?: boolean
}

// readingWidth is the width of a side panel (approval, attachments) shown full width as a tab.
export const readingWidth = 480

// Page is the frame every page template builds on: header (breadcrumbs, title, at most one
// primary action), then content. A record's or a form's page is centred at the width it stops at
// (centered: the content width, or 'detail' for the wider record width), its header too, so both
// share a left edge; lists and dashboards use the whole width.
export function Page({ children, className, centered, ...header }: PageHeaderProps & { children: ReactNode; className?: string; centered?: boolean | 'detail' }) {
  return (
    <KitPage variant={centered === 'detail' ? 'detail' : centered ? 'centered' : 'full'} className={className} header={<PageHeader {...header} />}>
      {children}
    </KitPage>
  )
}

export function PageHeader({ breadcrumbs, withBack, current, ...props }: PageHeaderProps) {
  const { t } = useTranslation()
  const parent = breadcrumbs?.at(-1)
  const crumbs = breadcrumbs?.length ? { breadcrumbs: breadcrumbs.map((c) => ({ label: c.label, render: <Link to={c.to} /> })), breadcrumbsLabel: t('shared.page.breadcrumb') } : {}
  return <KitPageHeader {...props} {...crumbs} current={withBack || current} back={withBack && parent ? <BackButton to={parent.to} /> : undefined} />
}

// BackButton returns to where the user came from, the list with its filters and page; opened
// directly (a link from elsewhere, a new tab), it goes to the page above. A real link, so it also
// opens in a new tab.
function BackButton({ to }: { to: string }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  return (
    <KitPageHeader.BackButton
      label={t('shared.page.back')}
      render={<Link to={to} />}
      onClick={(e) => {
        // The first entry of the session has the default key: there is nothing in the app to go back to.
        if (location.key === 'default' || e.metaKey || e.ctrlKey || e.shiftKey) return
        e.preventDefault()
        void navigate(-1)
      }}
    />
  )
}
