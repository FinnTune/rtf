import js from '@eslint/js'
import vitest from '@vitest/eslint-plugin'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import globals from 'globals'
import { defineConfig, globalIgnores } from 'eslint/config'
import tseslint from 'typescript-eslint'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat['recommended-latest'],
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 'latest',
      globals: globals.browser,
    },
  },
  {
    // Context files conventionally export both a Provider component and its
    // paired `useX()` hook (AuthContext, and later WebSocketContext/
    // ChatContext) — react-refresh's "only export components" rule exists
    // for hot-reload ergonomics, not correctness, and doesn't fit this
    // established pattern.
    files: ['src/contexts/**/*.tsx'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
  {
    files: ['**/*.test.{ts,tsx}'],
    plugins: { vitest },
    extends: [vitest.configs.recommended],
    languageOptions: {
      globals: globals.vitest,
    },
  },
])
