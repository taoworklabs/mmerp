import type { useRender } from '@base-ui/react/use-render'

// Re-export of Base UI's render-prop type so every polymorphic primitive shares one name.
export type RenderProp<State = Record<string, unknown>> = useRender.RenderProp<State>
