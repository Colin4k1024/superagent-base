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
  overrides: [
    {
      files: ['**/*.{ts,tsx}'],
      rules: {
        '@typescript-eslint/naming-convention': [
          'error',
          {
            selector: ['default', 'variableLike'],
            format: ['camelCase', 'UPPER_CASE', 'snake_case', 'PascalCase'],
          },
          {
            selector: ['class', 'interface', 'typeLike'],
            format: ['PascalCase'],
          },
          {
            selector: ['variable'],
            format: ['UPPER_CASE', 'camelCase'],
            modifiers: ['global', 'exported'],
          },
          {
            selector: 'objectLiteralProperty',
            format: null,
          },
          {
            selector: 'enumMember',
            format: ['UPPER_CASE', 'PascalCase'],
          },
          {
            selector: 'typeProperty',
            format: ['camelCase', 'snake_case', 'PascalCase'],
          },
          {
            selector: 'function',
            format: ['camelCase'],
            leadingUnderscore: 'forbid',
            trailingUnderscore: 'forbid',
          },
          {
            selector: 'parameter',
            format: ['camelCase', 'snake_case', 'PascalCase'],
            leadingUnderscore: 'allow',
            trailingUnderscore: 'forbid',
          },
          {
            selector: 'variable',
            modifiers: ['destructured'],
            format: [
              'camelCase',
              'PascalCase',
              'snake_case',
              'strictCamelCase',
              'StrictPascalCase',
              'UPPER_CASE',
            ],
          },
          {
            selector: 'import',
            format: ['camelCase', 'PascalCase', 'UPPER_CASE'],
          },
        ],
      },
    },
  ],
});
