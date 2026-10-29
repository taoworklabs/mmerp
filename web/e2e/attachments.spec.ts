import { expect, test } from '@playwright/test'
import { created, createUser, employee, signedInApi, runId } from './api'
import { as, fixture } from './helpers'

// One run: department Q; pay (HR with the payroll role) handles contracts, hr (no salaries)
// sees them without their attachments.
const run = runId()
const login = { pay: `atpay_${run}`, hr: `athr_${run}` }
let ids: { contract: number }

const scan = '%PDF-1.4\n% Hop dong lao dong da ky - ban scan mau cho e2e\n%%EOF\n'

test.beforeAll(async ({ baseURL }) => {
  const api = await signedInApi(baseURL!)
  const company = await created(api.post('/api/org-units', { data: { parent_id: null, kind: 'company', name: `Công ty AT ${run}` } }))
  const q = await created(api.post('/api/org-units', { data: { parent_id: company, kind: 'department', name: `Phòng Q ${run}` } }))
  await createUser(api, login.pay, [
    { role: 'hrm.hr', unit: q },
    { role: 'hrm.payroll', unit: q },
  ])
  await createUser(api, login.hr, [{ role: 'hrm.hr', unit: q }])
  const e = await created(api.post('/api/hrm/employees', { data: employee(`ATE${run}`, q) }))
  const kind = await created(api.post('/api/hrm/contract-types', { data: { name: `Không xác định thời hạn ${run}`, fixed_term: false, active: true } }))
  const contract = await created(
    api.post('/api/hrm/contracts', {
      data: { employee_id: e, contract_type_id: kind, start_date: '2026-01-01', end_date: null, terms: { salary: 9_000_000, lines: [] } },
    }),
  )
  ids = { contract }
})

test.describe('attachments', () => {
  test('a signed contract is attached and downloads unchanged; without salary rights it stays hidden', async ({ browser }) => {
    const pay = await as(browser, login.pay, `/hrm/contracts/${ids.contract}`)
    const chooser = pay.waitForEvent('filechooser')
    // The empty list is itself where files go.
    await pay.getByRole('button', { name: /Kéo thả file/ }).click()
    await (await chooser).setFiles(fixture('contract-scan.pdf'))
    await expect(pay.getByText('Đính kèm (1)')).toBeVisible()

    const link = pay.getByRole('link', { name: 'contract-scan.pdf' })
    const download = pay.waitForEvent('download')
    await link.click()
    expect((await download).suggestedFilename()).toBe('contract-scan.pdf')
    const again = await pay.request.get((await link.getAttribute('href'))!)
    expect(await again.text()).toBe(scan)

    const hr = await as(browser, login.hr, `/hrm/contracts/${ids.contract}`)
    await expect(hr.getByText('Không có quyền xem đính kèm của bản ghi này.')).toBeVisible()
    await expect(hr.getByRole('link', { name: 'contract-scan.pdf' })).toHaveCount(0)
  })
})
