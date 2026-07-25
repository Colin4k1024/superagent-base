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

import { describe, expect, it } from 'vitest';

import loader from '../index';
describe('test import-watch-loader', () => {
  it('code include tailwind utils', () => {
    const rawCode = `
      @tailwind utilities;
      body {
        width: 100%;
      }
    `;
    expect(() =>
      loader.call(
        {
          resourcePath: 'test1 resourcePath',
          callback: () => 0,
        },
        rawCode,
      ),
    ).toThrowError(
      'Error: test1 resourcePath:引入了多余的 @tailwind utilities,请删除。如有疑问请找wangfocheng',
    );
  });
  it('code not include tailwind utils', () => {
    const rawCode = `
        body {
          width: 100%;
        }
      `;
    const expectCode = rawCode;
    loader.call(
      {
        resourcePath: 'test2 resourcePath',
        callback: (error, code) => {
          expect(code).toBe(expectCode);
        },
      },
      rawCode,
    );
  });
});
