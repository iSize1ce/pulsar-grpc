import js from '@eslint/js'
import globals from 'globals'

export default [
  { ignores: ['node_modules/**', 'frontend/**', 'dist/**', 'release/**', 'static/**'] },

  {
    files: ['electron/**/*.cjs'],
    ...js.configs.recommended,
    languageOptions: {
      globals: {
        ...globals.node,
        ...globals.es2020,
      },
      sourceType: 'commonjs',
    },
  },
]
