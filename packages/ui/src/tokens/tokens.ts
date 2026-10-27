// Every colour, size and spacing of the app; gen-tokens turns these into the --mmerp-* CSS variables.
// Ramps run from palest (0) to darkest (9). Index 6 is the shade filled components use,
// 7 the text on a tinted background, 0 the tint itself.
const primary = ['#E9F2FE', '#CFE1FD', '#8FB8F6', '#669DF1', '#4688EC', '#357DE8', '#1868DB', '#1558BC', '#123263', '#1C2B42'] as const
const success = ['#EFFFD6', '#D3F1A7', '#B3DF72', '#94C748', '#82B536', '#6A9A23', '#5B7F24', '#4C6B1F', '#37471F', '#28311B'] as const
const warning = ['#FFF5DB', '#FCE4A6', '#FBC828', '#FCA700', '#F68909', '#E06C00', '#BD5B00', '#9E4C00', '#693200', '#3A2C1F'] as const
const danger = ['#FFECEB', '#FFD5D2', '#FD9891', '#F87168', '#F15B50', '#E2483D', '#C9372C', '#AE2E24', '#5D1F1A', '#42221F'] as const
const info = primary
// Discovery marks what is new or in beta, nothing else.
const discovery = ['#F8EEFE', '#EED7FC', '#D8A0F7', '#C97CF4', '#BF63F3', '#AF59E1', '#964AC0', '#803FA5', '#48245D', '#35243F'] as const
// The one neutral ramp; its first two steps double as borders, so tints start one step darker.
const gray = ['#F8F8F8', '#F0F1F2', '#DDDEE1', '#DDDEE1', '#B4B7BD', '#7D818A', '#6B6E76', '#505258', '#3B3D42', '#292A2E'] as const

export type ColorName = 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'discovery' | 'gray'

export const tokens = {
  color: {
    ramp: { primary, success, warning, danger, info, discovery, gray },
    // Offset of the tinted ("light") variant within its ramp; gray's palest step vanishes on white.
    tint: { primary: 0, success: 0, warning: 0, danger: 0, info: 0, discovery: 0, gray: 1 },
    semantic: {
      text: gray[9],
      dimmed: gray[7],
      placeholder: gray[6],
      disabled: gray[1],
      border: '#0B120E24',
      body: '#FFFFFF',
      error: danger[7],
      'elevation-surface': '#FFFFFF',
      'elevation-surface-sunken': gray[0],
      'table-header': '#F4F5F6',
      'input-border': '#0B120E33',
      'overlay-backdrop': '#1E1F2166',
    },
    // Decorative: picked for distinctness in avatars/tags, never meaning. Each pair is tuned for AA text contrast.
    decorative: {
      blue: { bg: '#E7F5FF', text: '#1864AB' },
      teal: { bg: '#E6FCF5', text: '#087F5B' },
      violet: { bg: '#F3F0FF', text: '#5F3DC4' },
      // Darkened from #D9480F (3.95:1, fails AA at Avatar's 12px text): #BF3F0D clears 4.5:1.
      orange: { bg: '#FFF4E6', text: '#BF3F0D' },
      pink: { bg: '#FFF0F6', text: '#A61E4D' },
      cyan: { bg: '#E3FAFC', text: '#0B7285' },
      grape: { bg: '#F8F0FC', text: '#862E9C' },
      indigo: { bg: '#EDF2FF', text: '#364FC7' },
    },
  },
  fontFamily: "'Inter Variable', system-ui, sans-serif",
  fontWeight: { regular: 400, medium: 500, semibold: 600, heading: 650 },
  // Lines sit on a 4px grid: 12/16, 14/20, 16/24, 20/24.
  fontSize: { xs: 12, sm: 14, md: 14, lg: 16, xl: 20 },
  lineHeight: { xs: 1.34, sm: 1.43, md: 1.43, lg: 1.5, xl: 1.2 },
  heading: {
    h1: { size: 24, lineHeight: 1.17 },
    h2: { size: 16, lineHeight: 1.25 },
    h3: { size: 14, lineHeight: 1.43 },
  },
  space: { xxxs: 2, xxs: 4, xs: 8, sm: 12, md: 16, lg: 24, xl: 32, xxl: 48 },
  // Radius follows role: tags 4; controls, frames and modals 6; menus 8.
  radius: { xs: 2, sm: 4, md: 6, lg: 8, xl: 12 },
  // Only overlays cast a shadow, and a footer while content scrolls under it.
  shadow: {
    xs: '0 1px 1px #1E1F2140, 0 0 1px #1E1F214F',
    sm: '0 1px 1px #1E1F2140, 0 0 1px #1E1F214F',
    md: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
    lg: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
    xl: '0 8px 12px #1E1F2126, 0 0 1px #1E1F214F',
    footer: '0 -6px 8px -6px #1E1F2126',
  },
  controlHeight: { sm: 30, lg: 40 },
  layoutWidth: { content: 1200, side: 360, aside: 480, detail: 1440 },
  filterWidth: { search: 280, searchMin: 160, chip: 320, menu: 240 },
  motionDuration: 150,
  touchTarget: 44,
  // Dialogs and drawers sit above the app shell and a page's sticky bars; floating lists, menus and
  // tooltips sit above any dialog or drawer they open from.
  zIndexModal: 200,
  zIndexPopup: 300,
} as const

// md is where navigation becomes a drawer (1024px), lg where it stops auto-collapsing (1280px).
export const breakpoint = { xs: '36em', sm: '48em', md: '64em', lg: '80em', xl: '96em' } as const

// Icon sizes: 16px in text, tables and xs buttons, 18px in sm buttons, 24px in empty states.
export const icon = {
  text: { size: 16, stroke: 1.5 },
  button: { size: 18, stroke: 1.5 },
  empty: { size: 24, stroke: 1.5 },
} as const

export type Space = keyof typeof tokens.space
export type Radius = keyof typeof tokens.radius
export type FontSize = keyof typeof tokens.fontSize
export type DecorativeColor = keyof typeof tokens.color.decorative
