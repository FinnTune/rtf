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
    rules: {
      // eslint-plugin-react-hooks v7's "recommended" set is oriented around
      // the React Compiler (not used in this project — no babel-plugin-
      // react-compiler is installed). set-state-in-effect flags React's own
      // documented "fetch data on mount/when deps change" pattern
      // (https://react.dev/learn/you-might-not-need-an-effect#fetching-data)
      // as an error, which is exactly the shape of every data-fetching hook
      // in this app (usePaginatedPosts, and more to come) given the
      // deliberate choice not to add a data-fetching library. The other
      // rules in this set (rules-of-hooks, exhaustive-deps, refs,
      // immutability, ...) still catch real bugs and stay enabled.
      'react-hooks/set-state-in-effect': 'off',
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
