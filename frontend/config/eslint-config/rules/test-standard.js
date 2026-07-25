/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

const globals = require('globals');

/** @type {(import('eslint').Linter.Config)[]} */
module.exports = [
  {
    files: [
      '**/*.{test,spec}.?(m|c){js,ts}?(x)',
      '**/{tests,__tests__,__test__}/**/*.?(m|c){js,ts}?(x)',
    ],
    languageOptions: {
      globals: {
        ...globals.jest,
        vi: 'readonly',
      },
    },
    rules: {
      'max-lines': 'off',
      'max-lines-per-function': 'off',
      'no-magic-numbers': 'off',
      'no-restricted-syntax': 'off',
      'import/no-named-as-default-member': 'off',
      '@coze-arch/max-line-per-function': 'off',
      '@coze-arch/no-deep-relative-import': 'off',
      '@typescript-eslint/consistent-type-assertions': 'off',
    },
  },
  {
    files: [
      '**/*.{test,spec}.?(m|c)ts?(x)',
      '**/{tests,__tests__,__test__}/**/*.?(m|c)ts?(x)',
    ],
    rules: {
      '@stylistic/ts/member-delimiter-style': [
        'warn',
        {
          multiline: {
            delimiter: 'semi',
            requireLast: true,
          },
          singleline: {
            delimiter: 'semi',
            requireLast: false,
          },
        },
      ],
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-magic-numbers': 'off',
      '@typescript-eslint/no-empty-function': 'off',
      '@coze-arch/no-batch-import-or-export': 'off',
    },
  },
];
