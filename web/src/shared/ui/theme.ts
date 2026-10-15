import { Badge, Button, Combobox, createTheme, Drawer, InputWrapper, Menu, Modal, NavLink, Paper, Popover, rem, ScrollArea, SegmentedControl, Select, Tabs, Tooltip, type CSSVariablesResolver, type MantineColorsTuple } from '@mantine/core'
import segmented from './SegmentedControl.module.css'

// Ramps run from palest (0) to darkest (9). Index 6 is the shade filled components use,
// 7 the text on a tinted background, 0 the tint itself.
const primary: MantineColorsTuple = ['#E9F2FE', '#CFE1FD', '#8FB8F6', '#669DF1', '#4688EC', '#357DE8', '#1868DB', '#1558BC', '#123263', '#1C2B42']
const success: MantineColorsTuple = ['#EFFFD6', '#D3F1A7', '#B3DF72', '#94C748', '#82B536', '#6A9A23', '#5B7F24', '#4C6B1F', '#37471F', '#28311B']
const warning: MantineColorsTuple = ['#FFF5DB', '#FCE4A6', '#FBC828', '#FCA700', '#F68909', '#E06C00', '#BD5B00', '#9E4C00', '#693200', '#3A2C1F']
const danger: MantineColorsTuple = ['#FFECEB', '#FFD5D2', '#FD9891', '#F87168', '#F15B50', '#E2483D', '#C9372C', '#AE2E24', '#5D1F1A', '#42221F']
const info = primary
// Discovery marks what is new or in beta, nothing else.
const discovery: MantineColorsTuple = ['#F8EEFE', '#EED7FC', '#D8A0F7', '#C97CF4', '#BF63F3', '#AF59E1', '#964AC0', '#803FA5', '#48245D', '#35243F']
// The one neutral ramp; it replaces Mantine's gray, whose steps Mantine reads by position:
// 3 frames and table rules, 4 checkbox and radio edges, so 2 and 3 share the light border tone.
// Text fields take the quieter --mmerp-input-border instead (global.css).
const gray: MantineColorsTuple = ['#F8F8F8', '#F0F1F2', '#DDDEE1', '#DDDEE1', '#B4B7BD', '#7D818A', '#6B6E76', '#505258', '#3B3D42', '#292A2E']

export const theme = createTheme({
  primaryColor: 'primary',
  primaryShade: 6,
  autoContrast: true,
  respectReducedMotion: true,
  colors: { primary, success, warning, danger, info, discovery, gray },
  fontFamily: "'Inter Variable', system-ui, sans-serif",
  // Lines sit on a 4px grid: 12/16, 14/20, 16/24, 20/24.
  fontSizes: { xs: rem(12), sm: rem(14), md: rem(14), lg: rem(16), xl: rem(20) },
  lineHeights: { xs: '1.34', sm: '1.43', md: '1.43', lg: '1.5', xl: '1.2' },
  spacing: { xxxs: rem(2), xxs: rem(4), xs: rem(8), sm: rem(12), md: rem(16), lg: rem(24), xl: rem(32), xxl: rem(48) },
  // Radius follows role: tags 4; controls, frames and modals 6; menus 8.
  radius: { xs: rem(2), sm: rem(4), md: rem(6), lg: rem(8), xl: rem(12) },
  defaultRadius: 'md',
  // Only overlays (menus, popovers, modals, toasts) cast a shadow, and a footer while content scrolls
  // under it (--mmerp-shadow-footer); xs/sm are for a card lifted off the page.
  shadows: {
    xs: '0 1px 1px #1E1F2140, 0 0 1px #1E1F214F',
    sm: '0 1px 1px #1E1F2140, 0 0 1px #1E1F214F',
    md: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
    lg: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
    xl: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
  },
  // md is where the navigation becomes a drawer (1024px), lg where it stops auto-collapsing (1280px).
  // Content regions sit on the surface colour, not the app background.
  components: {
    Paper: Paper.extend({ defaultProps: { bg: 'var(--mmerp-elevation-surface)', radius: 'md' } }),
    // Labels are quieter than the values they name.
    InputWrapper: InputWrapper.extend({
      defaultProps: { inputWrapperOrder: ['label', 'input', 'description', 'error'] },
      styles: { label: { fontSize: 'var(--mantine-font-size-xs)', fontWeight: 600, lineHeight: 'var(--mantine-line-height-xs)', color: 'var(--mantine-color-dimmed)', marginBottom: 'var(--mantine-spacing-xxs)' } },
    }),
    Button: Button.extend({ styles: { root: { fontWeight: 500 } } }),
    // The chosen option is tinted, without a check mark that pushes every label sideways.
    Select: Select.extend({ defaultProps: { withCheckIcon: false } }),
    // Tags stand out from the rows around them: a tint one step deeper than other tinted parts
    // (two for neutral, whose first step is the sunken surface), with the darkest readable text.
    Badge: Badge.extend({
      defaultProps: { variant: 'light', radius: 'sm' },
      styles: { root: { textTransform: 'none', fontWeight: 500, fontSize: 'var(--mantine-font-size-xs)' } },
      vars: (_, { color = 'primary', variant = 'light' }) =>
        variant === 'light' ? { root: { '--badge-bg': `var(--mantine-color-${color}-${color === 'gray' ? 2 : 1})`, '--badge-color': `var(--mantine-color-${color}-8)` } } : { root: {} },
    }),
    // A modal is its own small page: a 20px heading in a header band set off by a rule (its footer
    // band is ModalFooter), 24px of room at the sides and 16px above and below. Its fields are
    // 40px, like a form section's; ModalFooter keeps its buttons at 30px.
    Modal: Modal.extend({
      defaultProps: { radius: 'md', padding: 'lg' },
      styles: {
        header: { minHeight: 0, paddingBlock: 'var(--mantine-spacing-md)', borderBottom: '1px solid var(--mantine-color-default-border)' },
        body: { paddingBlock: 'var(--mantine-spacing-md)', '--mmerp-control-height': rem(40) },
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
  breakpoints: { xs: '36em', sm: '48em', md: '64em', lg: '80em', xl: '96em' },
  headings: {
    fontWeight: '650',
    sizes: {
      h1: { fontSize: rem(24), lineHeight: '1.17' },
      h2: { fontSize: rem(16), lineHeight: '1.25' },
      h3: { fontSize: rem(14), lineHeight: '1.43' },
    },
  },
})

// Icon sizes: 16px in text, tables and xs buttons, 18px in sm buttons, 24px in empty states.
export const icon = {
  text: { size: 16, stroke: 1.5 },
  button: { size: 18, stroke: 1.5 },
  empty: { size: 24, stroke: 1.5 },
} as const

// Filter bar sizes: the search box (it narrows to searchMin when the bar is full), the widest a
// filter chip grows before its value truncates, and a chip's dropdown.
export const filterWidth = { search: 280, searchMin: 160, chip: 320, menu: 240 } as const

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
  variables: {
    // The one control height (size sm), wherever the control sits; global.css applies it.
    '--mmerp-control-height': rem(30),
    // Forms and a document's main column stop here, so fields stay near their labels on wide screens.
    '--mmerp-content-width': rem(1200),
    // The inbox list, and the least a document's side column (approval, attachments, activity) shrinks to.
    '--mmerp-side-width': rem(360),
    // A record's own values beside its tabs: label–value pairs need more room than side panels.
    '--mmerp-aside-width': rem(480),
  },
  light: {
    '--mantine-color-text': gray[9],
    '--mantine-color-dimmed': gray[7],
    '--mantine-color-placeholder': gray[6],
    '--mantine-color-disabled': gray[1],
    // Dividers, table and frame borders: translucent, so they sit on any surface.
    '--mantine-color-default-border': '#0B120E24',
    '--mantine-color-body': '#FFFFFF',
    '--mmerp-elevation-surface': '#FFFFFF',
    '--mmerp-elevation-surface-sunken': gray[0],
    // A table's column header row: a step deeper than the sunken tone of its group and total rows.
    '--mmerp-table-header': '#F4F5F6',
    // Edges of text fields: a step stronger than border, quieter than the checkbox edge (gray[4]).
    '--mmerp-input-border': '#0B120E33',
    // A footer pinned over content that scrolls under it (a page form's footer): shadow upwards only.
    '--mmerp-shadow-footer': '0 -6px 8px -6px #1E1F2126',
    // Required marks and field errors use the danger text shade.
    '--mantine-color-error': danger[7],
    ...tint('primary', primary),
    ...tint('success', success),
    ...tint('warning', warning),
    ...tint('danger', danger),
    ...tint('info', info),
    ...tint('discovery', discovery),
    // The palest neutral vanishes on white; neutral tints start one step darker.
    ...tint('gray', gray, 1),
  },
  dark: {},
})
