import type { Preview } from '@storybook/react-vite'
import '../src/styles/index.css'

const preview: Preview = {
  tags: ['autodocs'],
  parameters: {
    a11y: {
      test: 'error',
      config: {
        // Base UI's focus guards are aria-hidden yet tabbable on purpose: they catch Tab at a
        // popup's edges. axe flags them only when it runs while a popup is open, so at random.
        rules: [{ id: 'aria-hidden-focus', selector: '[aria-hidden="true"]:not([data-base-ui-focus-guard])' }],
      },
    },
    // Stories pick one with globals.viewport; addon-vitest sizes the test browser from it too.
    viewport: {
      options: {
        desktop: { name: 'Desktop', styles: { width: '1300px', height: '800px' }, type: 'desktop' },
        drawer: { name: 'Below drawer breakpoint', styles: { width: '800px', height: '600px' }, type: 'tablet' },
        mobile: { name: 'Mobile', styles: { width: '375px', height: '740px' }, type: 'mobile' },
        short: { name: 'Short window', styles: { width: '900px', height: '500px' }, type: 'desktop' },
      },
    },
  },
}

export default preview
