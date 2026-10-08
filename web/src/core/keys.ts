import { documentKeys } from '@/shared/document'
import { orgUnitKeys } from '@/shared/ui/form'

// Query keys of the core area; they start with the product name.
export const coreKeys = {
  orgUnits: orgUnitKeys.all,
  users: {
    all: () => ['core', 'users'] as const,
    detail: (id: number) => ['core', 'users', id] as const,
    roles: (id: number) => ['core', 'users', id, 'roles'] as const,
  },
  roles: documentKeys.roles,
  periodLocks: () => ['core', 'period-locks'] as const,
  settings: (legalEntity: number) => ['core', 'settings', legalEntity] as const,
  mailServer: () => ['core', 'mail-server'] as const,
}
