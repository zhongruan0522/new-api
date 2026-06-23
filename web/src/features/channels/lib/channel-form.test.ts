import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
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
})
