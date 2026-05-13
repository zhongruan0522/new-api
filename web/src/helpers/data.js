/*
Copyright (C) 2025 QuantumNous

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

export function setStatusData(data) {
  localStorage.setItem('status', JSON.stringify(data));
  localStorage.setItem('system_name', data.system_name);
  localStorage.setItem('logo', data.logo);
  localStorage.setItem('footer_html', data.footer_html);
  localStorage.setItem('quota_per_unit', data.quota_per_unit);
  // 清理已移除的「聊天/操练场」相关缓存，避免遗留脏数据。
  localStorage.removeItem('chats');
  localStorage.removeItem('chat_link');
  localStorage.removeItem('chat_link2');
  // 清理已移除的额度展示类型相关缓存
  localStorage.removeItem('display_in_currency');
  localStorage.removeItem('quota_display_type');
  localStorage.setItem('enable_data_export', data.enable_data_export);
  localStorage.setItem(
    'data_export_default_time',
    data.data_export_default_time,
  );
  if (data.docs_link) {
    localStorage.setItem('docs_link', data.docs_link);
  } else {
    localStorage.removeItem('docs_link');
  }
}

export function setUserData(data) {
  localStorage.setItem('user', JSON.stringify(data));
}
