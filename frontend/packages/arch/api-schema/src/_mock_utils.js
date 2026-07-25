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

function rawParse(str) {
    const lines = (str || '').split('\n');
    const entries = lines.map((line) => {
      line = line.trim();
      const res = line.match(/at (.+) \((.+)\)/)||[]
      return {
        beforeParse: line,
        callee: res[1]
      };
    });
    return entries.filter((x) => x.callee !== undefined);
  }

  function createStruct(fn) {
    const structFactory = () => {
      const error = new Error();
      const items = rawParse(error.stack).filter((i) => i.callee === 'structFactory').map((i) => i.beforeParse);
      const isCircle = items.length > Array.from(new Set(items)).length;
      if (isCircle) {
        return {};
      }
      const res = fn();

      return res;
    };

    return structFactory;
  }
  module.exports={ createStruct }
  