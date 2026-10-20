// Allowed: shared/ui builds on any Mantine component and may style inline.
import { Alert } from '@mantine/core'

export const Fixture = ({ label }: { label: string }) => <Alert style={{ margin: 0 }}>{label}</Alert>
