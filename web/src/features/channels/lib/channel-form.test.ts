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
