import type { TFunction } from 'i18next'

// sectionTitle is a side section's title, with how many items it holds once that is known.
export function sectionTitle(t: TFunction, key: string, count: number | undefined): string {
  return count ? t(`${key}_count`, { count }) : t(key)
}
