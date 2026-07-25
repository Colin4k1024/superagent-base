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

import updateGeometry from '../update-geometry';

const clickRail = e => {
  const { element } = e;

  e.event.bind(e.scrollbarYRail, 'mousedown', e => {
    const positionTop =
      e.pageY -
      window.pageYOffset -
      e.scrollbarYRail.getBoundingClientRect().top;
    const direction = positionTop > e.scrollbarYTop ? 1 : -1;

    e.element.scrollTop += direction * e.containerHeight;
    updateGeometry(i);

    e.stopPropagation();
  });

  e.event.bind(e.scrollbarY, 'mousedown', e => e.stopPropagation());

  e.event.bind(e.scrollbarX, 'mousedown', e => e.stopPropagation());
  e.event.bind(e.scrollbarXRail, 'mousedown', e => {
    const left =
      e.pageX -
      window.pageXOffset -
      e.scrollbarXRail.getBoundingClientRect().left;
    const direction = left > e.scrollbarXLeft ? 1 : -1;

    e.element.scrollLeft += direction * e.containerWidth;
    updateGeometry(i);

    e.stopPropagation();
  });
};
export default clickRail;
