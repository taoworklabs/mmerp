import { breakpoint, type ColorName, icon as kitIcon, tokens } from '@mmerp/ui'
import { Combobox, createTheme, Drawer, InputWrapper, Menu, Modal, NavLink, Popover, rem, ScrollArea, SegmentedControl, Select, Tabs, Tooltip, type CSSVariablesResolver, type MantineColorsTuple } from '@mantine/core'
import segmented from './SegmentedControl.module.css'

// The numbers live in @mmerp/ui; this file only hands them to Mantine until Mantine is gone.
const { ramp, semantic } = tokens.color
const ramps = (name: ColorName) => [...ramp[name]] as unknown as MantineColorsTuple
const primary = ramps('primary')
const success = ramps('success')
const warning = ramps('warning')
const danger = ramps('danger')
const info = ramps('info')
const discovery = ramps('discovery')
const gray = ramps('gray')
const rems = <T extends Record<string, number>>(o: T) => Object.fromEntries(Object.entries(o).map(([k, v]) => [k, rem(v)])) as Record<keyof T, string>

export const theme = createTheme({
  primaryColor: 'primary',
  primaryShade: 6,
  autoContrast: true,
  respectReducedMotion: true,
  colors: { primary, success, warning, danger, info, discovery, gray },
  fontFamily: tokens.fontFamily,
  fontSizes: rems(tokens.fontSize),
  lineHeights: Object.fromEntries(Object.entries(tokens.lineHeight).map(([k, v]) => [k, String(v)])),
  spacing: rems(tokens.space),
  radius: rems(tokens.radius),
  defaultRadius: 'md',
  shadows: { xs: tokens.shadow.xs, sm: tokens.shadow.sm, md: tokens.shadow.md, lg: tokens.shadow.lg, xl: tokens.shadow.xl },
  // md is where the navigation becomes a drawer (1024px), lg where it stops auto-collapsing (1280px).
  components: {
    // Labels are quieter than the values they name.
    InputWrapper: InputWrapper.extend({
      defaultProps: { inputWrapperOrder: ['label', 'input', 'description', 'error'] },
      styles: { label: { fontSize: 'var(--mantine-font-size-xs)', fontWeight: 600, lineHeight: 'var(--mantine-line-height-xs)', color: 'var(--mantine-color-dimmed)', marginBottom: 'var(--mantine-spacing-xxs)' } },
    }),
    // The chosen option is tinted, without a check mark that pushes every label sideways.
    Select: Select.extend({ defaultProps: { withCheckIcon: false } }),
    // A modal is its own small page: a 20px heading in a header band set off by a rule (its footer
    // band is ModalFooter), 24px of room at the sides and 16px above and below. Its fields are
    // 40px, like a form section's; ModalFooter keeps its buttons at 30px.
    Modal: Modal.extend({
      defaultProps: { radius: 'md', padding: 'lg' },
      styles: {
        header: { minHeight: 0, paddingBlock: 'var(--mantine-spacing-md)', borderBottom: '1px solid var(--mantine-color-default-border)' },
        body: { paddingBlock: 'var(--mantine-spacing-md)', '--mmerp-control-height': rem(tokens.controlHeight.lg) },
        title: { fontWeight: 650, fontSize: 'var(--mantine-font-size-xl)', lineHeight: 'var(--mantine-line-height-xl)' },
      },
    }),
    Drawer: Drawer.extend({ styles: { title: { fontWeight: 650, fontSize: 'var(--mantine-font-size-lg)' } } }),
    Popover: Popover.extend({ defaultProps: { radius: 'lg' } }),
    Combobox: Combobox.extend({ defaultProps: { radius: 'lg' } }),
    Menu: Menu.extend({ defaultProps: { radius: 'lg' } }),
    SegmentedControl: SegmentedControl.extend({ classNames: segmented }),
    Tooltip: Tooltip.extend({ defaultProps: { radius: 'sm' } }),
    // Thin scrollbars, so they do not compete with the content.
    ScrollArea: ScrollArea.extend({ defaultProps: { scrollbarSize: 8 } }),
    Tabs: Tabs.extend({ styles: { tab: { fontWeight: 500 } } }),
    NavLink: NavLink.extend({ styles: { root: { borderRadius: 'var(--mantine-radius-md)', minHeight: 'var(--mmerp-control-height)' }, label: { fontWeight: 500 } } }),
  },
  breakpoints: { ...breakpoint },
  headings: {
    fontWeight: String(tokens.fontWeight.heading),
    sizes: Object.fromEntries(Object.entries(tokens.heading).map(([k, h]) => [k, { fontSize: rem(h.size), lineHeight: String(h.lineHeight) }])),
  },
})

// Icon sizes and filter bar sizes come from the kit.
export const icon = kitIcon

export const filterWidth = tokens.filterWidth

// A tinted ("light") variant: the palest step behind, a dark step for text, so it reads at 4.5:1.
function tint(name: string, ramp: MantineColorsTuple, bg = 0) {
  return {
    [`--mantine-color-${name}-light`]: ramp[bg],
    [`--mantine-color-${name}-light-hover`]: ramp[bg + 1],
    [`--mantine-color-${name}-light-color`]: ramp[7],
  }
}

// Semantic tokens on top of Mantine's own variables. No dark mode yet: names are by role,
// so a dark set only fills the dark block.
export const cssVariablesResolver: CSSVariablesResolver = () => ({
  // Lengths and the --mmerp-* semantic tokens come from the kit stylesheet; only Mantine's own variables are set here.
  variables: {},
  light: {
    '--mantine-color-text': semantic.text,
    '--mantine-color-dimmed': semantic.dimmed,
    '--mantine-color-placeholder': semantic.placeholder,
    '--mantine-color-disabled': semantic.disabled,
    '--mantine-color-default-border': semantic.border,
    '--mantine-color-body': semantic.body,
    '--mantine-color-error': semantic.error,
    ...tint('primary', primary),
    ...tint('success', success),
    ...tint('warning', warning),
    ...tint('danger', danger),
    ...tint('info', info),
    ...tint('discovery', discovery),
    // The palest neutral vanishes on white; neutral tints start one step darker.
    ...tint('gray', gray, tokens.color.tint.gray),
  },
  dark: {},
})
