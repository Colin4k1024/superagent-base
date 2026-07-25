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

const cls = {
  main: 'ide-ps',
  rtl: 'ide-ps__rtl',
  element: {
    thumb: x => `ide-ps__thumb-${x}`,
    rail: x => `ide-ps__rail-${x}`,
    consuming: 'ide-ps__child--consume',
  },
  state: {
    focus: 'ide-ps--focus',
    clicking: 'ide-ps--clicking',
    active: x => `ide-ps--active-${x}`,
    scrolling: x => `ide-ps--scrolling-${x}`,
  },
};

export default cls;

/*
 * Helper methods
 */
const scrollingClassTimeout = { x: null, y: null };

export function addScrollingClass(i, x) {
  const classList = i.element.classList;
  const className = cls.state.scrolling(x);

  if (classList.contains(className)) {
    clearTimeout(scrollingClassTimeout[x]);
  } else {
    classList.add(className);
  }
}

export function removeScrollingClass(i, x) {
  scrollingClassTimeout[x] = setTimeout(
    () => i.isAlive && i.element.classList.remove(cls.state.scrolling(x)),
    i.settings.scrollingThreshold
  );
}

export function setScrollingClassInstantly(i, x) {
  addScrollingClass(i, x);
  removeScrollingClass(i, x);
}
