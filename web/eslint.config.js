import js from '@eslint/js'
import i18next from 'eslint-plugin-i18next'
import globals from 'globals'
import tseslint from 'typescript-eslint'

// Areas are every top-level folder of src except app and shared.
const areaFiles = ['src/*/**/*.{ts,tsx}']
const nonAreas = ['src/app/**', 'src/shared/**']

// @mantine/core names already replaced by @mmerp/ui; banned everywhere in src. Each migration task appends here.
export const mantineMigrated = [
  'Box', 'Stack', 'Group', 'Flex', 'Grid', 'SimpleGrid', 'Center', 'Divider', 'Paper', 'Text', 'Title', 'Anchor', 'Code', 'List',
  'Button', 'ActionIcon', 'UnstyledButton', 'Badge', 'ThemeIcon', 'Loader', 'Skeleton', 'VisuallyHidden',
  'TextInput', 'PasswordInput', 'Textarea', 'NumberInput', 'Checkbox', 'Select', 'MultiSelect', 'Combobox', 'useCombobox',
  'SegmentedControl', 'FileInput', 'FileButton', 'Input', 'InputWrapper',
]
const migratedPath = mantineMigrated.length
  ? [{ name: '@mantine/core', importNames: mantineMigrated, message: 'Import it from @mmerp/ui.' }]
  : []
// Only the kit touches Base UI.
const baseUi = { group: ['@base-ui/react', '@base-ui/react/*'], message: 'Import from @mmerp/ui; only the kit uses Base UI.' }

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
    files: ['src/**/*.{ts,tsx}'],
    rules: { 'no-restricted-imports': ['error', { paths: [...migratedPath, { name: '@mantine/hooks', message: 'Import the hook from @mmerp/ui.' }], patterns: [baseUi] }] },
  },
  {
    files: areaFiles,
    ignores: nonAreas,
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [baseUi],
          paths: [
            ...migratedPath,
            // Tooltip is the one overlay areas still take from Mantine.
            { name: '@mantine/core', allowImportNames: ['Tooltip'], message: 'Use the shared/ui component, or the @mmerp/ui primitive when it is on the areas\' list.' },
            { name: '@mantine/hooks', message: 'Import the hook from @mmerp/ui.' },
            {
              // Areas compose screens from shared/ui; only these layout and text primitives and look-only blocks come from the kit directly.
              name: '@mmerp/ui',
              allowImportNames: ['Anchor', 'Button', 'Divider', 'Group', 'List', 'Loader', 'Skeleton', 'Stack', 'Text', 'Tooltip', 'Panel', 'FieldList', 'KeyFacts', 'RecordColumns', 'ActionBar'],
              message:
                'Use the shared/ui component (DataTable, FormModal, Notice, MenuAction, QueryView, StatusBadge, the form fields…); add one there if none fits.',
            },
            { name: '@mantine/notifications', message: 'Use the shared/ui notification helpers.' },
            { name: '@mantine/dates', message: 'Use DateField from shared/ui.' },
          ],
        },
      ],
    },
  },
)
