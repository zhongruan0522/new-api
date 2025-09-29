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

import { API, showError } from '../helpers';
import { 
  prepareCredentialRequestOptions, 
  buildAssertionResult, 
  isPasskeySupported 
} from '../helpers/passkey';

/**
 * 通用安全验证服务
 */
export class SecureVerificationService {
  /**
   * 检查用户可用的验证方式
   * @returns {Promise<{has2FA: boolean, hasPasskey: boolean, passkeySupported: boolean}>}
   */
  static async checkAvailableVerificationMethods() {
    try {
      console.log('Checking user verification methods...');
      const [twoFAResponse, passkeyResponse, passkeySupported] = await Promise.all([
        API.get('/api/user/2fa/status'),
        API.get('/api/user/passkey'),
        isPasskeySupported()
      ]);

      console.log('2FA response:', twoFAResponse);
      console.log('Passkey response:', passkeyResponse);
      console.log('Passkey browser support:', passkeySupported);

      const result = {
        has2FA: twoFAResponse.success && twoFAResponse.data?.enabled === true,
        hasPasskey: passkeyResponse.success && (passkeyResponse.data?.enabled === true || passkeyResponse.data?.status === 'enabled' || passkeyResponse.data !== null),
        passkeySupported
      };
      
      console.log('Final verification methods result:', result);
      return result;
    } catch (error) {
      console.error('Failed to check verification methods:', error);
      return {
        has2FA: false,
        hasPasskey: false,
        passkeySupported: false
      };
    }
  }

  /**
   * 执行2FA验证
   * @param {string} code - 验证码
   * @param {Function} apiCall - API调用函数，接收 {method: '2fa', code} 参数
   * @returns {Promise<any>} API响应结果
   */
  static async verify2FA(code, apiCall) {
    if (!code?.trim()) {
      throw new Error('请输入验证码或备用码');
    }

    return await apiCall({
      method: '2fa',
      code: code.trim()
    });
  }

  /**
   * 执行Passkey验证
   * @param {Function} apiCall - API调用函数，接收 {method: 'passkey'} 参数
   * @returns {Promise<any>} API响应结果
   */
  static async verifyPasskey(apiCall) {
    try {
      // 开始Passkey验证
      const beginResponse = await API.post('/api/user/passkey/verify/begin');
      if (!beginResponse.success) {
        throw new Error(beginResponse.message);
      }

      // 准备WebAuthn选项
      const publicKey = prepareCredentialRequestOptions(beginResponse.data);
      
      // 执行WebAuthn验证
      const credential = await navigator.credentials.get({ publicKey });
      if (!credential) {
        throw new Error('Passkey 验证被取消');
      }

      // 构建验证结果
      const assertionResult = buildAssertionResult(credential);
      
      // 完成验证
      const finishResponse = await API.post('/api/user/passkey/verify/finish', assertionResult);
      if (!finishResponse.success) {
        throw new Error(finishResponse.message);
      }

      // 调用业务API
      return await apiCall({
        method: 'passkey'
      });
    } catch (error) {
      if (error.name === 'NotAllowedError') {
        throw new Error('Passkey 验证被取消或超时');
      } else if (error.name === 'InvalidStateError') {
        throw new Error('Passkey 验证状态无效');
      } else {
        throw error;
      }
    }
  }

  /**
   * 通用验证方法，根据验证类型执行相应的验证流程
   * @param {string} method - 验证方式: '2fa' | 'passkey'
   * @param {Object} params - 参数对象
   * @param {string} params.code - 2FA验证码（当method为'2fa'时必需）
   * @param {Function} params.apiCall - API调用函数
   * @returns {Promise<any>} API响应结果
   */
  static async verify(method, { code, apiCall }) {
    switch (method) {
      case '2fa':
        return await this.verify2FA(code, apiCall);
      case 'passkey':
        return await this.verifyPasskey(apiCall);
      default:
        throw new Error(`不支持的验证方式: ${method}`);
    }
  }
}

/**
 * 预设的API调用函数工厂
 */
export const createApiCalls = {
  /**
   * 创建查看渠道密钥的API调用
   * @param {number} channelId - 渠道ID
   */
  viewChannelKey: (channelId) => async (verificationData) => {
    return await API.post(`/api/channel/${channelId}/key`, verificationData);
  },

  /**
   * 创建自定义API调用
   * @param {string} url - API URL
   * @param {string} method - HTTP方法，默认为 'POST'
   * @param {Object} extraData - 额外的请求数据
   */
  custom: (url, method = 'POST', extraData = {}) => async (verificationData) => {
    const data = { ...extraData, ...verificationData };
    
    switch (method.toUpperCase()) {
      case 'GET':
        return await API.get(url, { params: data });
      case 'POST':
        return await API.post(url, data);
      case 'PUT':
        return await API.put(url, data);
      case 'DELETE':
        return await API.delete(url, { data });
      default:
        throw new Error(`不支持的HTTP方法: ${method}`);
    }
  }
};