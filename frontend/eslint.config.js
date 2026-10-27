import js from '@eslint/js';
import parser from '@typescript-eslint/parser';
import ts from '@typescript-eslint/eslint-plugin';
import hooks from 'eslint-plugin-react-hooks';
import globals from 'globals';

export default [
  { ignores: ['dist/**', 'coverage/**', 'node_modules/**'] },
  { ...js.configs.recommended, files: ['**/*.js'], languageOptions: { globals: globals.node } },
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: { parser, parserOptions: { ecmaVersion: 'latest', sourceType: 'module', ecmaFeatures: { jsx: true } }, globals: { ...globals.browser, ...globals.node } },
    plugins: { '@typescript-eslint': ts, 'react-hooks': hooks },
    rules: {
      ...ts.configs.recommended.rules,
      ...hooks.configs.recommended.rules,
    },
  },
];
