import { useState, useCallback, useRef, useEffect } from 'react';
import { DEFAULT_MESSAGES, DEFAULT_CONFIG, DEBUG_TABS } from '../utils/constants';
import { loadConfig, saveConfig } from '../components/playground/configStorage';

export const usePlaygroundState = () => {
  // 使用 ref 缓存初始配置，只加载一次
  const initialConfigRef = useRef(null);
  if (!initialConfigRef.current) {
    initialConfigRef.current = loadConfig();
  }
  const savedConfig = initialConfigRef.current;

  // 基础配置状态
  const [inputs, setInputs] = useState(savedConfig.inputs || DEFAULT_CONFIG.inputs);
  const [parameterEnabled, setParameterEnabled] = useState(
    savedConfig.parameterEnabled || DEFAULT_CONFIG.parameterEnabled
  );
  const [systemPrompt, setSystemPrompt] = useState(
    savedConfig.systemPrompt || DEFAULT_CONFIG.systemPrompt
  );
  const [showDebugPanel, setShowDebugPanel] = useState(
    savedConfig.showDebugPanel || DEFAULT_CONFIG.showDebugPanel
  );

  // UI状态
  const [showSettings, setShowSettings] = useState(false);
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [status, setStatus] = useState({});

  // 消息相关状态
  const [message, setMessage] = useState(DEFAULT_MESSAGES);

  // 调试状态
  const [debugData, setDebugData] = useState({
    request: null,
    response: null,
    timestamp: null,
    previewRequest: null,
    previewTimestamp: null
  });
  const [activeDebugTab, setActiveDebugTab] = useState(DEBUG_TABS.PREVIEW);
  const [previewPayload, setPreviewPayload] = useState(null);

  // 编辑状态
  const [editingMessageId, setEditingMessageId] = useState(null);
  const [editValue, setEditValue] = useState('');

  // Refs
  const sseSourceRef = useRef(null);
  const chatRef = useRef(null);
  const saveConfigTimeoutRef = useRef(null);

  // 配置更新函数
  const handleInputChange = useCallback((name, value) => {
    setInputs(prev => ({ ...prev, [name]: value }));
  }, []);

  const handleParameterToggle = useCallback((paramName) => {
    setParameterEnabled(prev => ({
      ...prev,
      [paramName]: !prev[paramName]
    }));
  }, []);

  // 配置保存
  const debouncedSaveConfig = useCallback(() => {
    if (saveConfigTimeoutRef.current) {
      clearTimeout(saveConfigTimeoutRef.current);
    }

    saveConfigTimeoutRef.current = setTimeout(() => {
      const configToSave = {
        inputs,
        parameterEnabled,
        systemPrompt,
        showDebugPanel,
      };
      saveConfig(configToSave);
    }, 1000);
  }, [inputs, parameterEnabled, systemPrompt, showDebugPanel]);

  // 配置导入/重置
  const handleConfigImport = useCallback((importedConfig) => {
    if (importedConfig.inputs) {
      setInputs(prev => ({ ...prev, ...importedConfig.inputs }));
    }
    if (importedConfig.parameterEnabled) {
      setParameterEnabled(prev => ({ ...prev, ...importedConfig.parameterEnabled }));
    }
    if (importedConfig.systemPrompt) {
      setSystemPrompt(importedConfig.systemPrompt);
    }
    if (typeof importedConfig.showDebugPanel === 'boolean') {
      setShowDebugPanel(importedConfig.showDebugPanel);
    }
  }, []);

  const handleConfigReset = useCallback(() => {
    setInputs(DEFAULT_CONFIG.inputs);
    setParameterEnabled(DEFAULT_CONFIG.parameterEnabled);
    setSystemPrompt(DEFAULT_CONFIG.systemPrompt);
    setShowDebugPanel(DEFAULT_CONFIG.showDebugPanel);
  }, []);

  return {
    // 配置状态
    inputs,
    parameterEnabled,
    systemPrompt,
    showDebugPanel,

    // UI状态
    showSettings,
    models,
    groups,
    status,

    // 消息状态
    message,

    // 调试状态
    debugData,
    activeDebugTab,
    previewPayload,

    // 编辑状态
    editingMessageId,
    editValue,

    // Refs
    sseSourceRef,
    chatRef,
    saveConfigTimeoutRef,

    // 更新函数
    setInputs,
    setParameterEnabled,
    setSystemPrompt,
    setShowDebugPanel,
    setShowSettings,
    setModels,
    setGroups,
    setStatus,
    setMessage,
    setDebugData,
    setActiveDebugTab,
    setPreviewPayload,
    setEditingMessageId,
    setEditValue,

    // 处理函数
    handleInputChange,
    handleParameterToggle,
    debouncedSaveConfig,
    handleConfigImport,
    handleConfigReset,
  };
}; 