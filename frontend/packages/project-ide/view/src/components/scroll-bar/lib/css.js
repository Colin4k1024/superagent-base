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

export function get(element) {
  return getComputedStyle(element);
}

export function set(element, obj) {
  for (const key in obj) {
    let val = obj[key];
    if (typeof val === 'number') {
      val = `${val}px`;
    }
    element.style[key] = val;
  }
  return element;
}
