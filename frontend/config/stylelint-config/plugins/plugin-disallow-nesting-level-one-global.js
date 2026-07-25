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

const stylelint = require('stylelint');

const ruleName = 'plugin/disallow-first-level-global';

module.exports = stylelint.createPlugin(ruleName, function (ruleValue) {
  if (ruleValue === null || ruleValue === undefined || ruleValue === false) {
    return () => {
      // Nop.
    };
  }
  return function (postcssRoot, postcssResult) {
    postcssRoot.walkRules(rule => {
      if (rule.parent.type === 'root' && /:global/.test(rule.selector)) {
        stylelint.utils.report({
          ruleName,
          result: postcssResult,
          node: rule,
          message: 'Disallow :global class with nesting level of 1',
        });
      }
    });
  };
});

module.exports.ruleName = ruleName;
module.exports.messages = stylelint.utils.ruleMessages(ruleName, {
  expected: 'Disallow :global class with nesting level of 1',
});
