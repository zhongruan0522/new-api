/*
Copyright (C) 2023-2026 QuantumNous

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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { UsageLog } from '../data/schema'
import type { LogOtherData } from '../types'
import {
  buildTokenBreakdownGroups,
  buildTokenTooltipRows,
  getPriceSnapshotComponentLabelKey,
  getPriceSnapshotComponentQuantity,
  formatPriceSnapshotUnitPrice,
  parseBillingDetails,
  resolveDisplayTokens,
} from './billing-details'

function createLog(overrides: Partial<UsageLog> = {}): UsageLog {
  return {
    id: 1,
    user_id: 1,
    created_at: 0,
    type: 2,
    content: '',
    username: '',
    token_name: '',
    model_name: 'test-model',
    quota: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    use_time: 0,
    is_stream: false,
    channel: 0,
    channel_name: '',
    token_id: 0,
    group: '',
    ip: '',
    ua: '',
    x_title: '',
    http_referer: '',
    other: '',
    billing_details: null,
    request_id: '',
    upstream_request_id: '',
    model_icon: '',
    ...overrides,
  }
}

describe('parseBillingDetails', () => {
  test('empty value is legacy', () => {
    assert.deepEqual(parseBillingDetails(null), { status: 'legacy' })
    assert.deepEqual(parseBillingDetails(''), { status: 'legacy' })
  })

  test('schema v1 keeps official token dimensions', () => {
    const raw =
      '{"schema_version":1,"tokens":{"input":{"text_input":60,"audio_input":20},"output":{"reasoning_output":3},"cache":{"read_cache":40,"write_cache":5,"write_cache_5m":5}}}'
    const parsed = parseBillingDetails(raw)

    assert.equal(parsed.status, 'valid')
    assert.ok(parsed.status === 'valid')
    assert.equal(parsed.tokens.text_input, 60)
    assert.equal(parsed.tokens.audio_input, 20)
    assert.equal(parsed.tokens.reasoning_output, 3)
    assert.equal(parsed.tokens.read_cache, 40)
    assert.equal(parsed.tokens.write_cache, 5)
    assert.equal(parsed.tokens.write_cache_5m, 5)
    assert.equal(parsed.tokens.write_cache_1h, undefined)
  })

  test('explicit official zero and every schema dimension survive parsing', () => {
    const raw =
      '{"schema_version":1,"tokens":{"input":{"text_input":0,"image_input":1,"audio_input":2,"video_input":3,"document_input":4},"output":{"text_output":5,"audio_output":6,"image_output":7,"reasoning_output":8,"accepted_prediction":9,"rejected_prediction":10},"cache":{"read_cache":11,"write_cache":16,"write_cache_5m":7,"write_cache_1h":9}}}'
    const parsed = parseBillingDetails(raw)

    assert.equal(parsed.status, 'valid')
    assert.ok(parsed.status === 'valid')
    assert.equal(parsed.tokens.text_input, 0)
    assert.equal(parsed.tokens.image_input, 1)
    assert.equal(parsed.tokens.video_input, 3)
    assert.equal(parsed.tokens.document_input, 4)
    assert.equal(parsed.tokens.accepted_prediction, 9)
    assert.equal(parsed.tokens.rejected_prediction, 10)
    assert.equal(parsed.tokens.write_cache_1h, 9)
  })

  test('null and omitted optional fields are equivalent and unallocated cache is preserved', () => {
    const omitted = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{},"output":{"text_output":3},"cache":{"write_cache":12,"write_cache_5m":7}}}'
    )
    const explicit = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":null},"output":{"text_output":3},"cache":{"read_cache":null,"write_cache":12,"write_cache_5m":7,"write_cache_1h":null}}}'
    )

    assert.deepEqual(omitted, explicit)
    assert.ok(omitted.status === 'valid')
    assert.ok(explicit.status === 'valid')
    assert.equal(explicit.tokens.text_input, undefined)
    assert.equal(explicit.tokens.read_cache, undefined)
    assert.equal(explicit.tokens.write_cache, 12)
    assert.equal(explicit.tokens.write_cache_5m, 7)
    assert.equal(explicit.tokens.write_cache_1h, undefined)
  })

  test('rejects malformed JSON, unknown version and unknown fields', () => {
    assert.deepEqual(parseBillingDetails('{bad'), {
      status: 'invalid',
      code: 'malformed_json',
      errorKey: 'usageLogs.errors.billingDetails.malformed_json',
    })
    assert.deepEqual(parseBillingDetails('{"schema_version":2,"tokens":{}}'), {
      status: 'invalid',
      code: 'unknown_version',
      errorKey: 'usageLogs.errors.billingDetails.unknown_version',
    })
    const parsed = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"input_tokens":10},"output":{},"cache":{}}}'
    )

    assert.equal(parsed.status, 'invalid')
    assert.ok(parsed.status === 'invalid')
    assert.equal(parsed.code, 'invalid_fields')
  })

  test('rejects unsafe integers', () => {
    const parsed = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":9007199254740993},"output":{},"cache":{}}}'
    )

    assert.equal(parsed.status, 'invalid')
    assert.ok(parsed.status === 'invalid')
    assert.equal(parsed.code, 'invalid_fields')
  })

  test('rejects negative or fractional token values', () => {
    for (const textInput of [-1, 1.5]) {
      const parsed = parseBillingDetails(
        JSON.stringify({
          schema_version: 1,
          tokens: { input: { text_input: textInput }, output: {}, cache: {} },
        })
      )

      assert.equal(parsed.status, 'invalid')
      assert.ok(parsed.status === 'invalid')
      assert.equal(parsed.code, 'invalid_fields')
    }
  })

  test('rejects split cache without total or exceeding total', () => {
    for (const cache of [
      { write_cache_5m: 5 },
      { write_cache: 4, write_cache_5m: 5 },
    ]) {
      const parsed = parseBillingDetails(
        JSON.stringify({
          schema_version: 1,
          tokens: { input: {}, output: {}, cache },
        })
      )

      assert.equal(parsed.status, 'invalid')
      assert.ok(parsed.status === 'invalid')
      assert.equal(parsed.code, 'invalid_cache_splits')
    }
  })
})

describe('resolveDisplayTokens', () => {
  test('valid details do not derive tokens from aggregate columns', () => {
    const log = createLog({ prompt_tokens: 999, completion_tokens: 888 })
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{"text_output":7,"reasoning_output":3},"cache":{"read_cache":4}}}'
    )
    const tokens = resolveDisplayTokens(log, billing, null)

    assert.equal(tokens.input, 12)
    assert.equal(tokens.output, 7)
    assert.equal(tokens.reasoningOutput, 3)
    assert.equal(tokens.cacheRead, 4)
    assert.equal(tokens.fromBillingDetails, true)
  })

  test('valid cache splits expose their unallocated remainder', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{},"output":{},"cache":{"write_cache":12,"write_cache_5m":7,"write_cache_1h":3}}}'
    )
    const tokens = resolveDisplayTokens(createLog(), billing, null)

    assert.equal(tokens.cacheWrite, 12)
    assert.equal(tokens.cacheWrite5m, 7)
    assert.equal(tokens.cacheWrite1h, 3)
    assert.equal(tokens.cacheWriteUnallocated, 2)
  })

  test('legacy prompt aggregate remains the fallback when Other is absent', () => {
    const tokens = resolveDisplayTokens(
      createLog({ prompt_tokens: 120, completion_tokens: 30 }),
      parseBillingDetails(null),
      null
    )

    assert.equal(tokens.input, 120)
    assert.equal(tokens.output, 30)
    assert.equal(tokens.fromBillingDetails, false)
  })

  test('legacy logs keep existing Other subtraction and split fallback', () => {
    const log = createLog({ prompt_tokens: 120, completion_tokens: 30 })
    const other: LogOtherData = {
      cache_tokens: 40,
      cache_creation_tokens: 20,
      cache_creation_tokens_5m: 20,
      audio_input_token_count: 10,
      audio_input_seperate_price: true,
    }
    const tokens = resolveDisplayTokens(log, parseBillingDetails(null), other)

    assert.equal(tokens.input, 50)
    assert.equal(tokens.output, 30)
    assert.equal(tokens.cacheRead, 40)
    assert.equal(tokens.cacheWrite, 20)
    assert.equal(tokens.audioInput, 10)
    assert.equal(tokens.fromBillingDetails, false)
  })

  test('invalid details never leak legacy aggregate fallback', () => {
    const tokens = resolveDisplayTokens(
      createLog({ prompt_tokens: 120, completion_tokens: 30 }),
      parseBillingDetails('{bad'),
      null
    )

    assert.equal(tokens.input, null)
    assert.equal(tokens.output, null)
    assert.equal(tokens.cacheRead, null)
    assert.equal(tokens.cacheWrite, null)
    assert.equal(tokens.fromBillingDetails, true)
    assert.equal(tokens.hasValues, false)
    assert.deepEqual(buildTokenTooltipRows(tokens), [])
  })
})

describe('buildTokenTooltipRows', () => {
  test('reuses resolved official dimensions, explicit zeros and unallocated cache', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":0,"image_input":2},"output":{"text_output":7,"reasoning_output":3},"cache":{"read_cache":11,"write_cache":12,"write_cache_5m":7,"write_cache_1h":3}}}'
    )
    const tokens = resolveDisplayTokens(createLog(), billing, null)
    const rows = buildTokenTooltipRows(tokens)

    assert.deepEqual(
      rows.filter((row) => row.labelKey === 'usageLogs.fields.inputTokens'),
      [{ labelKey: 'usageLogs.fields.inputTokens', value: 0 }]
    )
    assert.ok(
      rows.some(
        (row) =>
          row.labelKey === 'usageLogs.fields.imageInput' && row.value === 2
      )
    )
    assert.ok(
      rows.some(
        (row) =>
          row.labelKey === 'usageLogs.fields.reasoningOutput' && row.value === 3
      )
    )
    assert.ok(
      rows.some(
        (row) =>
          row.labelKey === 'usageLogs.fields.cacheCreationUnallocated' &&
          row.value === 2
      )
    )
  })

  test('legacy tooltips retain positive values and omit unavailable dimensions', () => {
    const tokens = resolveDisplayTokens(
      createLog({ prompt_tokens: 120, completion_tokens: 30 }),
      parseBillingDetails(null),
      null
    )
    const rows = buildTokenTooltipRows(tokens)

    assert.ok(
      rows.some(
        (row) =>
          row.labelKey === 'usageLogs.fields.inputTokens' && row.value === 120
      )
    )
    assert.ok(
      rows.some(
        (row) =>
          row.labelKey === 'usageLogs.fields.outputTokens' && row.value === 30
      )
    )
    assert.ok(
      !rows.some((row) => row.labelKey === 'usageLogs.fields.reasoningOutput')
    )
  })
})

describe('buildTokenBreakdownGroups', () => {
  test('separates official modalities, output audit splits and cache tiers', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":0,"image_input":2},"output":{"text_output":7,"reasoning_output":3,"rejected_prediction":1},"cache":{"read_cache":11,"write_cache":12,"write_cache_5m":7,"write_cache_1h":3}}}'
    )
    const groups = buildTokenBreakdownGroups(
      resolveDisplayTokens(createLog(), billing, null),
      { aggregatePromptTokens: 999, formatTokens: String }
    )

    const modality = groups.find(
      (group) => group.titleKey === 'usageLogs.fields.multimodalTokens'
    )
    const outputSplits = groups.find(
      (group) => group.titleKey === 'usageLogs.fields.outputSplitTokens'
    )
    assert.deepEqual(
      modality?.rows.map((row) => row.labelKey),
      ['usageLogs.fields.imageInput']
    )
    assert.deepEqual(outputSplits?.rows, [
      {
        labelKey: 'usageLogs.fields.reasoningOutput',
        value: '3',
      },
      {
        labelKey: 'usageLogs.fields.rejectedPrediction',
        value: '1',
      },
    ])
    assert.ok(
      groups
        .find((group) => group.titleKey === 'usageLogs.fields.cacheTokens')
        ?.rows.some(
          (row) =>
            row.labelKey === 'usageLogs.fields.cacheCreationUnallocated' &&
            row.value === '2'
        )
    )
  })

  test('legacy output uses aggregate tokens while optional Other splits stay explicit', () => {
    const groups = buildTokenBreakdownGroups(
      resolveDisplayTokens(
        createLog({ prompt_tokens: 120, completion_tokens: 30 }),
        parseBillingDetails(null),
        null
      ),
      { aggregatePromptTokens: 120, formatTokens: String }
    )
    const standard = groups.find(
      (group) => group.titleKey === 'usageLogs.fields.standardTokens'
    )

    assert.deepEqual(standard?.rows.slice(0, 2), [
      { labelKey: 'usageLogs.fields.inputTokens', value: '120' },
      { labelKey: 'usageLogs.fields.outputTokens', value: '30' },
    ])
  })
})

describe('price snapshot helpers', () => {
  test('map official snapshot components without deriving absent quantities', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{"reasoning_output":3},"cache":{"read_cache":4,"write_cache":5,"write_cache_5m":5}}}'
    )
    const tokens = resolveDisplayTokens(createLog(), billing, null)

    assert.equal(
      getPriceSnapshotComponentQuantity('text_input', tokens, String),
      '12'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('reasoning_output', tokens, String),
      '3'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('write_cache', tokens, String),
      '5'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('request', tokens, String),
      '1'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('image_output', tokens, String),
      '—'
    )
    assert.equal(
      getPriceSnapshotComponentLabelKey('read_cache'),
      'systemSettings.fields.cacheRead'
    )
    assert.equal(
      getPriceSnapshotComponentLabelKey('custom_component'),
      'usageLogs.fields.billingItem'
    )
  })

  test('map contract snapshot component aliases without recalculation', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{"text_output":7,"reasoning_output":3},"cache":{"read_cache":4,"write_cache":8,"write_cache_5m":5,"write_cache_1h":3}}}'
    )
    const tokens = resolveDisplayTokens(createLog(), billing, null)

    assert.equal(
      getPriceSnapshotComponentQuantity('input', tokens, String),
      '12'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('output', tokens, String),
      '7'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('cache_read', tokens, String),
      '4'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('cache_write_5m', tokens, String),
      '5'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('cache_write_1h', tokens, String),
      '3'
    )
    assert.equal(
      getPriceSnapshotComponentLabelKey('cache_read'),
      'systemSettings.fields.cacheRead'
    )
    assert.equal(
      getPriceSnapshotComponentLabelKey('cache_write_5m'),
      'usageLogs.fields.cacheCreation5m'
    )
  })

  test('prefer saved settlement quantities over display token projection', () => {
    const billing = parseBillingDetails(
      '{"schema_version":1,"tokens":{"input":{"text_input":12},"output":{"reasoning_output":3},"cache":{"write_cache":12,"write_cache_5m":7,"write_cache_1h":3}}}'
    )
    const tokens = resolveDisplayTokens(createLog(), billing, null)

    assert.equal(
      getPriceSnapshotComponentQuantity('input', tokens, String, 988),
      '988'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('cache_write_5m', tokens, String, 9),
      '9'
    )
    assert.equal(
      getPriceSnapshotComponentQuantity('reasoning_output', tokens, String, -1),
      '—'
    )
  })

  test('display snapshot unit prices without currency conversion', () => {
    assert.equal(
      formatPriceSnapshotUnitPrice({
        unit_price: ' 4.2500 ',
        unit: 'per_1m_tokens',
        currency: 'EUR',
      }),
      '4.2500/M EUR'
    )
    assert.equal(
      formatPriceSnapshotUnitPrice({
        unit_price: '1.50',
        unit: 'per_request',
        currency: 'JPY',
      }),
      '1.50 JPY'
    )
    assert.equal(formatPriceSnapshotUnitPrice({ unit_price: ' ' }), null)
    assert.equal(
      formatPriceSnapshotUnitPrice({ unit_price: '4.25', currency: 'USD' }),
      '4.25 USD'
    )
    assert.equal(
      formatPriceSnapshotUnitPrice({
        unit_price: '4.25',
        unit: 'unsupported',
        currency: 'USD',
      }),
      '4.25 USD'
    )
  })
})
