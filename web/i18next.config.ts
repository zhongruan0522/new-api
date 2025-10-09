import { defineConfig } from 'i18next-cli';

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