import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'
import prettier from 'eslint-config-prettier'

export default tseslint.config(
  { ignores: ['dist/**', 'node_modules/**'] },

  // Base JS rules
  js.configs.recommended,

  // TypeScript rules
  ...tseslint.configs.recommended,

  // Vue rules
  ...pluginVue.configs['flat/recommended'],

  // Prettier — disables ESLint rules that conflict with formatting
  prettier,

  {
    files: ['**/*.vue', '**/*.ts', '**/*.js'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2020,
      },
      parserOptions: {
        parser: tseslint.parser,
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      // gRPC/proto data is inherently dynamic — any is intentional
      '@typescript-eslint/no-explicit-any': 'off',
      // Standard Vue module augmentation pattern in env.d.ts
      '@typescript-eslint/no-empty-object-type': 'off',
    },
  },
)
