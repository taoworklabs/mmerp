import type * as React from 'react'

// Lets a style object carry CSS custom properties (--mmerp-*) alongside standard CSS properties,
// so a token value can be passed through inline style instead of a colour or size literal.
export type StyleVars = React.CSSProperties & Record<`--mmerp-${string}`, string | number | undefined>
