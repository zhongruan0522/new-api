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
import { buildSearchParams } from './filter'

// 回归 #310：筛选必须以"补丁"形式合并回 URL search，否则会丢掉 pageSize。
const urlWithPageSize = {
  page: 3,
  pageSize: 100,
}

describe('buildSearchParams', () => {
  test('common: 清空筛选后每个筛选键都显式存在（值为 undefined），可覆盖旧值', () => {
    const patch = buildSearchParams({}, 'common')

    for (const key of [
      'startTime',
      'endTime',
      'channel',
      'model',
      'token',
      'group',
      'username',
      'requestId',
      'upstreamRequestId',
      'ip',
      'ua',
      'xTitle',
      'httpReferer',
    ]) {
      assert.ok(
        key in patch,
        `key "${key}" must exist in patch so spreading over prev clears it`
      )
      assert.equal(patch[key], undefined)
    }
  })

  test('common: 合并回旧 search 时保留分页状态并清掉被清空的筛选', () => {
    const prev = {
      ...urlWithPageSize,
      model: 'gpt-4o',
      token: 'sk-x',
    }
    const next: Record<string, unknown> = {
      ...prev,
      ...buildSearchParams({ startTime: new Date(0) }, 'common'),
    }

    assert.equal(next.pageSize, 100)
    assert.equal(next.startTime, 0)
    assert.equal(next.model, undefined)
    assert.equal(next.token, undefined)
  })

  test('task: taskId 映射到 filter 键，空值时显式清除', () => {
    const filled = buildSearchParams(
      { startTime: new Date(1), endTime: new Date(2), taskId: 't-1' },
      'task'
    )
    assert.equal(filled.filter, 't-1')
    assert.equal(filled.channel, undefined)

    const cleared: Record<string, unknown> = {
      ...urlWithPageSize,
      ...buildSearchParams({}, 'task'),
    }
    assert.equal(cleared.pageSize, 100)
    assert.equal(cleared.filter, undefined)
    assert.equal(cleared.channel, undefined)
  })

  test('drawing: mjId 映射到 filter 键，空值时显式清除', () => {
    const filled = buildSearchParams({ mjId: 'mj-9' }, 'drawing')
    assert.equal(filled.filter, 'mj-9')

    const cleared = buildSearchParams({}, 'drawing')
    assert.equal(cleared.filter, undefined)
  })
})
