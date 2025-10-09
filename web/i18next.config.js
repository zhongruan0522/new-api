/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import { defineConfig } from 'i18next-cli';

/** @type {import('i18next-cli').I18nextToolkitConfig} */
export default defineConfig({
  locales: [
    "zh",
    "en",
    "fr"
  ],
  extract: {
    input: [
      "src/**/*.{js,jsx,ts,tsx}"
    ],
    ignore: [
      "src/i18n/**/*"
    ],
    output: "src/i18n/locales/{{language}}.json",
    ignoredAttributes: [
      "data-testid",
      "aria-label",
      "role",
      "className",
      "id",
      "key",
      "shape",
      "color",
      "size",
      "theme",
      "position",
      "layout",
      "margin",
      "trigger",
      "itemKey",
      "defaultActiveKey",
      "field",
      "value",
      "rel",
      "name",
      "validateStatus",
      "direction",
      "clipRule",
      "fillRule",
      "viewBox",
      "editorType",
      "autoComplete",
      "fill",
      "searchPosition",
      "uploadTrigger",
      "accept",
      "uploadTrigger",
      "placement",
      "rowKey",
      "style",
      "align",
      "crossOrigin",
      "field",
      "data-name",
      "data-index",
      "data-type",
      "height",
      "width",
      "overflow",
      "keyPrefix",
      "htmlType",
      "mode",
      "maxHeight",
      "hoverStyle",
      "selectedStyle"
    ],
    sort: true,
    disablePlurals: true,
    removeUnusedKeys: true,
    nsSeparator: false,
    keySeparator: false,
    mergeNamespaces: true
  }
});