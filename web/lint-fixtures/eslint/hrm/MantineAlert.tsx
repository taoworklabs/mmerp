// Planted: an area importing a Mantine component outside the allowlist.
import { Alert, Button } from '@mantine/core'

export const Fixture = ({ label }: { label: string }) => (
  <Alert>
    <Button>{label}</Button>
  </Alert>
)
