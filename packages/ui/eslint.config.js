import js from '@eslint/js'
import i18next from 'eslint-plugin-i18next'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  { ignores: ['storybook-static'] },
  js.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: { globals: globals.browser },
  },
  {
    files: ['scripts/**', '.storybook/**', '*.cjs'],
    languageOptions: { globals: globals.node },
  },
  {
    files: ['src/**/*.tsx'],
    ignores: ['src/**/*.stories.tsx', 'src/**/*.test.tsx'],
    ...i18next.configs['flat/recommended'],
  },
  {
    files: ['src/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            { group: ['i18next', 'react-i18next'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['react-router'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['react-hook-form'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['@tanstack/*'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['openapi-fetch'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['dayjs'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['@mantine/*'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
            { group: ['**/web/**'], message: '@mmerp/ui knows nothing of the app; take it as a prop.' },
          ],
        },
      ],
    },
  },
)
