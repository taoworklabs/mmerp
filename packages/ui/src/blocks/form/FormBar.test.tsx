import { render } from 'vitest-browser-react'
import { describe, expect, it, vi } from 'vitest'
import { FormBar } from './FormBar.tsx'
import '../../styles/index.css'

describe('FormBar', () => {
  it('keeps exactly one live IntersectionObserver across re-renders', async () => {
    const disconnect = vi.spyOn(IntersectionObserver.prototype, 'disconnect')
    let constructed = 0
    const Original = IntersectionObserver
    vi.stubGlobal(
      'IntersectionObserver',
      class extends Original {
        constructor(...args: ConstructorParameters<typeof Original>) {
          super(...args)
          constructed++
        }
      },
    )

    const { rerender } = await render(
      <FormBar>
        <button type="button">Lưu</button>
      </FormBar>,
    )
    for (let i = 0; i < 5; i++) {
      await rerender(
        <FormBar data-rerender={i}>
          <button type="button">Lưu</button>
        </FormBar>,
      )
    }

    // Every re-attach disconnects the previous observer (or none attaches again at all), so
    // exactly one stays live after mounting and re-rendering several times.
    expect(constructed - disconnect.mock.calls.length).toBe(1)
    vi.unstubAllGlobals()
  })
})
