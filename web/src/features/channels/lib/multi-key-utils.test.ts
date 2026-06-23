import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { formatMultiKeyDisplayIndex } from './multi-key-utils'

describe('formatMultiKeyDisplayIndex', () => {
  test('shows one-based indexes to users', () => {
    assert.equal(formatMultiKeyDisplayIndex(0), '#1')
    assert.equal(formatMultiKeyDisplayIndex(1), '#2')
    assert.equal(formatMultiKeyDisplayIndex(9), '#10')
  })
})
