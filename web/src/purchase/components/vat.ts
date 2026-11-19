import { useTranslation } from 'react-i18next'
import type { VatRate } from '@/shared/api/purchase'

const rates: VatRate[] = ['10', '8', '5', '0', 'none']

// useVatOptions names the VAT rates a line or item may carry.
export function useVatOptions() {
  const { t } = useTranslation()
  const label = (r: string) => (r === 'none' ? t('purchase.vat.none') : t('purchase.vat.rate', { rate: r }))
  return { label, options: rates.map((r) => ({ value: r, label: label(r) })) }
}
