import i18n, { type TFunction } from 'i18next'
import { initReactI18next } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import en from './locales/en.json'
import vi from './locales/vi.json'

export const locales = ['vi', 'en'] as const
export type Locale = (typeof locales)[number]

const storageKey = 'mmerp.locale'

function isLocale(v: string | null | undefined): v is Locale {
  return locales.includes(v as Locale)
}

// Provisional locale until /me says otherwise: last used, then the browser, then vi.
export function pickLocale(stored: string | null, browserLanguage: string): Locale {
  if (isLocale(stored)) return stored
  const browser = browserLanguage.slice(0, 2)
  return isLocale(browser) ? browser : 'vi'
}

export async function initI18n(locale = pickLocale(localStorage.getItem(storageKey), navigator.language)) {
  document.documentElement.lang = locale
  // Keys are flat strings shared with the backend style ("hrm.nav.employees"), so no key separator.
  await i18n.use(initReactI18next).init({
    lng: locale,
    fallbackLng: 'vi',
    keySeparator: false,
    nsSeparator: false,
    interpolation: { escapeValue: false },
    resources: { vi: { translation: vi }, en: { translation: en } },
  })
  return locale
}

// switchLocale moves the page to the user's language and remembers it for the next start.
export async function switchLocale(locale: Locale) {
  rememberLocale(locale)
  document.documentElement.lang = locale
  await i18n.changeLanguage(locale)
}

export function rememberLocale(locale: Locale) {
  localStorage.setItem(storageKey, locale)
}

export async function addTranslations(locale: Locale, bundles: Array<() => Promise<{ default: Record<string, string> }>>) {
  for (const b of await Promise.all(bundles.map((load) => load()))) {
    i18n.addResourceBundle(locale, 'translation', b.default, true, true)
  }
}

// Prefixes that own error messages besides shared: an area's own, so a product's wording
// lives with the product. Error codes are unique across modules, so the order is irrelevant.
let errorPrefixes: string[] = []

// setErrorPrefixes is called once at startup with the areas in play. shared learns no
// product name of its own: it is given the list by whoever owns the areas.
export function setErrorPrefixes(prefixes: string[]) {
  errorPrefixes = prefixes
}

// errorText translates an API error code; unknown codes fall back to a generic message.
export function errorText(t: TFunction, err: unknown): string {
  const code = err instanceof ApiError ? err.code : 'network_error'
  const keys = [...errorPrefixes.map((p) => `${p}.error.${code}`), `shared.error.${code}`]
  return t(keys, { ...(err instanceof ApiError ? err.params : {}), defaultValue: t('shared.error.unexpected_error') })
}

const intlLocale = (locale: string) => (locale === 'en' ? 'en-GB' : 'vi-VN')

// formatFileSize shows a file's size the way people read it: 840 KB, 2,4 MB.
export function formatFileSize(bytes: number): string {
  const n = (v: number, digits: number) => new Intl.NumberFormat(intlLocale(i18n.language), { maximumFractionDigits: digits }).format(v)
  if (bytes < 1 << 20) return `${n(Math.max(1, Math.round(bytes / 1024)), 0)} KB`
  return `${n(bytes / (1 << 20), 1)} MB`
}

// formatDate shows a business date (YYYY-MM-DD) in the user's language: 31/03/2026.
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return ''
  return new Intl.DateTimeFormat(intlLocale(i18n.language), { timeZone: 'UTC', day: '2-digit', month: '2-digit', year: 'numeric' }).format(
    new Date(`${iso}T00:00:00Z`),
  )
}

// formatWeekday names a business date's day of the week in a few letters: T2 in vi, Mon in en.
export function formatWeekday(iso: string): string {
  const weekday = i18n.language === 'en' ? 'short' : 'narrow'
  return new Intl.DateTimeFormat(intlLocale(i18n.language), { timeZone: 'UTC', weekday }).format(new Date(`${iso}T00:00:00Z`))
}

// formatNumber groups digits the way the user's language does: 1.234.567 in vi.
export function formatNumber(n: number): string {
  return new Intl.NumberFormat(intlLocale(i18n.language)).format(n)
}

// formatMoney shows an amount in đồng on its own: 1.234.567 ₫ in vi, ₫1,234,567 in en.
export function formatMoney(n: number): string {
  return new Intl.NumberFormat(intlLocale(i18n.language), { style: 'currency', currency: 'VND', maximumFractionDigits: 0 }).format(n)
}

// formatMonth shows a month (YYYY-MM) in the user's language: 03/2026.
export function formatMonth(ym: string | null | undefined): string {
  if (!ym) return ''
  return new Intl.DateTimeFormat(intlLocale(i18n.language), { timeZone: 'UTC', month: '2-digit', year: 'numeric' }).format(new Date(`${ym}-01T00:00:00Z`))
}

// formatDateTime shows an instant in the tenant time zone: 31/03/2026 14:05.
export function formatDateTime(iso: string | null | undefined, timeZone: string): string {
  if (!iso) return ''
  const parts = new Intl.DateTimeFormat(intlLocale(i18n.language), {
    timeZone,
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date(iso))
  const p = Object.fromEntries(parts.map((x) => [x.type, x.value]))
  return `${p.day}/${p.month}/${p.year} ${p.hour}:${p.minute}`
}

// formatAgo shows how long ago an instant was while it is recent: 5 phút trước, 2 giờ trước;
// from a day on, the date in the tenant time zone.
export function formatAgo(iso: string, timeZone: string, now = new Date()): string {
  const minutes = Math.floor((now.getTime() - new Date(iso).getTime()) / 60000)
  const rtf = new Intl.RelativeTimeFormat(intlLocale(i18n.language), { numeric: 'auto' })
  if (minutes < 60) return rtf.format(-Math.max(1, minutes), 'minute')
  if (minutes < 24 * 60) return rtf.format(-Math.floor(minutes / 60), 'hour')
  return formatDateTime(iso, timeZone).slice(0, 10)
}

// formatDecimal shows a decimal string from the API (e.g. "2.5") the way the user's language writes numbers: 2,5.
export function formatDecimal(s: string | null | undefined, maxDigits = 2): string {
  if (s === null || s === undefined || s === '') return ''
  return new Intl.NumberFormat(intlLocale(i18n.language), { maximumFractionDigits: maxDigits }).format(Number(s))
}

// currentYear is this year in the tenant time zone, not the browser's.
export function currentYear(timeZone: string): number {
  return Number(new Intl.DateTimeFormat('en-GB', { timeZone, year: 'numeric' }).format(new Date()))
}
