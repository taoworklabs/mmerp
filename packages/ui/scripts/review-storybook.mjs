/* global document, getComputedStyle */
import assert from 'node:assert/strict'
import { mkdir, writeFile } from 'node:fs/promises'
import { chromium } from 'playwright'

// Run against either the dev server or a locally served static Storybook build.
const base = process.env.STORYBOOK_URL ?? 'http://127.0.0.1:6006'
const output = new URL('../artifacts/storybook/', import.meta.url)
await mkdir(output, { recursive: true })
const browser = await chromium.launch()
const results = []
const overlayChecks = []
try {
  const index = await (await fetch(`${base}/index.json`)).json()
  const stories = Object.values(index.entries).filter(entry => entry.type === 'story')
  const cases = [
    'foundations-components--desktop',
    'screens-employees--list', 'screens-employees--record', 'screens-employees--new',
  ]
  for (const width of [1280, 1024, 375]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, reducedMotion: 'reduce' })
    page.setDefaultTimeout(10000)
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    for (const id of cases) {
      assert(stories.some(story => story.id === id), `Missing story: ${id}`)
      await page.goto(`${base}/iframe.html?id=${id}&viewMode=story`, { waitUntil: 'networkidle' })
      await page.locator('#storybook-root h1').waitFor()
      await page.evaluate(() => document.fonts.ready)
      assert.equal(await page.locator('.sb-errordisplay').isVisible(), false, `Story error: ${id}`)
      const dimensions = await page.evaluate(() => ({ width: document.documentElement.clientWidth, content: document.documentElement.scrollWidth }))
      assert(dimensions.content <= dimensions.width, `Horizontal overflow: ${id} at ${width}px`)
      const screenshot = `${id}-${width}.png`
      await page.screenshot({ path: new URL(screenshot, output).pathname, fullPage: true })
      results.push({ id, width, screenshot, horizontalOverflow: false })
    }
    if (width >= 1024) {
      await page.goto(`${base}/iframe.html?id=blocks-datatable--overlay-scrollbars&viewMode=story`, { waitUntil: 'networkidle' })
      const area = page.locator('[data-scroll-area]')
      const viewport = area.locator('[data-scroll-area-viewport]')
      await area.hover()
      for (const orientation of ['horizontal', 'vertical']) {
        const thumb = area.locator(`[data-scroll-area-scrollbar][data-orientation="${orientation}"] [data-scroll-area-thumb]`)
        const rect = await thumb.boundingBox()
        assert(rect, `Missing ${orientation} thumb`)
        const x = rect.x + rect.width / 2
        const y = rect.y + rect.height / 2
        await page.mouse.move(x, y)
        await page.mouse.down()
        await page.mouse.move(x + (orientation === 'horizontal' ? 100 : 0), y + (orientation === 'vertical' ? 80 : 0), { steps: 10 })
        await page.mouse.up()
        await page.waitForFunction(axis => {
          const element = document.querySelector('[data-scroll-area-viewport]')
          return (axis === 'horizontal' ? element.scrollLeft : element.scrollTop) > 0
        }, orientation)
      }
      await page.screenshot({ path: new URL(`table-overlay-${width}.png`, output).pathname })
      await viewport.evaluate(element => { element.scrollTop = 0; element.scrollLeft = 0 })
      await viewport.hover()
      await page.mouse.wheel(100, 200)
      await page.waitForFunction(() => {
        const element = document.querySelector('[data-scroll-area-viewport]')
        return element.scrollTop > 0 && element.scrollLeft > 0
      })
      await viewport.focus()
      await page.keyboard.press('Control+End')
      await page.waitForFunction(() => {
        const element = document.querySelector('[data-scroll-area-viewport]')
        return element.scrollTop + element.clientHeight >= element.scrollHeight - 1
      })
      await viewport.evaluate(element => element.blur())
      await page.mouse.move(width - 1, 899)
      await page.waitForFunction(() => [...document.querySelectorAll('[data-scroll-area-scrollbar]')].every(element => getComputedStyle(element).opacity === '0'))
      overlayChecks.push({ width, dragBothAxes: true, wheelBothAxes: true, keyboard: true, autoHide: true })
    }
    // Exercise a real popup, its focus trap and focus return in the served Storybook.
    await page.goto(`${base}/iframe.html?id=overlay-dialog--default&viewMode=story`, { waitUntil: 'networkidle' })
    const trigger = page.getByRole('button', { name: 'Mở hộp thoại' })
    await trigger.click()
    const dialog = page.getByRole('dialog')
    await dialog.waitFor()
    assert(await dialog.evaluate(element => element.contains(document.activeElement)))
    for (let i = 0; i < 3; i++) await page.keyboard.press('Tab')
    assert(await dialog.evaluate(element => element.contains(document.activeElement)))
    if (width < 1024) {
      const bounds = await dialog.boundingBox()
      assert.equal(bounds.width, width)
      assert.equal(bounds.height, 900)
    }
    await page.screenshot({ path: new URL(`dialog-${width}.png`, output).pathname })
    await page.keyboard.press('Escape')
    await dialog.waitFor({ state: 'hidden' })
    assert(await trigger.evaluate(element => element === document.activeElement))
    assert.deepEqual(errors, [], `Browser errors at ${width}px`)
    await page.close()
  }
  await writeFile(new URL('report.json', output), JSON.stringify({ storyCount: stories.length, checked: results, overlayChecks, dialogKeyboardChecks: [1280, 1024, 375] }, null, 2))
  console.log(`Reviewed ${results.length} compositions, ${overlayChecks.length} overlay scrollbar flows and 3 dialog keyboard flows. Screenshots: ${output.pathname}`)
} finally {
  await browser.close()
}
