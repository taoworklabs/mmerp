import { Badge, Button, createTheme, InputWrapper, Modal, NavLink, Paper, rem, Select, Tabs, type CSSVariablesResolver, type MantineColorsTuple } from '@mantine/core'

// Index 6 is the shade Mantine uses for filled components.
const primary: MantineColorsTuple = ['#EFF6FF', '#DBEAFE', '#BFDBFE', '#93C5FD', '#60A5FA', '#2563EB', '#1E40AF', '#1E3A8A', '#172554', '#0F1B3D']
const success: MantineColorsTuple = ['#F0FDF4', '#DCFCE7', '#BBF7D0', '#86EFAC', '#4ADE80', '#22C55E', '#16A34A', '#15803D', '#166534', '#14532D']
const warning: MantineColorsTuple = ['#FFFBEB', '#FEF3C7', '#FDE68A', '#FCD34D', '#FBBF24', '#F59E0B', '#D97706', '#B45309', '#92400E', '#78350F']
const danger: MantineColorsTuple = ['#FEF2F2', '#FEE2E2', '#FECACA', '#FCA5A5', '#F87171', '#EF4444', '#DC2626', '#B91C1C', '#991B1B', '#7F1D1D']
const info: MantineColorsTuple = ['#F0F9FF', '#E0F2FE', '#BAE6FD', '#7DD3FC', '#38BDF8', '#0EA5E9', '#0284C7', '#0369A1', '#075985', '#0C4A6E']

const radius = rem(6)

export const theme = createTheme({
  primaryColor: 'primary',
  primaryShade: 6,
  autoContrast: true,
  respectReducedMotion: true,
  colors: { primary, success, warning, danger, info },
  fontFamily: "'Inter Variable', system-ui, sans-serif",
  fontSizes: { xs: rem(12), sm: rem(14), md: rem(14), lg: rem(16), xl: rem(20) },
  lineHeights: { xs: '1.5', sm: '1.5', md: '1.5', lg: '1.5', xl: '1.5' },
  spacing: { xxs: rem(4), xs: rem(8), sm: rem(12), md: rem(16), lg: rem(24), xl: rem(32) },
  radius: { xs: radius, sm: radius, md: radius, lg: radius, xl: radius },
  defaultRadius: 'sm',
  // md is where the navigation becomes a drawer (1024px), lg where it stops auto-collapsing (1280px).
  // Content regions sit on the surface colour, not the app background.
  components: {
    Paper: Paper.extend({ defaultProps: { bg: 'var(--mmerp-color-surface)' } }),
    // Labels are quieter than the values they name.
    InputWrapper: InputWrapper.extend({ defaultProps: { inputWrapperOrder: ['label', 'input', 'description', 'error'] }, styles: { label: { fontSize: rem(13), fontWeight: 500, color: 'var(--mantine-color-dimmed)', marginBottom: rem(4) } } }),
    Button: Button.extend({ styles: { root: { fontWeight: 500 } } }),
    // The chosen option is tinted, without a check mark that pushes every label sideways.
    Select: Select.extend({ defaultProps: { withCheckIcon: false } }),
    Badge: Badge.extend({ defaultProps: { variant: 'light', radius: 'sm' }, styles: { root: { textTransform: 'none', fontWeight: 500 } } }),
    Modal: Modal.extend({ styles: { title: { fontWeight: 600, fontSize: rem(16) } } }),
    Tabs: Tabs.extend({ styles: { tab: { fontWeight: 500 } } }),
    NavLink: NavLink.extend({ styles: { root: { borderRadius: 'var(--mantine-radius-sm)', minHeight: rem(32) }, label: { fontWeight: 500 } } }),
  },
  breakpoints: { xs: '36em', sm: '48em', md: '64em', lg: '80em', xl: '96em' },
  headings: {
    fontWeight: '600',
    sizes: {
      h1: { fontSize: rem(22), lineHeight: '1.4' },
      h2: { fontSize: rem(16), lineHeight: '1.5' },
      h3: { fontSize: rem(14), lineHeight: '1.5' },
    },
  },
})

// Icon sizes: 16px in text, tables and xs buttons, 18px in sm buttons.
export const icon = {
  text: { size: 16, stroke: 1.75 },
  button: { size: 18, stroke: 1.75 },
} as const

// Semantic tokens on top of Mantine's own variables. No dark mode yet.
export const cssVariablesResolver: CSSVariablesResolver = () => ({
  variables: {},
  light: {
    '--mantine-color-text': '#0F172A',
    '--mantine-color-dimmed': '#475569',
    '--mantine-color-placeholder': '#94A3B8',
    '--mantine-color-default-border': '#E2E8F0',
    // Content and navigation sit on white; table headers use the subtle tone.
    '--mantine-color-body': '#FFFFFF',
    '--mmerp-color-subtle': '#F8FAFC',
    '--mmerp-color-surface': '#FFFFFF',
    // Required marks and field errors use the danger token, not Mantine's lighter red.
    '--mantine-color-error': '#DC2626',
  },
  dark: {},
})
