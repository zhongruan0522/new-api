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
import type { PingStatus } from '@/features/dashboard/types'

/**
 * Get color class for latency status
 */
export function getLatencyColorClass(latency: number): string {
  if (latency < 200) {
    return 'text-success'
  }
  if (latency < 500) {
    return 'text-warning'
  }
  return 'text-destructive'
}

/**
 * detectMixedContent 检测 HTTPS 页面请求 HTTP 资源的混合内容场景。
 * 浏览器会直接阻止这类请求并抛出 TypeError: Failed to fetch，
 * 无法在 catch 中精确区分，因此在前端调用 fetch 前提前识别。
 *
 * 仅依赖 location.protocol 与目标 URL 的 protocol 比较，
 * 相对路径 URL 会被解析为当前页面协议，不会误判。
 */
function detectMixedContent(url: string): boolean {
  if (typeof window === 'undefined') {
    return false
  }
  if (window.location.protocol !== 'https:') {
    return false
  }
  let targetProtocol: string
  try {
    targetProtocol = new URL(url, window.location.href).protocol
  } catch {
    return false
  }
  return targetProtocol === 'http:'
}

/**
 * Test URL latency
 */
export async function testUrlLatency(url: string): Promise<PingStatus> {
  // 混合内容场景浏览器会直接拦截，提前返回结构化原因，避免无意义的 fetch。
  if (detectMixedContent(url)) {
    return {
      latency: null,
      testing: false,
      error: true,
      errorReason: 'mixed-content',
    }
  }

  try {
    const startTime = performance.now()
    await fetch(url, {
      method: 'HEAD',
      mode: 'no-cors',
      cache: 'no-cache',
    })
    const endTime = performance.now()
    const latency = Math.round(endTime - startTime)

    return { latency, testing: false, error: false }
  } catch (_error) {
    // 浏览器对所有 fetch 失败（CORS、网络不可达、DNS、TLS 等）都抛出相同的
    // TypeError: Failed to fetch，这里无法进一步区分，统一归类为无法访问。
    return {
      latency: null,
      testing: false,
      error: true,
      errorReason: null,
    }
  }
}

/**
 * Open external speed test link
 */
export function openExternalSpeedTest(url: string): void {
  const encodedUrl = encodeURIComponent(url)
  const speedTestUrl = `https://www.tcptest.cn/http/${encodedUrl}`
  window.open(speedTestUrl, '_blank', 'noopener,noreferrer')
}

/**
 * Get default ping status
 */
export function getDefaultPingStatus(): PingStatus {
  return {
    latency: null,
    testing: false,
    error: false,
  }
}
