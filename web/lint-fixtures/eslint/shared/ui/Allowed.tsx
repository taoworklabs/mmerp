// Allowed: shared/ui builds on any kit component and may style inline.
import { Alert } from '@mmerp/ui'

export const Fixture = ({ label }: { label: string }) => <Alert style={{ margin: 0 }}>{label}</Alert>
