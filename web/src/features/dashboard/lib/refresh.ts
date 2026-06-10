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
const DEFAULT_DATA_DASHBOARD_REFRESH_MINUTES = 60

function readRefreshIntervalMinutes(status: unknown): number {
  if (!status || typeof status !== 'object') {
    return DEFAULT_DATA_DASHBOARD_REFRESH_MINUTES
  }

  const raw = (status as Record<string, unknown>).data_export_interval
  const value = typeof raw === 'number' ? raw : Number(raw)
  if (!Number.isFinite(value) || value < 1) {
    return DEFAULT_DATA_DASHBOARD_REFRESH_MINUTES
  }

  return Math.min(value, 1440)
}

export function getDataDashboardRefreshIntervalMs(status: unknown): number {
  return readRefreshIntervalMinutes(status) * 60 * 1000
}
