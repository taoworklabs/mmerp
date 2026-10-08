import js from '@eslint/js'
import i18next from 'eslint-plugin-i18next'
import globals from 'globals'
import tseslint from 'typescript-eslint'

// Areas are every top-level folder of src except app and shared.
const areaFiles = ['src/*/**/*.{ts,tsx}']
const nonAreas = ['src/app/**', 'src/shared/**']

export default tseslint.config(
  { ignores: ['dist', 'lint-fixtures', 'src/shared/api/*.gen.ts'] },
  js.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: { globals: globals.browser },
  },
  {
    files: ['scripts/**', '*.cjs'],
    languageOptions: { globals: globals.node },
  },
  {
    files: ['src/**/*.tsx'],
    ...i18next.configs['flat/recommended'],
  },
  {
    files: ['src/**/*.tsx'],
    ignores: ['src/shared/ui/**'],
    rules: {
      'no-restricted-syntax': [
        'error',
        { selector: "JSXAttribute[name.name='style']", message: 'No inline style: use theme tokens and shared/ui components.' },
      ],
    },
  },
  {
    files: areaFiles,
    ignores: nonAreas,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@mantine/core',
              importNames: ['Table', 'Modal', 'Badge', 'Notification', 'NumberInput'],
              message: 'Use the shared/ui component (DataTable, FormModal, DocumentStatus, MoneyField, DecimalField…).',
            },
            { name: '@mantine/notifications', message: 'Use the shared/ui notification helpers.' },
            { name: '@mantine/dates', message: 'Use DateField from shared/ui.' },
          ],
        },
      ],
    },
  },
)
