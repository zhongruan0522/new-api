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
import { loadVisibilityState } from './use-persistent-column-visibility.ts'

class FakeStorage {
  private store = new Map<string, string>()

  clear() {
    this.store.clear()
  }

  getItem(key: string) {
    return this.store.has(key) ? this.store.get(key)! : null
  }

  setItem(key: string, value: string) {
    this.store.set(key, value)
  }

  removeItem(key: string) {
    this.store.delete(key)
  }
}

function withFakeWindow(testFn: () => void) {
  const originalWindow = (globalThis as unknown as Record<string, unknown>)
    .window
  const fakeStorage = new FakeStorage()
  ;(globalThis as unknown as Record<string, unknown>).window = {
    localStorage: fakeStorage as unknown as Storage,
  } as Window
  try {
    testFn()
  } finally {
    ;(globalThis as unknown as Record<string, unknown>).window = originalWindow
  }
}

describe('loadVisibilityState', () => {
  test('returns defaults when no localStorage is available', () => {
    const defaults = { id: true, name: false }
    const result = loadVisibilityState('test-key', defaults)
    assert.deepEqual(result, defaults)
  })

  test('merges user overrides on top of defaults', () => {
    withFakeWindow(() => {
      const storageKey = 'nookmux:table:column-visibility:merge-key'
      const defaults = { id: true, name: false, createdAt: true }
      window.localStorage.setItem(
        storageKey,
        JSON.stringify({ name: true, createdAt: false })
      )
      const result = loadVisibilityState(storageKey, defaults)
      assert.equal(result.name, true)
      assert.equal(result.createdAt, false)
      assert.equal(result.id, true)
    })
  })

  test('ignores malformed persisted values and returns defaults', () => {
    withFakeWindow(() => {
      const storageKey = 'nookmux:table:column-visibility:bad-json-key'
      const defaults = { id: true }
      window.localStorage.setItem(storageKey, 'not-valid-json')
      const result = loadVisibilityState(storageKey, defaults)
      assert.deepEqual(result, defaults)
    })
  })

  test('ignores non-object persisted values and returns defaults', () => {
    withFakeWindow(() => {
      const storageKey = 'nookmux:table:column-visibility:array-key'
      const defaults = { id: true }
      window.localStorage.setItem(storageKey, JSON.stringify(['id']))
      const result = loadVisibilityState(storageKey, defaults)
      assert.deepEqual(result, defaults)
    })
  })
})
