const STORAGE_KEY = 'playground_config';

const DEFAULT_CONFIG = {
  inputs: {
    model: 'deepseek-r1',
    group: '',
    max_tokens: 0,
    temperature: 0,
    top_p: 1,
    frequency_penalty: 0,
    presence_penalty: 0,
    seed: null,
    stream: true,
    imageUrls: [],
    imageEnabled: false,
  },
  parameterEnabled: {
    max_tokens: true,
    temperature: true,
    top_p: false,
    frequency_penalty: false,
    presence_penalty: false,
    seed: false,
  },
  systemPrompt: 'You are a helpful assistant. You can help me by answering my questions. You can also ask me questions.',
  showDebugPanel: false,
};

/**
 * 保存配置到 localStorage
 * @param {Object} config - 要保存的配置对象
 */
export const saveConfig = (config) => {
  try {
    const configToSave = {
      ...config,
      timestamp: new Date().toISOString(),
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(configToSave));
    console.log('配置已保存到本地存储');
  } catch (error) {
    console.error('保存配置失败:', error);
  }
};

/**
 * 从 localStorage 加载配置
 * @returns {Object} 配置对象，如果不存在则返回默认配置
 */
export const loadConfig = () => {
  try {
    const savedConfig = localStorage.getItem(STORAGE_KEY);
    if (savedConfig) {
      const parsedConfig = JSON.parse(savedConfig);

      const mergedConfig = {
        inputs: {
          ...DEFAULT_CONFIG.inputs,
          ...parsedConfig.inputs,
        },
        parameterEnabled: {
          ...DEFAULT_CONFIG.parameterEnabled,
          ...parsedConfig.parameterEnabled,
        },
        systemPrompt: parsedConfig.systemPrompt || DEFAULT_CONFIG.systemPrompt,
        showDebugPanel: parsedConfig.showDebugPanel || DEFAULT_CONFIG.showDebugPanel,
      };

      console.log('配置已从本地存储加载');
      return mergedConfig;
    }
  } catch (error) {
    console.error('加载配置失败:', error);
  }

  console.log('使用默认配置');
  return DEFAULT_CONFIG;
};

/**
 * 清除保存的配置
 */
export const clearConfig = () => {
  try {
    localStorage.removeItem(STORAGE_KEY);
    console.log('配置已清除');
  } catch (error) {
    console.error('清除配置失败:', error);
  }
};

/**
 * 检查是否有保存的配置
 * @returns {boolean} 是否存在保存的配置
 */
export const hasStoredConfig = () => {
  try {
    return localStorage.getItem(STORAGE_KEY) !== null;
  } catch (error) {
    console.error('检查配置失败:', error);
    return false;
  }
};

/**
 * 获取配置的最后保存时间
 * @returns {string|null} 最后保存时间的 ISO 字符串
 */
export const getConfigTimestamp = () => {
  try {
    const savedConfig = localStorage.getItem(STORAGE_KEY);
    if (savedConfig) {
      const parsedConfig = JSON.parse(savedConfig);
      return parsedConfig.timestamp || null;
    }
  } catch (error) {
    console.error('获取配置时间戳失败:', error);
  }
  return null;
};

/**
 * 导出配置为 JSON 文件
 * @param {Object} config - 要导出的配置
 */
export const exportConfig = (config) => {
  try {
    const configToExport = {
      ...config,
      exportTime: new Date().toISOString(),
      version: '1.0',
    };

    const dataStr = JSON.stringify(configToExport, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });

    const link = document.createElement('a');
    link.href = URL.createObjectURL(dataBlob);
    link.download = `playground-config-${new Date().toISOString().split('T')[0]}.json`;
    link.click();

    URL.revokeObjectURL(link.href);

    console.log('配置已导出');
  } catch (error) {
    console.error('导出配置失败:', error);
  }
};

/**
 * 从文件导入配置
 * @param {File} file - 包含配置的 JSON 文件
 * @returns {Promise<Object>} 导入的配置对象
 */
export const importConfig = (file) => {
  return new Promise((resolve, reject) => {
    try {
      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const importedConfig = JSON.parse(e.target.result);

          if (importedConfig.inputs && importedConfig.parameterEnabled) {
            console.log('配置已从文件导入');
            resolve(importedConfig);
          } else {
            reject(new Error('配置文件格式无效'));
          }
        } catch (parseError) {
          reject(new Error('解析配置文件失败: ' + parseError.message));
        }
      };
      reader.onerror = () => reject(new Error('读取文件失败'));
      reader.readAsText(file);
    } catch (error) {
      reject(new Error('导入配置失败: ' + error.message));
    }
  });
}; 