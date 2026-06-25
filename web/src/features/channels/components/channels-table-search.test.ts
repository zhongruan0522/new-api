import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { shouldCommitDebouncedSearch } from './channels-table'

describe('shouldCommitDebouncedSearch', () => {
  test('does not submit search while a user is composing Chinese input', () => {
    const composing = shouldCommitDebouncedSearch('渠道', '渠道', '', true)

    assert.equal(composing, false)
  })

  test('submits only after composition ends and debounce catches the final text', () => {
    const beforeDebounce = shouldCommitDebouncedSearch('渠道', '渠', '', false)
    const afterDebounce = shouldCommitDebouncedSearch('渠道', '渠道', '', false)

    assert.equal(beforeDebounce, false)
    assert.equal(afterDebounce, true)
  })
})
