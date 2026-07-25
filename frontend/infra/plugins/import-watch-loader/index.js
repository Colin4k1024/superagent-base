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

const defaultRuleOwner = 'wangfocheng';
const rules = [
  {
    regexp: /@tailwind utilities/,
    message: '引入了多余的 @tailwind utilities,请删除',
    owner: defaultRuleOwner,
  },
  {
    regexp: /@ies\/starling_intl/,
    message: '请使用@coze-arch/i18n代替直接引入@ies/starling_intl',
    owner: defaultRuleOwner,
  },
  {
    regexp: /\@coze-arch\/bot-env(?:['"]|(?:\/(?!runtime).*)?$)/,
    message:
      '请勿在web中引入@coze-arch/bot-env。GLOBAL_ENV已注入到页面中,直接使用变量即可(例: GLOBAL_ENVS.IS_BOE❌ IS_BOE✅)',
  },
];

module.exports = function (code, map) {
  try {
    rules.forEach(rule => {
      if (rule.regexp.test(code)) {
        throw Error(
          `${this.resourcePath}:${rule.message}。如有疑问请找${
            rule.owner || defaultRuleOwner
          }`,
        );
      }
    });
    this.callback(null, code, map);
  } catch (err) {
    this.callback(err, code, map);
    throw Error(err);
  }
};
