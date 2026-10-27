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
  delegations: {
    list: () => ['core', 'delegations'] as const,
    meta: () => ['core', 'delegations', 'meta'] as const,
    users: () => ['core', 'delegations', 'users'] as const,
  },
  sessions: () => ['core', 'me', 'sessions'] as const,
  periodLocks: () => ['core', 'period-locks'] as const,
  codeSeries: () => ['core', 'code-series'] as const,
  settings: (legalEntity: number) => ['core', 'settings', legalEntity] as const,
  mailServer: () => ['core', 'mail-server'] as const,
  printTemplates: {
    all: () => ['core', 'print-templates'] as const,
    detail: (code: string) => ['core', 'print-templates', code] as const,
  },
}
