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
import { useState, useEffect } from 'react';
import { API } from '../../helpers';

/**
 * 用户权限钩子 - 从后端获取用户权限
 */
export const useUserPermissions = () => {
  const [permissions, setPermissions] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 加载用户权限（从用户信息接口获取）
  const loadPermissions = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await API.get('/api/user/self');
      if (res.data.success) {
        const userPermissions = res.data.data.permissions;
        setPermissions(userPermissions);
      } else {
        setError(res.data.message || '获取权限失败');
      }
    } catch (error) {
      setError('网络错误，请重试');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPermissions();
  }, []);

  return {
    permissions,
    loading,
    error,
    loadPermissions,
  };
};

export default useUserPermissions;
