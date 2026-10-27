// Planted: an area importing a kit block outside the allowlist.
import { Button, DataTable } from '@mmerp/ui'

export const Fixture = ({ label }: { label: string }) => (
  <DataTable>
    <Button>{label}</Button>
  </DataTable>
)
