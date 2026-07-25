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

const babelOptions = {
  plugins: [
    [
      require.resolve('@babel/plugin-proposal-decorators'),
      {
        legacy: true,
      },
    ],
    [
      require.resolve('@babel/plugin-proposal-class-properties'),
      {
        loose: true,
      },
    ],
    require.resolve('@babel/plugin-proposal-object-rest-spread'),
    require.resolve('@babel/plugin-proposal-optional-catch-binding'),
    require.resolve('@babel/plugin-proposal-async-generator-functions'),
    require.resolve('@babel/plugin-proposal-export-namespace-from'),
    require.resolve('@babel/plugin-proposal-export-default-from'),
    require.resolve('@babel/plugin-proposal-nullish-coalescing-operator'),
    require.resolve('@babel/plugin-proposal-optional-chaining'),
    [
      require.resolve('@babel/plugin-proposal-pipeline-operator'),
      {
        proposal: 'minimal',
      },
    ],
    require.resolve('@babel/plugin-proposal-do-expressions'),
    require.resolve('@babel/plugin-proposal-function-bind'),
    require.resolve('@babel/plugin-syntax-dynamic-import'),
    require.resolve('@babel/plugin-syntax-jsx'),
  ],
};

/** @type {(import('eslint').Linter.Config)[]} */
module.exports = [
  {
    files: ['**/*.?(m|c)js?(x)'],
    rules: {
      'no-import-assign': 'error',
      'no-extra-semi': 'error',
      'no-undef': 'error',
      'no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
        },
      ],
      'comma-dangle': ['error', 'always-multiline'],
      'no-array-constructor': 'error',
      'dot-notation': 'error',
      'constructor-super': 'error',
      'no-this-before-super': 'error',
      'no-useless-constructor': 'error',
      'getter-return': [
        'error',
        {
          allowImplicit: true,
        },
      ],
      'no-setter-return': 'error',
      'no-dupe-class-members': 'error',
      'default-param-last': 'error',
      'no-func-assign': 'error',
      'no-unsafe-negation': 'error',
      'valid-typeof': 'error',
      'no-empty-function': 'error',
      quotes: [
        'error',
        'single',
        {
          avoidEscape: true,
        },
      ],
      'no-loss-of-precision': 'error',
      'no-magic-numbers': [
        'warn',
        {
          ignoreArrayIndexes: true,
          ignoreDefaultValues: true,
          enforceConst: true,
          ignore: [0, 1, -1],
          ignoreClassFieldInitialValues: true,
        },
      ],
      'no-unreachable': 'error',
      'no-throw-literal': 'error',
      'no-implied-eval': 'error',
      'no-new-symbol': 'error',
      'no-obj-calls': 'error',
      camelcase: [
        'error',
        {
          properties: 'never',
          allow: ['^UNSAFE_'],
          ignoreDestructuring: true,
        },
      ],
      'unicorn/filename-case': [
        'error',
        {
          cases: {
            kebabCase: true,
          },
        },
      ],
      'max-lines-per-function': [
        'error',
        {
          max: 300,
          IIFEs: true,
        },
      ],
    },
    languageOptions: {
      parser: require('@babel/eslint-parser'),
      parserOptions: {
        ecmaVersion: 11,
        sourceType: 'module',
        ecmaFeatures: {
          jsx: true,
          legacyDecorators: true,
        },
        requireConfigFile: false,
        babelOptions,
      },
    },
  },
  {
    // for jsx
    files: ['**/*.?(m|c)jsx'],
    rules: {
      'react/jsx-uses-react': 'error',
      'react/jsx-uses-vars': 'error',
      'unicorn/filename-case': [
        'warn',
        {
          cases: {
            kebabCase: true,
            snakeCase: true,
            pascalCase: true,
          },
          ignore: ['^(?!.*?\\.jsx$)(?!.*?\\.tsx$).+'],
        },
      ],
      'max-lines-per-function': [
        'warn',
        {
          max: 300,
          IIFEs: true,
        },
      ],
      'max-lines': [
        'error',
        {
          max: 500,
          skipComments: true,
          skipBlankLines: true,
        },
      ],
    },
  },
];
