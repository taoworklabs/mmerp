import i18next from 'i18next'
import { afterEach, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/hrm'
import { errorText, setErrorPrefixes } from '@/shared/i18n'
import vi from './vi/meta.json'

afterEach(() => setErrorPrefixes([]))

// The wording names the day kind through another key; this checks the nesting really resolves.
it('names the day kind the calendar expects', async () => {
  setErrorPrefixes(['hrm'])
  const i18n = i18next.createInstance()
  await i18n.init({ lng: 'vi', resources: { vi: { translation: vi } }, interpolation: { escapeValue: false } })
  expect(errorText(i18n.t, new ApiError(422, 'overtime_day_kind_mismatch', { day_kind: 'holiday' }))).toBe(
    'Theo lịch làm việc của công ty, ngày này thuộc loại “Ngày lễ”. Hãy chọn đúng loại đó.',
  )
})
