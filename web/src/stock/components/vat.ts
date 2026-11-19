import { useTranslation } from 'react-i18next'
import type { Item } from '@/shared/api/core'

type VatRate = Item['vat_rate']

const rates: VatRate[] = ['10', '8', '5', '0', 'none']

// useVatOptions names the VAT rates a line or item may carry.
export function useVatOptions() {
  const { t } = useTranslation()
  const label = (r: string) => (r === 'none' ? t('stock.vat.none') : t('stock.vat.rate', { rate: r }))
  return { label, options: rates.map((r) => ({ value: r, label: label(r) })) }
}
