import { Group, Text, ThemeIcon, type ThemeIconProps } from '@mantine/core'
import { IconBuildingSkyscraper, IconHierarchy2, IconMapPin, IconUsersGroup } from '@tabler/icons-react'
import type { OrgUnit } from '@/shared/api/core'
import { icon } from './theme'

// Each kind of unit has its own icon and accent colour, so a tree reads at a glance.
const kindLook = {
  group: { Icon: IconHierarchy2, color: 'grape' },
  company: { Icon: IconBuildingSkyscraper, color: 'blue' },
  branch: { Icon: IconMapPin, color: 'teal' },
  department: { Icon: IconUsersGroup, color: 'orange' },
} as const

// OrgUnitIcon is a unit's kind as its tinted icon: in the tree, and in a field holding the unit.
export function OrgUnitIcon({ kind, size = 24 }: { kind: OrgUnit['kind']; size?: ThemeIconProps['size'] }) {
  const { Icon, color } = kindLook[kind]
  return (
    <ThemeIcon variant="light" color={color} size={size} radius="sm" aria-hidden>
      <Icon {...icon.text} />
    </ThemeIcon>
  )
}

// OrgUnitName is an org unit as one row of the tree: indented by depth, with its kind's icon.
// Legal entities and groups are bold, as the roots of what sits under them.
export function OrgUnitName({ unit, depth = 0 }: { unit: Pick<OrgUnit, 'kind' | 'name'>; depth?: number }) {
  return (
    <Group gap="xs" wrap="nowrap" pl={`calc(var(--mantine-spacing-lg) * ${depth})`}>
      <OrgUnitIcon kind={unit.kind} />
      <Text inherit fw={unit.kind === 'company' || unit.kind === 'group' ? 600 : undefined} truncate>
        {unit.name}
      </Text>
    </Group>
  )
}
