import { Button } from '@mantine/core'
import { IconPencil } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { OrgTree } from '@/shared/org'
import { ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'

// OrgStructurePage: the org tree, read-only; the tree is edited only in administration.
export function OrgStructurePage() {
  const { t } = useTranslation()
  const canEdit = useCan('core.org.manage')
  return (
    <ListPage
      title={t('hrm.org.title')}
      description={t('hrm.org.description')}
      action={
        canEdit && (
          <Button component={Link} to="/admin/org-units" variant="default" leftSection={<IconPencil {...icon.button} />}>
            {t('hrm.org.edit')}
          </Button>
        )
      }
    >
      <OrgTree label={t('hrm.org.title')} />
    </ListPage>
  )
}
