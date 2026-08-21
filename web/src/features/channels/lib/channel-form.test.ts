import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  transformChannelToFormDefaults,
  transformFormDataToUpdatePayload,
} from './channel-form'

describe('transformFormDataToUpdatePayload', () => {
  test('sends an explicit empty remark when a user clears channel remark', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'Claude primary',
        models: 'claude-3-5-sonnet',
        group: ['default'],
        remark: '',
      },
      42
    )

    assert.equal(payload.id, 42)
    assert.equal(payload.remark, '')
  })

  test('sends Claude cache_control and speed passthrough settings when enabled', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        type: 14,
        name: 'Claude primary',
        models: 'claude-sonnet-4-5',
        group: ['default'],
        allow_cache_control: true,
        allow_speed: true,
      },
      42
    )

    const settings = JSON.parse(payload.settings || '{}')
    assert.equal(settings.allow_cache_control, true)
    assert.equal(settings.allow_speed, true)
  })

  test('carries multi_key_mode when updating a multi-key channel', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'Key pool',
        models: 'gpt-4o-mini',
        group: ['default'],
        multi_key_type: 'polling',
      },
      42,
      true
    )

    assert.equal(payload.multi_key_mode, 'polling')
  })

  test('keeps multi_key_mode default for multi-key channel without explicit choice', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'Key pool',
        models: 'gpt-4o-mini',
        group: ['default'],
      },
      42,
      true
    )

    assert.equal(payload.multi_key_mode, 'random')
  })

  test('omits multi_key_mode when updating a single-key channel', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        name: 'Solo key',
        models: 'gpt-4o-mini',
        group: ['default'],
        multi_key_type: 'polling',
      },
      42
    )

    assert.equal('multi_key_mode' in payload, false)
  })
})

describe('transformChannelToFormDefaults', () => {
  test('loads Claude passthrough settings for a user editing a channel', () => {
    const defaults = transformChannelToFormDefaults({
      id: 42,
      type: 14,
      key: '',
      test_model: '',
      status: 1,
      name: 'Claude primary',
      created_time: 0,
      test_time: 0,
      response_time: 0,
      other: '',
      balance: 0,
      balance_updated_time: 0,
      models: 'claude-sonnet-4-5',
      group: 'default',
      used_quota: 0,
      auto_ban: 1,
      remark: '',
      max_input_tokens: 0,
      channel_info: {
        is_multi_key: false,
        is_plan: false,
        plan_name: '',
        multi_key_size: 0,
        multi_key_polling_index: 0,
        multi_key_mode: 'random',
      },
      settings: JSON.stringify({
        allow_cache_control: true,
        allow_speed: true,
      }),
    } as Parameters<typeof transformChannelToFormDefaults>[0])

    assert.equal(defaults.allow_cache_control, true)
    assert.equal(defaults.allow_speed, true)
  })
})

describe('openrouter_routing settings roundtrip', () => {
  test('builds openrouter_routing with omitempty semantics for OpenRouter channels', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        type: 20,
        name: 'OpenRouter',
        models: 'openai/gpt-4o',
        group: ['default'],
        or_order: 'anthropic, google-vertex/us-east5',
        or_allow_fallbacks: 'false',
        or_data_collection: 'deny',
        or_quantizations: 'fp8, int4',
        or_sort: 'price',
        or_sort_partition: 'none',
        or_pref_min_throughput: '50',
        or_pref_min_throughput_percentile: 'p90',
        or_pref_max_latency: '2.5',
        or_max_price_prompt: '1',
        or_max_price_completion: '2',
      },
      42
    )

    const settings = JSON.parse(payload.settings || '{}')
    const routing = settings.openrouter_routing
    assert.deepEqual(routing.order, ['anthropic', 'google-vertex/us-east5'])
    assert.equal(routing.allow_fallbacks, false)
    assert.equal(routing.data_collection, 'deny')
    assert.deepEqual(routing.quantizations, ['fp8', 'int4'])
    assert.deepEqual(routing.sort, { by: 'price', partition: 'none' })
    assert.deepEqual(routing.preferred_min_throughput, { p90: 50 })
    assert.equal(routing.preferred_max_latency, 2.5)
    assert.deepEqual(routing.max_price, { prompt: 1, completion: 2 })
    // 未配置的三态与空列表必须省略
    assert.equal('require_parameters' in routing, false)
    assert.equal('only' in routing, false)
    assert.equal('ignore' in routing, false)
    assert.equal('zdr' in routing, false)
    assert.equal('enforce_distillable_text' in routing, false)
    assert.equal('request' in routing.max_price, false)
  })

  test('drops openrouter_routing entirely when nothing is configured', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        type: 20,
        name: 'OpenRouter',
        models: 'openai/gpt-4o',
        group: ['default'],
        or_order: '',
        or_data_collection: '',
      },
      42
    )

    const settings = JSON.parse(payload.settings || '{}')
    assert.equal('openrouter_routing' in settings, false)
  })

  test('strips openrouter_routing when the channel type changes away from OpenRouter', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        type: 1,
        name: 'OpenAI',
        models: 'gpt-4o',
        group: ['default'],
        settings: JSON.stringify({
          openrouter_routing: { data_collection: 'deny' },
        }),
      },
      42
    )

    const settings = JSON.parse(payload.settings || '{}')
    assert.equal('openrouter_routing' in settings, false)
  })

  test('preserves unrelated keys while replacing openrouter_routing', () => {
    const payload = transformFormDataToUpdatePayload(
      {
        ...CHANNEL_FORM_DEFAULT_VALUES,
        type: 20,
        name: 'OpenRouter',
        models: 'openai/gpt-4o',
        group: ['default'],
        is_enterprise_account: true,
        settings: JSON.stringify({
          openrouter_routing: { order: ['old-provider'] },
          custom_future_key: 'keep-me',
        }),
        or_ignore: 'bad-provider',
      },
      42
    )

    const settings = JSON.parse(payload.settings || '{}')
    assert.equal(settings.openrouter_enterprise, true)
    assert.equal(settings.custom_future_key, 'keep-me')
    assert.deepEqual(settings.openrouter_routing, {
      ignore: ['bad-provider'],
    })
  })

  test('loads stored openrouter_routing back into form fields', () => {
    const defaults = transformChannelToFormDefaults({
      id: 42,
      type: 20,
      key: '',
      test_model: '',
      status: 1,
      name: 'OpenRouter',
      created_time: 0,
      test_time: 0,
      response_time: 0,
      other: '',
      balance: 0,
      balance_updated_time: 0,
      models: 'openai/gpt-4o',
      group: 'default',
      used_quota: 0,
      auto_ban: 1,
      remark: '',
      max_input_tokens: 0,
      channel_info: {
        is_multi_key: false,
        is_plan: false,
        plan_name: '',
        multi_key_size: 0,
        multi_key_polling_index: 0,
        multi_key_mode: 'random',
      },
      settings: JSON.stringify({
        openrouter_enterprise: true,
        openrouter_routing: {
          order: ['anthropic', 'deepinfra/turbo'],
          allow_fallbacks: false,
          require_parameters: true,
          data_collection: 'deny',
          quantizations: ['fp8'],
          sort: { by: 'throughput' },
          preferred_min_throughput: { p75: 40 },
          preferred_max_latency: 3,
          max_price: { prompt: 1, image: 0.1 },
        },
      }),
    } as Parameters<typeof transformChannelToFormDefaults>[0])

    assert.equal(defaults.or_order, 'anthropic, deepinfra/turbo')
    assert.equal(defaults.or_allow_fallbacks, 'false')
    assert.equal(defaults.or_require_parameters, 'true')
    assert.equal(defaults.or_data_collection, 'deny')
    assert.equal(defaults.or_quantizations, 'fp8')
    assert.equal(defaults.or_sort, 'throughput')
    assert.equal(defaults.or_sort_partition, '')
    assert.equal(defaults.or_pref_min_throughput, '40')
    assert.equal(defaults.or_pref_min_throughput_percentile, 'p75')
    assert.equal(defaults.or_pref_max_latency, '3')
    assert.equal(defaults.or_pref_max_latency_percentile, '')
    assert.equal(defaults.or_max_price_prompt, '1')
    assert.equal(defaults.or_max_price_image, '0.1')
    assert.equal(defaults.or_max_price_completion, '')
  })
})
