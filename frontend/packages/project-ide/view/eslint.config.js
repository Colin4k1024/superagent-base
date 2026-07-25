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

const { defineConfig } = require('@coze-arch/eslint-config');

module.exports = defineConfig({
  packageRoot: __dirname,
  preset: 'web',
  rules: {
    'no-restricted-syntax': 'off',
    '@typescript-eslint/naming-convention': 'off',
    '@typescript-eslint/no-magic-numbers': 'off',
    '@coze-arch/no-batch-import-or-export': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/no-non-null-assertion': 'off',
    'rule-empty-line-before': 'off',
    'alpha-value-notation': 'off',
    '@typescript-eslint/require-await': 'off',
    '@typescript-eslint/no-namespace': 'off',
    '@typescript-eslint/no-invalid-void-type': 'off',
    '@typescript-eslint/no-empty-interface': 'off',
    '@typescript-eslint/no-empty-function': 'off',
    'max-params': 'off',
    '@typescript-eslint/no-this-alias': 'off',
    '@typescript-eslint/consistent-type-assertions': 'off',
    '@typescript-eslint/no-useless-constructor': 'off',
    'react-hooks/rules-of-hooks': 'off',
    'default-case': 'off',
    'no-implicit-coercion': 'off',
    '@typescript-eslint/method-signature-style': 'off',
    '@typescript-eslint/no-shadow': 'off',
    '@typescript-eslint/method-signature-style': 'off',
    'no-empty': 'off',
    'max-lines': 'off',
    'no-cond-assign': 'off',
    '@typescript-eslint/ban-ts-comment': 'off',
    eqeqeq: 'off',
    '@coze-arch/max-line-per-function': 'off',
    '@coze-arch/no-empty-catch': 'off',
    '@typescript-eslint/unified-signatures': 'off',
    'no-unused-vars': 'off',
    'no-undef': 'off',
  },
});
