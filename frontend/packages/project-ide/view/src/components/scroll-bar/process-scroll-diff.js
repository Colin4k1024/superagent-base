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

import { setScrollingClassInstantly } from './lib/class-names';

function createEvent(name) {
  if (typeof window.CustomEvent === 'function') {
    return new CustomEvent(name);
  } else {
    const evt = document.createEvent('CustomEvent');
    evt.initCustomEvent(name, false, false, undefined);
    return evt;
  }
}

export default function (
  i,
  axis,
  diff,
  useScrollingClass = true,
  forceFireReachEvent = false,
) {
  let fields;
  if (axis === 'top') {
    fields = [
      'contentHeight',
      'containerHeight',
      'scrollTop',
      'y',
      'up',
      'down',
    ];
  } else if (axis === 'left') {
    fields = [
      'contentWidth',
      'containerWidth',
      'scrollLeft',
      'x',
      'left',
      'right',
    ];
  } else {
    throw new Error('A proper axis should be provided');
  }

  processScrollDiff(i, diff, fields, useScrollingClass, forceFireReachEvent);
}

function processScrollDiff(
  i,
  diff,
  [contentHeight, containerHeight, scrollTop, y, up, down],
  useScrollingClass = true,
  forceFireReachEvent = false,
) {
  const { element } = i;

  // reset reach
  i.reach[y] = null;

  // 1 for subpixel rounding
  if (element[scrollTop] < 1) {
    i.reach[y] = 'start';
  }

  // 1 for subpixel rounding
  if (element[scrollTop] > i[contentHeight] - i[containerHeight] - 1) {
    i.reach[y] = 'end';
  }

  if (diff) {
    element.dispatchEvent(createEvent(`ide-ps-scroll-${y}`));

    if (diff < 0) {
      element.dispatchEvent(createEvent(`ide-ps-scroll-${up}`));
    } else if (diff > 0) {
      element.dispatchEvent(createEvent(`ide-ps-scroll-${down}`));
    }

    if (useScrollingClass) {
      setScrollingClassInstantly(i, y);
    }
  }

  if (i.reach[y] && (diff || forceFireReachEvent)) {
    element.dispatchEvent(createEvent(`ide-ps-${y}-reach-${i.reach[y]}`));
  }
}
