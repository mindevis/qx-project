import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';

// Focused, high-signal config. `tsc` (noUnusedLocals/Parameters, strict) already
// covers unused vars/imports and type errors, so ESLint adds what tsc cannot:
// React Hooks correctness and a few real-bug rules.
export default tseslint.config(
  { ignores: ['dist', 'coverage', 'playwright-report', 'test-results'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      globals: { ...globals.browser, ...globals.node },
    },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      // Covered by tsc; avoid double-reporting.
      '@typescript-eslint/no-unused-vars': 'off',
    },
  },
);
