export const DEFAULT_ENDPOINT = '/api/ratio_config';

/**
 * buildEndpointUrl: 拼接 baseUrl 与 endpoint，确保不会出现双斜杠或缺失斜杠问题。
 * 使用 URL 构造函数保证协议/域名安全；若 baseUrl 非标准 URL，则退回字符串拼接。
 * @param {string} baseUrl - 基础地址，例如 https://api.example.com
 * @param {string} endpoint - 接口路径，例如 /api/ratio_config
 * @returns {string}
 */
export const buildEndpointUrl = (baseUrl, endpoint) => {
  if (!baseUrl) return endpoint;
  try {
    return new URL(endpoint, baseUrl).toString();
  } catch (_) {
    // fallback 处理不规范的 baseUrl
    const cleanedBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
    const cleanedEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
    return `${cleanedBase}/${cleanedEndpoint}`;
  }
}; 