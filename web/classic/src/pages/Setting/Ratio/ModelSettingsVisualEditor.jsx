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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import {
  Table,
  Button,
  Input,
  RadioGroup,
  Radio,
  Checkbox,
  Tag,
  Switch,
  Banner,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconPlus,
  IconSearch,
  IconSave,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

// 所有按量计费的倍率字段（内部存储用，前端统一以价格展示）
const PER_TOKEN_RATIO_FIELDS = [
  'ratio',
  'completionRatio',
  'cacheRatio',
  'createCacheRatio',
  'audioRatio',
  'audioCompletionRatio',
];

const MODEL_PRICING_FIELDS = ['price', ...PER_TOKEN_RATIO_FIELDS];
const PER_TOKEN_ZERO_FILL_FIELDS = [...PER_TOKEN_RATIO_FIELDS];
const DECIMAL_PRECISION = 12;
const PER_TOKEN_OUTPUT_KEY_MAP = {
  ratio: 'ModelRatio',
  completionRatio: 'CompletionRatio',
  cacheRatio: 'CacheRatio',
  createCacheRatio: 'CreateCacheRatio',
  audioRatio: 'AudioRatio',
  audioCompletionRatio: 'AudioCompletionRatio',
};
const DEFAULT_CONTEXT_TIER_BOUNDS = [
  { minTokens: '0', maxTokens: '200000', name: '<200K' },
  { minTokens: '200000', maxTokens: '1000000', name: '200K~1000K' },
];
const CONTEXT_TIER_PRICE_FIELDS = [
  'tokenPrice',
  'completionTokenPrice',
  'cacheTokenPrice',
  'createCacheTokenPrice',
  'audioTokenPrice',
  'audioCompletionTokenPrice',
];

const hasValue = (value) =>
  value !== '' && value !== undefined && value !== null;

const normalizeEditableValue = (value) => (hasValue(value) ? `${value}` : '');

const normalizeNumberValue = (value) => {
  if (!Number.isFinite(value)) {
    return null;
  }
  return Number(value.toFixed(DECIMAL_PRECISION));
};

const formatNumberValue = (value) => {
  const normalizedValue = normalizeNumberValue(value);
  return normalizedValue === null ? '' : normalizedValue.toString();
};

const normalizeNumericEditableValue = (value) => {
  if (!hasValue(value)) {
    return '';
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? formatNumberValue(parsed) : `${value}`;
};

const isEditableNumberInput = (value) => /^\d*(\.\d*)?$/.test(value);

const shouldDeferNumericSync = (value) =>
  value === '.' || (typeof value === 'string' && value.endsWith('.'));

const createEmptyModel = (name = '') => ({
  name,
  price: '',
  ratio: '',
  completionRatio: '',
  cacheRatio: '',
  createCacheRatio: '',
  audioRatio: '',
  audioCompletionRatio: '',
  contextPricing: { enabled: false, tiers: [] },
  hasConflict: false,
  isUnset: true,
});

const hasAnyPricingConfig = (model) =>
  MODEL_PRICING_FIELDS.some((field) => hasValue(model?.[field])) ||
  Boolean(model?.contextPricing?.enabled);

const sortModels = (modelList) =>
  [...modelList].sort((left, right) => {
    if (Boolean(left.isUnset) !== Boolean(right.isUnset)) {
      return left.isUnset ? -1 : 1;
    }
    return left.name.localeCompare(right.name);
  });

const parseInputNumber = (value) => {
  if (!hasValue(value)) {
    return null;
  }
  const parsed = Number(value);
  const normalizedValue = normalizeNumberValue(parsed);
  return normalizedValue === null ? null : normalizedValue;
};

// 倍率 → 价格：ratio * 2 = $/1M tokens（因为 ratio 1 = $0.002/1K = $2/1M）
const ratioToPrice = (ratio) => normalizeNumberValue(ratio * 2);

// 价格 → 倍率
const priceToRatio = (price) => normalizeNumberValue(price / 2);

// 相对倍率计算（用于补全、缓存等相对于基础倍率的换算）
const calculateRelativeRatio = (targetPrice, basePrice) => {
  if (
    !Number.isFinite(targetPrice) ||
    !Number.isFinite(basePrice) ||
    basePrice <= 0
  ) {
    return '';
  }
  return formatNumberValue(targetPrice / basePrice);
};

const hasPerTokenPricing = (model) =>
  PER_TOKEN_RATIO_FIELDS.some((field) => hasValue(model?.[field]));

const buildConflictState = (model) =>
  hasValue(model?.price) && hasPerTokenPricing(model);

// 从倍率字段构建价格显示值（用于详情面板的价格字段回显）
const buildPriceFieldsFromRatios = (model) => {
  const ratio = parseInputNumber(model?.ratio);
  const completionRatio = parseInputNumber(model?.completionRatio);
  const cacheRatio = parseInputNumber(model?.cacheRatio);
  const createCacheRatio = parseInputNumber(model?.createCacheRatio);
  const audioRatio = parseInputNumber(model?.audioRatio);
  const audioCompletionRatio = parseInputNumber(model?.audioCompletionRatio);

  const tokenPrice = ratio !== null ? ratioToPrice(ratio) : null;

  return {
    tokenPrice: tokenPrice !== null ? formatNumberValue(tokenPrice) : '',
    completionTokenPrice:
      tokenPrice !== null && completionRatio !== null
        ? formatNumberValue(tokenPrice * completionRatio)
        : '',
    cacheTokenPrice:
      tokenPrice !== null && cacheRatio !== null
        ? formatNumberValue(tokenPrice * cacheRatio)
        : '',
    createCacheTokenPrice:
      tokenPrice !== null && createCacheRatio !== null
        ? formatNumberValue(tokenPrice * createCacheRatio)
        : '',
    audioTokenPrice:
      tokenPrice !== null && audioRatio !== null
        ? formatNumberValue(tokenPrice * audioRatio)
        : '',
    audioCompletionTokenPrice:
      tokenPrice !== null &&
      audioRatio !== null &&
      audioCompletionRatio !== null
        ? formatNumberValue(tokenPrice * audioRatio * audioCompletionRatio)
        : '',
  };
};

// 从价格字段同步回倍率字段（保存前调用，确保存储的是倍率）
const syncRatioFieldsFromPrices = (model) => {
  const tokenPrice = parseInputNumber(model?.tokenPrice);
  const completionTokenPrice = parseInputNumber(model?.completionTokenPrice);
  const cacheTokenPrice = parseInputNumber(model?.cacheTokenPrice);
  const createCacheTokenPrice = parseInputNumber(model?.createCacheTokenPrice);
  const audioTokenPrice = parseInputNumber(model?.audioTokenPrice);
  const audioCompletionTokenPrice = parseInputNumber(
    model?.audioCompletionTokenPrice,
  );

  const updatedModel = {
    ...(model || {}),
    ratio:
      tokenPrice !== null
        ? formatNumberValue(priceToRatio(tokenPrice))
        : hasValue(model?.tokenPrice)
          ? ''
          : normalizeNumericEditableValue(model?.ratio),
    completionRatio: hasValue(model?.completionTokenPrice)
      ? calculateRelativeRatio(completionTokenPrice, tokenPrice)
      : normalizeNumericEditableValue(model?.completionRatio),
    cacheRatio: hasValue(model?.cacheTokenPrice)
      ? calculateRelativeRatio(cacheTokenPrice, tokenPrice)
      : normalizeNumericEditableValue(model?.cacheRatio),
    createCacheRatio: hasValue(model?.createCacheTokenPrice)
      ? calculateRelativeRatio(createCacheTokenPrice, tokenPrice)
      : normalizeNumericEditableValue(model?.createCacheRatio),
    audioRatio: hasValue(model?.audioTokenPrice)
      ? calculateRelativeRatio(audioTokenPrice, tokenPrice)
      : normalizeNumericEditableValue(model?.audioRatio),
    audioCompletionRatio: hasValue(model?.audioCompletionTokenPrice)
      ? calculateRelativeRatio(audioCompletionTokenPrice, audioTokenPrice)
      : normalizeNumericEditableValue(model?.audioCompletionRatio),
  };

  updatedModel.hasConflict = buildConflictState(updatedModel);
  return updatedModel;
};

const clearPerTokenPricing = (model) => ({
  ...(model || {}),
  ratio: '',
  completionRatio: '',
  cacheRatio: '',
  createCacheRatio: '',
  audioRatio: '',
  audioCompletionRatio: '',
  hasConflict: false,
});

// 倍率 → 显示价格（用于表格列渲染）
const ratioToDisplayPrice = (ratio) => {
  if (!hasValue(ratio)) return '';
  const r = Number(ratio);
  return Number.isFinite(r) ? formatNumberValue(r * 2) : '';
};

// 显示价格 → 倍率（用于表格列输入后存储）
const displayPriceToRatio = (price) => {
  if (!hasValue(price)) return '';
  const p = Number(price);
  return Number.isFinite(p) ? formatNumberValue(p / 2) : '';
};

const backendTierToEditable = (tier = {}) => {
  const modelRatio = parseInputNumber(tier.model_ratio);
  const basePrice = modelRatio !== null ? modelRatio * 2 : 0;
  const audioInputPrice = basePrice * (Number(tier.audio_ratio) || 0);

  return {
    name: tier.name || '',
    minTokens:
      tier.min_tokens === undefined || tier.min_tokens === null
        ? ''
        : `${tier.min_tokens}`,
    maxTokens:
      tier.max_tokens === undefined || tier.max_tokens === null
        ? ''
        : `${tier.max_tokens}`,
    tokenPrice: formatNumberValue(basePrice),
    completionTokenPrice: formatNumberValue(
      basePrice * (Number(tier.completion_ratio) || 0),
    ),
    cacheTokenPrice: formatNumberValue(
      basePrice * (Number(tier.cache_ratio) || 0),
    ),
    createCacheTokenPrice: formatNumberValue(
      basePrice * (Number(tier.create_cache_ratio) || 0),
    ),
    audioTokenPrice: formatNumberValue(audioInputPrice),
    audioCompletionTokenPrice: formatNumberValue(
      audioInputPrice * (Number(tier.audio_completion_ratio) || 0),
    ),
  };
};

const editableTierToBackend = (tier = {}) => {
  const tokenPrice = parseInputNumber(tier.tokenPrice) || 0;
  const audioTokenPrice = parseInputNumber(tier.audioTokenPrice) || 0;
  const toRatio = (price) => {
    const parsed = parseInputNumber(price);
    if (parsed === null || tokenPrice === 0) return 0;
    return normalizeNumberValue(parsed / tokenPrice) || 0;
  };
  const toAudioCompletionRatio = (price) => {
    const parsed = parseInputNumber(price);
    if (parsed === null || audioTokenPrice === 0) return 0;
    return normalizeNumberValue(parsed / audioTokenPrice) || 0;
  };
  const maxTokens = hasValue(tier.maxTokens) ? Number(tier.maxTokens) : null;

  const backend = {
    min_tokens: Number(tier.minTokens),
    model_ratio: priceToRatio(tokenPrice) || 0,
    completion_ratio: toRatio(tier.completionTokenPrice),
    cache_ratio: toRatio(tier.cacheTokenPrice),
    create_cache_ratio: toRatio(tier.createCacheTokenPrice),
    audio_ratio: toRatio(tier.audioTokenPrice),
    audio_completion_ratio: toAudioCompletionRatio(
      tier.audioCompletionTokenPrice,
    ),
  };
  if (hasValue(tier.name)) {
    backend.name = tier.name;
  }
  if (maxTokens !== null) {
    backend.max_tokens = maxTokens;
  }
  return backend;
};

const buildEditableContextPricing = (config) => {
  if (config?.enabled && Array.isArray(config.tiers)) {
    return {
      enabled: true,
      tiers: config.tiers.map((tier) => backendTierToEditable(tier)),
    };
  }
  return { enabled: false, tiers: [] };
};

const createDefaultContextPricing = (model = {}) => {
  const priceFields = buildPriceFieldsFromRatios(model);
  return {
    enabled: true,
    tiers: DEFAULT_CONTEXT_TIER_BOUNDS.map((bounds) => ({
      ...bounds,
      tokenPrice: priceFields.tokenPrice || '0',
      completionTokenPrice: priceFields.completionTokenPrice || '0',
      cacheTokenPrice: priceFields.cacheTokenPrice || '0',
      createCacheTokenPrice: priceFields.createCacheTokenPrice || '0',
      audioTokenPrice: priceFields.audioTokenPrice || '0',
      audioCompletionTokenPrice: priceFields.audioCompletionTokenPrice || '0',
    })),
  };
};

const normalizeContextPricingForSave = (contextPricing) => ({
  enabled: true,
  tiers: (contextPricing?.tiers || []).map((tier) =>
    editableTierToBackend(tier),
  ),
});

const formatContextRangeSummary = (tier) => {
  if (!hasValue(tier?.maxTokens)) {
    return `≥${tier?.minTokens || 0}`;
  }
  return `${tier?.minTokens || 0}~${tier.maxTokens}`;
};

/**
 * 获取模型的计费模式摘要文本，用于左侧列表展示
 */
const getModelPricingSummary = (model) => {
  if (model?.contextPricing?.enabled) {
    const tierCount = model.contextPricing.tiers?.length || 0;
    return `分段计费 ${tierCount} 档`;
  }
  if (hasValue(model?.price)) {
    return `按次 $${model.price} / 次`;
  }

  // 按量计费模式
  const parts = [];
  const inputPrice = ratioToDisplayPrice(model?.ratio);
  if (inputPrice) {
    parts.push(`输入 $${inputPrice}`);
  }

  // 统计额外已配置项数量
  let extraCount = 0;
  if (hasValue(model?.completionRatio)) extraCount++;
  if (hasValue(model?.cacheRatio)) extraCount++;
  if (hasValue(model?.createCacheRatio)) extraCount++;
  if (hasValue(model?.audioRatio)) extraCount++;
  if (hasValue(model?.audioCompletionRatio)) extraCount++;

  if (extraCount > 0) {
    parts.push(`额外价格项 ${extraCount}`);
  }

  return parts.length > 0 ? parts.join('，') : '';
};

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [models, setModels] = useState([]);
  const [enabledModels, setEnabledModels] = useState([]);
  const [selectedModelName, setSelectedModelName] = useState(null);
  const [searchText, setSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [conflictOnly, setConflictOnly] = useState(false);
  const [listPageSize, setListPageSize] = useState(100); // 左侧列表每页条数（可变）

  // 当前选中模型的编辑状态（右侧面板使用）
  const [editingModel, setEditingModel] = useState(null);

  const getAllEnabledModels = async () => {
    try {
      const res = await API.get('/api/channel/models_enabled');
      const { success, message, data } = res.data;
      if (success) {
        setEnabledModels(Array.isArray(data) ? data : []);
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('获取启用模型失败:'), error);
      showError(t('获取启用模型失败'));
    }
  };

  useEffect(() => {
    getAllEnabledModels();
  }, []);

  useEffect(() => {
    try {
      const modelPrice = JSON.parse(props.options.ModelPrice || '{}');
      const modelRatio = JSON.parse(props.options.ModelRatio || '{}');
      const completionRatio = JSON.parse(props.options.CompletionRatio || '{}');
      const cacheRatio = JSON.parse(props.options.CacheRatio || '{}');
      const createCacheRatio = JSON.parse(
        props.options.CreateCacheRatio || '{}',
      );
      const audioRatio = JSON.parse(props.options.AudioRatio || '{}');
      const audioCompletionRatio = JSON.parse(
        props.options.AudioCompletionRatio || '{}',
      );
      const contextPricing = JSON.parse(props.options.ContextPricing || '{}');

      const configuredModelNames = new Set([
        ...Object.keys(modelPrice),
        ...Object.keys(modelRatio),
        ...Object.keys(completionRatio),
        ...Object.keys(cacheRatio),
        ...Object.keys(createCacheRatio),
        ...Object.keys(audioRatio),
        ...Object.keys(audioCompletionRatio),
        ...Object.keys(contextPricing),
      ]);

      const configuredModelData = Array.from(configuredModelNames).map(
        (name) => {
          const price = modelPrice[name] === undefined ? '' : modelPrice[name];
          const ratio = modelRatio[name] === undefined ? '' : modelRatio[name];
          const comp =
            completionRatio[name] === undefined ? '' : completionRatio[name];
          const cache = cacheRatio[name] === undefined ? '' : cacheRatio[name];
          const createCache =
            createCacheRatio[name] === undefined ? '' : createCacheRatio[name];
          const audio = audioRatio[name] === undefined ? '' : audioRatio[name];
          const audioComp =
            audioCompletionRatio[name] === undefined
              ? ''
              : audioCompletionRatio[name];
          const currentContextPricing =
            contextPricing[name] === undefined
              ? { enabled: false, tiers: [] }
              : contextPricing[name];

          return {
            name,
            price: normalizeNumericEditableValue(price),
            ratio: normalizeNumericEditableValue(ratio),
            completionRatio: normalizeNumericEditableValue(comp),
            cacheRatio: normalizeNumericEditableValue(cache),
            createCacheRatio: normalizeNumericEditableValue(createCache),
            audioRatio: normalizeNumericEditableValue(audio),
            audioCompletionRatio: normalizeNumericEditableValue(audioComp),
            contextPricing: buildEditableContextPricing(currentContextPricing),
            isUnset: false,
            hasConflict: buildConflictState({
              price: normalizeNumericEditableValue(price),
              ratio: normalizeNumericEditableValue(ratio),
              completionRatio: normalizeNumericEditableValue(comp),
              cacheRatio: normalizeNumericEditableValue(cache),
              createCacheRatio: normalizeNumericEditableValue(createCache),
              audioRatio: normalizeNumericEditableValue(audio),
              audioCompletionRatio: normalizeNumericEditableValue(audioComp),
            }),
          };
        },
      );

      const unsetModelData = Array.from(new Set(enabledModels))
        .filter((name) => !configuredModelNames.has(name))
        .map((name) => createEmptyModel(name))
        .filter((model) => !hasAnyPricingConfig(model));

      setModels(sortModels([...unsetModelData, ...configuredModelData]));
    } catch (error) {
      console.error('JSON解析错误:', error);
    }
  }, [enabledModels, props.options]);

  // 当前选中的模型对象
  const selectedModel = useMemo(() => {
    if (!selectedModelName) return null;
    return models.find((m) => m.name === selectedModelName) || null;
  }, [models, selectedModelName]);

  // 同步 editingModel 与选中模型
  useEffect(() => {
    if (selectedModel) {
      if (selectedModel?.contextPricing?.enabled) {
        setEditingModel({
          ...selectedModel,
          pricingMode: 'per-token',
          price: '',
        });
      } else if (hasValue(selectedModel?.price)) {
        // 按次计费
        setEditingModel({
          ...selectedModel,
          pricingMode: 'per-request',
        });
      } else {
        // 按量计费：构建价格字段用于回显
        setEditingModel({
          ...selectedModel,
          ...buildPriceFieldsFromRatios(selectedModel),
          pricingMode: 'per-token',
        });
      }
    } else {
      setEditingModel(null);
    }
  }, [selectedModel]);

  const getPagedData = (data, currentPage, pageSize) => {
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    return data.slice(start, end);
  };

  const filteredModels = models.filter((model) => {
    const keywordMatch = searchText
      ? model.name.toLowerCase().includes(searchText.toLowerCase())
      : true;
    const conflictMatch = conflictOnly ? model.hasConflict : true;
    return keywordMatch && conflictMatch;
  });

  const pagedData = getPagedData(filteredModels, currentPage, listPageSize);

  const validateContextPricingModel = (model) => {
    if (!model?.contextPricing?.enabled) return true;
    const tiers = model.contextPricing.tiers || [];
    if (tiers.length === 0) {
      showError(`${model.name}: ${t('分段计费至少需要一个区间')}`);
      return false;
    }

    const normalizedRanges = [];
    for (let index = 0; index < tiers.length; index++) {
      const tier = tiers[index];
      const label = `${model.name} ${t('第')} ${index + 1} ${t('档')}`;
      if (!hasValue(tier.minTokens)) {
        showError(`${label}: ${t('请输入区间下限')}`);
        return false;
      }
      const minTokens = Number(tier.minTokens);
      const maxTokens = hasValue(tier.maxTokens)
        ? Number(tier.maxTokens)
        : null;
      if (!Number.isInteger(minTokens) || minTokens < 0) {
        showError(`${label}: ${t('区间下限必须是非负整数')}`);
        return false;
      }
      if (
        maxTokens !== null &&
        (!Number.isInteger(maxTokens) || maxTokens <= minTokens)
      ) {
        showError(`${label}: ${t('区间上限必须大于下限')}`);
        return false;
      }
      for (const field of CONTEXT_TIER_PRICE_FIELDS) {
        if (!hasValue(tier[field])) {
          showError(`${label}: ${t('价格字段不能为空')}`);
          return false;
        }
        const value = Number(tier[field]);
        if (!Number.isFinite(value) || value < 0) {
          showError(`${label}: ${t('价格必须是非负数字')}`);
          return false;
        }
      }
      normalizedRanges.push({ minTokens, maxTokens, index });
    }

    normalizedRanges.sort((left, right) => left.minTokens - right.minTokens);
    for (let index = 1; index < normalizedRanges.length; index++) {
      const prev = normalizedRanges[index - 1];
      const current = normalizedRanges[index];
      if (prev.maxTokens === null || current.minTokens < prev.maxTokens) {
        showError(`${model.name}: ${t('分段计费区间不能重叠')}`);
        return false;
      }
    }

    return true;
  };

  /** 保存所有模型数据到后端 */
  const SubmitData = async () => {
    setLoading(true);
    const output = {
      ModelPrice: {},
      ModelRatio: {},
      CompletionRatio: {},
      CacheRatio: {},
      CreateCacheRatio: {},
      AudioRatio: {},
      AudioCompletionRatio: {},
      ContextPricing: {},
    };
    try {
      // 如果当前正在编辑某个模型，先同步到 models 列表
      let modelsToSave = models;
      if (editingModel && selectedModelName) {
        modelsToSave = saveEditingModelToList(modelsToSave);
      }

      for (const model of modelsToSave) {
        if (!validateContextPricingModel(model)) {
          setLoading(false);
          return;
        }
      }

      modelsToSave.forEach((model) => {
        if (model.contextPricing?.enabled) {
          output.ContextPricing[model.name] = normalizeContextPricingForSave(
            model.contextPricing,
          );
          return;
        }
        if (model.price !== '') {
          output.ModelPrice[model.name] = parseInputNumber(model.price);
        } else {
          const hasPerTokenValue = PER_TOKEN_ZERO_FILL_FIELDS.some((field) =>
            hasValue(model[field]),
          );

          if (hasPerTokenValue) {
            PER_TOKEN_ZERO_FILL_FIELDS.forEach((field) => {
              const targetKey = PER_TOKEN_OUTPUT_KEY_MAP[field];
              output[targetKey][model.name] = hasValue(model[field])
                ? parseInputNumber(model[field])
                : 0;
            });
          }
        }
      });

      const finalOutput = {
        ModelPrice: JSON.stringify(output.ModelPrice, null, 2),
        ModelRatio: JSON.stringify(output.ModelRatio, null, 2),
        CompletionRatio: JSON.stringify(output.CompletionRatio, null, 2),
        CacheRatio: JSON.stringify(output.CacheRatio, null, 2),
        CreateCacheRatio: JSON.stringify(output.CreateCacheRatio, null, 2),
        AudioRatio: JSON.stringify(output.AudioRatio, null, 2),
        AudioCompletionRatio: JSON.stringify(
          output.AudioCompletionRatio,
          null,
          2,
        ),
        ContextPricing: JSON.stringify(output.ContextPricing, null, 2),
      };

      const requestQueue = Object.entries(finalOutput).map(([key, value]) => {
        return API.put('/api/option/', {
          key,
          value,
        });
      });

      const results = await Promise.all(requestQueue);

      if (requestQueue.length === 1) {
        if (results.includes(undefined)) return;
      } else if (requestQueue.length > 1) {
        if (results.includes(undefined)) {
          return showError('部分保存失败，请重试');
        }
      }

      for (const res of results) {
        if (!res.data.success) {
          return showError(res.data.message);
        }
      }

      showSuccess('保存成功');
      props.refresh();
    } catch (error) {
      console.error('保存失败:', error);
      showError('保存失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  /**
   * 将当前 editingModel 的修改同步回 models 列表，返回新列表
   */
  const saveEditingModelToList = (baseModels) => {
    if (!editingModel || !selectedModelName) return baseModels;

    let valuesToSave = { ...editingModel };

    if (editingModel.pricingMode === 'per-token') {
      if (editingModel.contextPricing?.enabled) {
        valuesToSave = {
          ...valuesToSave,
          price: '',
          ratio: '',
          completionRatio: '',
          cacheRatio: '',
          createCacheRatio: '',
          audioRatio: '',
          audioCompletionRatio: '',
          contextPricing: {
            enabled: true,
            tiers: editingModel.contextPricing.tiers || [],
          },
        };
      } else {
        valuesToSave = syncRatioFieldsFromPrices(valuesToSave);
        valuesToSave.price = '';
        valuesToSave.contextPricing = { enabled: false, tiers: [] };
      }
    } else {
      valuesToSave = clearPerTokenPricing(valuesToSave);
      valuesToSave.contextPricing = { enabled: false, tiers: [] };
    }

    // 移除临时字段
    delete valuesToSave.tokenPrice;
    delete valuesToSave.completionTokenPrice;
    delete valuesToSave.cacheTokenPrice;
    delete valuesToSave.createCacheTokenPrice;
    delete valuesToSave.audioTokenPrice;
    delete valuesToSave.audioCompletionTokenPrice;
    delete valuesToSave.pricingMode;

    return baseModels.map((model) => {
      if (model.name !== selectedModelName) return model;
      const updated = { ...model, ...valuesToSave };
      updated.hasConflict = buildConflictState(updated);
      return updated;
    });
  };

  /** 处理左侧列表点击选中 */
  const handleSelectModel = (record) => {
    setSelectedModelName(record.name);
  };

  /** 处理删除模型 */
  const handleDeleteModel = (name) => {
    setModels((prev) => {
      const next = prev.filter((model) => model.name !== name);
      const totalPages = Math.max(1, Math.ceil(next.length / listPageSize));
      if (currentPage > totalPages) {
        setCurrentPage(totalPages);
      }
      return next;
    });
    if (selectedModelName === name) {
      setSelectedModelName(null);
      setEditingModel(null);
    }
  };

  /** 处理添加新模型 */
  const handleAddModel = () => {
    // 先提交当前正在编辑的修改
    commitEditingChanges();
    const newName = t('新模型') + '_' + Date.now().toString(36);
    const newModel = createEmptyModel(newName);
    newModel.isUnset = false;
    setModels((prev) => sortModels([newModel, ...prev]));
    setSelectedModelName(newName);
  };

  /** 更新 editingModel 的指定字段 */
  const updateEditingField = (field, value) => {
    setEditingModel((prev) => {
      if (!prev) return prev;

      // 数字校验：非空值必须是合法数字
      if (hasValue(value) && !isEditableNumberInput(value)) {
        showError(t('请输入数字'));
        return prev;
      }

      const updated = { ...prev, [field]: value };

      // 如果修改了基础价格，需要联动更新相对倍率字段
      if (field === 'tokenPrice' && prev.pricingMode === 'per-token') {
        return syncRatioFieldsFromPrices(updated);
      }
      if (field === 'audioTokenPrice' && prev.pricingMode === 'per-token') {
        return syncRatioFieldsFromPrices(updated);
      }

      updated.hasConflict = buildConflictState(updated);
      return updated;
    });
  };

  /** 切换计费模式 */
  const handlePricingModeChange = (newMode) => {
    if (!editingModel) return;

    setEditingModel((prev) => {
      if (!prev) return prev;
      if (prev.pricingMode === newMode) return prev; // 模式未变则跳过

      if (newMode === 'per-token') {
        // 从按次切到按量：基于当前编辑状态，从倍率重建价格字段
        return {
          ...prev,
          ...buildPriceFieldsFromRatios(prev),
          pricingMode: 'per-token',
          price: '', // 清除固定价格
        };
      } else {
        // 从按量切到按次：清除按量字段，保留当前编辑的固定价格
        return {
          ...prev,
          pricingMode: 'per-request',
          tokenPrice: '',
          completionTokenPrice: '',
          cacheTokenPrice: '',
          createCacheTokenPrice: '',
          audioTokenPrice: '',
          audioCompletionTokenPrice: '',
          ratio: '',
          completionRatio: '',
          cacheRatio: '',
          createCacheRatio: '',
          audioRatio: '',
          audioCompletionRatio: '',
          contextPricing: { enabled: false, tiers: [] },
        };
      }
    });
  };

  const toggleContextPricing = (enabled) => {
    if (!editingModel) return;
    setEditingModel((prev) => {
      if (!prev) return prev;
      if (enabled) {
        const contextPricing =
          prev.contextPricing?.tiers?.length > 0
            ? { ...prev.contextPricing, enabled: true }
            : createDefaultContextPricing(prev);
        return {
          ...prev,
          pricingMode: 'per-token',
          price: '',
          contextPricing,
        };
      }
      return {
        ...prev,
        contextPricing: { enabled: false, tiers: [] },
        ...buildPriceFieldsFromRatios(prev),
      };
    });
  };

  const updateContextTier = (index, field, value) => {
    setEditingModel((prev) => {
      if (!prev?.contextPricing?.enabled) return prev;
      if (field !== 'name' && hasValue(value) && !/^\d*(\.\d*)?$/.test(value)) {
        showError(t('请输入数字'));
        return prev;
      }
      const tiers = [...(prev.contextPricing.tiers || [])];
      tiers[index] = { ...tiers[index], [field]: value };
      return {
        ...prev,
        contextPricing: { ...prev.contextPricing, tiers },
      };
    });
  };

  const addContextTier = () => {
    setEditingModel((prev) => {
      if (!prev?.contextPricing?.enabled) return prev;
      const tiers = prev.contextPricing.tiers || [];
      const last = tiers[tiers.length - 1] || {};
      const minTokens = hasValue(last.maxTokens) ? last.maxTokens : '1000000';
      const nextTier = {
        ...createDefaultContextPricing(prev).tiers[0],
        name: `≥${minTokens}`,
        minTokens,
        maxTokens: '',
      };
      return {
        ...prev,
        contextPricing: {
          ...prev.contextPricing,
          tiers: [...tiers, nextTier],
        },
      };
    });
  };

  const removeContextTier = (index) => {
    setEditingModel((prev) => {
      if (!prev?.contextPricing?.enabled) return prev;
      const tiers = (prev.contextPricing.tiers || []).filter(
        (_, tierIndex) => tierIndex !== index,
      );
      return {
        ...prev,
        contextPricing: { ...prev.contextPricing, tiers },
      };
    });
  };

  /** 切换可选价格开关 */
  const toggleOptionalField = (field, enabled) => {
    if (!editingModel) return;
    if (enabled) {
      // 启用时，如果为空则给一个默认值
      updateEditingField(field, editingModel[field] || '0');
    } else {
      // 禁用时：同时清除显示价格和底层倍率字段
      setEditingModel((prev) => {
        if (!prev) return prev;
        const updated = { ...prev, [field]: '' };
        // 映射：显示价格字段 → 底层倍率字段
        const fieldToRatioMap = {
          completionTokenPrice: 'completionRatio',
          cacheTokenPrice: 'cacheRatio',
          createCacheTokenPrice: 'createCacheRatio',
          audioTokenPrice: 'audioRatio',
          audioCompletionTokenPrice: 'audioCompletionRatio',
        };
        const ratioField = fieldToRatioMap[field];
        if (ratioField) {
          updated[ratioField] = '';
        }
        updated.hasConflict = buildConflictState(updated);
        return updated;
      });
    }
  };

  /** 实时保存当前编辑内容到 models 列表（切换选中项前自动调用） */
  const commitEditingChanges = () => {
    if (editingModel && selectedModelName) {
      setModels((prev) => saveEditingModelToList(prev));
    }
  };

  /** 安全地切换选中模型 */
  const handleSafeSelectModel = (record) => {
    if (record.name === selectedModelName) return; // 已选中则跳过
    commitEditingChanges();
    handleSelectModel(record);
  };

  // ========== 保存预览数据 ==========
  const previewData = useMemo(() => {
    if (!editingModel || !selectedModelName) return null;

    if (editingModel.pricingMode === 'per-request') {
      return {
        ModelPrice: editingModel.price || '',
      };
    }
    if (editingModel.contextPricing?.enabled) {
      return {
        ContextPricing: `${editingModel.contextPricing.tiers?.length || 0} ${t('档')}`,
      };
    }

    // 按量计费：基于当前编辑状态生成预览
    const synced = syncRatioFieldsFromPrices({ ...editingModel });
    return {
      ModelRatio: synced.ratio || '0',
      CompletionRatio: synced.completionRatio || '0',
      CacheRatio: synced.cacheRatio || '0',
      CreateCacheRatio: synced.createCacheRatio || '0',
      ImageRatio: '0',
      AudioRatio: synced.audioRatio || '0',
      AudioCompletionRatio: synced.audioCompletionRatio || '1',
    };
  }, [editingModel, selectedModelName]);

  // ========== 左侧列表列定义 ==========
  const listColumns = [
    {
      title: '',
      dataIndex: 'name',
      key: 'select',
      width: 40,
      render: (text, record) => (
        <div
          style={{
            width: 4,
            height: '100%',
            background:
              record.name === selectedModelName
                ? 'var(--semi-color-primary)'
                : 'transparent',
            borderRadius: 2,
            marginRight: 8,
          }}
        />
      ),
    },
    {
      title: t('模型名称'),
      dataIndex: 'name',
      key: 'name',
      width: 220,
      render: (text, record) => (
        <span
          style={{
            fontWeight: record.name === selectedModelName ? 600 : 'normal',
            color:
              record.name === selectedModelName
                ? 'var(--semi-color-primary)'
                : 'inherit',
            cursor: 'pointer',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            display: 'block',
          }}
          onClick={() => handleSafeSelectModel(record)}
          title={text}
        >
          {text}
        </span>
      ),
    },
    {
      title: t('计费方式'),
      dataIndex: 'pricingType',
      key: 'pricingType',
      width: 150,
      render: (_, record) => {
        const isPerRequest = hasValue(record?.price);
        const isContextPricing = record?.contextPricing?.enabled;
        return (
          <div className='visual-ratio-billing-tags'>
            <Tag color={isPerRequest ? 'green' : 'blue'} shape='rounded'>
              {isPerRequest ? t('按次计费') : t('按量计费')}
            </Tag>
            {isContextPricing && (
              <Tag
                color='white'
                shape='rounded'
                style={{ background: '#e8f3ff', color: '#0f66d0' }}
              >
                {t('分段计费')}
              </Tag>
            )}
          </div>
        );
      },
    },
    {
      title: t('价格摘要'),
      dataIndex: 'summary',
      key: 'summary',
      width: 140,
      render: (_, record) => (
        <span
          className='visual-ratio-summary'
          title={getModelPricingSummary(record)}
        >
          {getModelPricingSummary(record)}
        </span>
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 60,
      render: (_, record) => (
        <Button
          icon={<IconDelete />}
          type='danger'
          theme='borderless'
          size='small'
          onClick={(e) => {
            e.stopPropagation();
            handleDeleteModel(record.name);
          }}
        />
      ),
    },
  ];

  const renderContextPricingEditor = () => {
    const tiers = editingModel?.contextPricing?.tiers || [];
    const priceFields = [
      { field: 'tokenPrice', label: t('输入价格') },
      { field: 'completionTokenPrice', label: t('补全价格') },
      { field: 'cacheTokenPrice', label: t('缓存读取价格') },
      { field: 'createCacheTokenPrice', label: t('缓存创建价格') },
      { field: 'audioTokenPrice', label: t('音频输入价格') },
      { field: 'audioCompletionTokenPrice', label: t('音频补全价格') },
    ];

    return (
      <div style={{ marginBottom: 24 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 12,
          }}
        >
          <div>
            <div style={{ fontWeight: 600, fontSize: 14 }}>
              {t('分段计费区间')}
            </div>
            <div
              style={{
                color: 'var(--semi-color-text-2)',
                fontSize: 13,
                marginTop: 2,
              }}
            >
              {t('按输入侧上下文 token 命中区间，整次请求使用该区间价格。')}
            </div>
          </div>
          <Button icon={<IconPlus />} size='small' onClick={addContextTier}>
            {t('新增区间')}
          </Button>
        </div>

        {tiers.map((tier, index) => (
          <div
            key={`context-tier-${index}`}
            style={{
              border: '1px solid var(--semi-color-border)',
              borderRadius: 8,
              padding: 12,
              marginBottom: 12,
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 12,
              }}
            >
              <div style={{ fontWeight: 600 }}>
                {tier.name || formatContextRangeSummary(tier)}
              </div>
              <Button
                icon={<IconDelete />}
                type='danger'
                theme='borderless'
                size='small'
                disabled={tiers.length <= 1}
                onClick={() => removeContextTier(index)}
              />
            </div>

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: 8,
                marginBottom: 12,
              }}
            >
              <Input
                value={tier.minTokens ?? ''}
                prefix={t('下限')}
                suffix='tokens'
                onChange={(value) =>
                  updateContextTier(index, 'minTokens', value)
                }
              />
              <Input
                value={tier.maxTokens ?? ''}
                prefix={t('上限')}
                suffix='tokens'
                placeholder={t('留空表示无上限')}
                onChange={(value) =>
                  updateContextTier(index, 'maxTokens', value)
                }
              />
            </div>
            <Input
              value={tier.name ?? ''}
              prefix={t('名称')}
              placeholder={formatContextRangeSummary(tier)}
              style={{ marginBottom: 12 }}
              onChange={(value) => updateContextTier(index, 'name', value)}
            />

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr 1fr',
                gap: 8,
              }}
            >
              {priceFields.map((item) => (
                <Input
                  key={item.field}
                  value={tier[item.field] ?? ''}
                  prefix={item.label}
                  suffix='$/1M'
                  onChange={(value) =>
                    updateContextTier(index, item.field, value)
                  }
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    );
  };

  // ========== 渲染 ==========
  return (
    <div
      className='visual-ratio-editor'
      style={{
        display: 'flex',
        gap: 16,
        height: 'calc(100vh - 280px)',
        minHeight: 400,
      }}
    >
      {/* ====== 左侧：模型列表 ====== */}
      <div
        className='visual-ratio-list-pane'
        style={{
          flex: '0 0 55%',
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        {/* 工具栏 */}
        <div
          className='visual-ratio-toolbar mt-2'
          style={{
            display: 'flex',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 8,
            marginBottom: 12,
            flexShrink: 0,
          }}
        >
          <Button icon={<IconPlus />} type='primary' onClick={handleAddModel}>
            {t('添加模型')}
          </Button>
          <Button icon={<IconSave />} onClick={SubmitData} loading={loading}>
            {t('应用更改')}
          </Button>
          <Input
            prefix={<IconSearch />}
            placeholder={t('搜索模型名称')}
            value={searchText}
            onChange={(value) => {
              setSearchText(value);
              setCurrentPage(1);
            }}
            style={{ width: 200 }}
            showClear
          />
          <Checkbox
            checked={conflictOnly}
            onChange={(e) => {
              setConflictOnly(e.target.checked);
              setCurrentPage(1);
            }}
          >
            {t('仅显示矛盾倍率')}
          </Checkbox>
        </div>

        {/* 模型表格 */}
        <div
          className='visual-ratio-table-wrap'
          style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}
        >
          <Table
            className='visual-ratio-model-table'
            columns={listColumns}
            dataSource={pagedData}
            rowKey='name'
            style={{ width: '100%' }}
            tableLayout='fixed'
            scroll={{ x: '100%', y: '100%' }}
            pagination={{
              currentPage: currentPage,
              pageSize: listPageSize,
              total: filteredModels.length,
              onPageChange: (page) => setCurrentPage(page),
              showTotal: true,
              showSizeChanger: true,
              pageSizeOpts: [50, 100, 200],
              pageSizeOptions: [50, 100, 200],
              onPageSizeChange: (size) => {
                setListPageSize(size);
                setCurrentPage(1);
              },
            }}
            onRow={(record) => ({
              onClick: () => handleSafeSelectModel(record),
              style: {
                cursor: 'pointer',
                background:
                  record.name === selectedModelName
                    ? 'var(--semi-color-fill-0)'
                    : undefined,
              },
            })}
            size='small'
          />
        </div>
      </div>

      {/* ====== 右侧：详情配置面板 ====== */}
      <div
        className='visual-ratio-detail-pane'
        style={{
          flex: '1 1 0',
          borderLeft: '1px solid var(--semi-color-border)',
          paddingLeft: 16,
          overflowY: 'auto',
          overflowX: 'hidden',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {editingModel ? (
          <>
            {/* 标题行：模型名称 + 计费方式标签 */}
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 16,
              }}
            >
              <Input
                value={editingModel.name}
                onChange={(value) => {
                  setEditingModel((prev) =>
                    prev ? { ...prev, name: value } : prev,
                  );
                  // 同步更新 models 列表中的名称和选中状态
                  setModels((prev) =>
                    prev.map((m) =>
                      m.name === selectedModelName ? { ...m, name: value } : m,
                    ),
                  );
                  setSelectedModelName(value);
                }}
                style={{
                  fontWeight: 600,
                  fontSize: 18,
                  border: 'none',
                  padding: 0,
                  boxShadow: 'none',
                }}
              />
              <Tag
                color={
                  editingModel.pricingMode === 'per-request' ? 'blue' : 'blue'
                }
              >
                {editingModel.pricingMode === 'per-request'
                  ? t('按次计费')
                  : editingModel.contextPricing?.enabled
                    ? t('分段计费')
                    : t('按量计费')}
              </Tag>
            </div>

            {/* 计费方式切换 */}
            <div style={{ marginBottom: 4 }}>
              <span
                style={{
                  color: 'var(--semi-color-text-2)',
                  marginBottom: 8,
                  display: 'block',
                }}
              >
                {t('计费方式')}
              </span>
              <RadioGroup
                type='button'
                value={editingModel.pricingMode || 'per-token'}
                onChange={(e) => handlePricingModeChange(e.target.value)}
              >
                <Radio value='per-token'>{t('按量计费')}</Radio>
                <Radio value='per-request'>{t('按次计费')}</Radio>
              </RadioGroup>
            </div>

            <Banner
              type='info'
              description={t(
                '这个界面默认按价格填写，保存时会自动换算回后端需要的倍率 JSON。',
              )}
              style={{ marginBottom: 16 }}
            />

            {/* ====== 按量计费表单 ====== */}
            {editingModel.pricingMode === 'per-token' && (
              <>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: 16,
                    padding: '10px 12px',
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 8,
                    background: 'var(--semi-color-fill-0)',
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600, fontSize: 14 }}>
                      {t('分段计费')}
                    </div>
                    <div
                      style={{
                        color: 'var(--semi-color-text-2)',
                        fontSize: 13,
                        marginTop: 2,
                      }}
                    >
                      {t('开启后按上下文区间设置多组按量价格。')}
                    </div>
                  </div>
                  <Switch
                    checked={Boolean(editingModel.contextPricing?.enabled)}
                    onChange={toggleContextPricing}
                  />
                </div>

                {editingModel.contextPricing?.enabled ? (
                  renderContextPricingEditor()
                ) : (
                  <>
                    {/* 基础价格 */}
                    <div style={{ marginBottom: 24 }}>
                      <div
                        style={{
                          fontWeight: 600,
                          marginBottom: 12,
                          fontSize: 14,
                        }}
                      >
                        {t('基础价格')}
                      </div>

                      <div style={{ marginBottom: 12 }}>
                        <div
                          style={{
                            color: 'var(--semi-color-text-2)',
                            marginBottom: 4,
                            fontSize: 13,
                          }}
                        >
                          {t('输入价格')}
                        </div>
                        <Input
                          value={editingModel.tokenPrice ?? ''}
                          placeholder='0'
                          onChange={(value) =>
                            updateEditingField('tokenPrice', value)
                          }
                          suffix='$/1M tokens'
                        />
                      </div>

                      <div style={{ marginBottom: 12 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('补全价格')}
                          </span>
                          <Switch
                            checked={hasValue(
                              editingModel.completionTokenPrice,
                            )}
                            onChange={(checked) =>
                              toggleOptionalField(
                                'completionTokenPrice',
                                checked,
                              )
                            }
                            size='small'
                          />
                        </div>
                        <Input
                          value={editingModel.completionTokenPrice ?? ''}
                          placeholder='0'
                          onChange={(value) =>
                            updateEditingField('completionTokenPrice', value)
                          }
                          suffix='$/1M tokens'
                          disabled={
                            !hasValue(editingModel.completionTokenPrice)
                          }
                        />
                      </div>

                      <div style={{ marginBottom: 12 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('缓存读取价格')}
                          </span>
                          <Switch
                            checked={hasValue(editingModel.cacheTokenPrice)}
                            onChange={(checked) =>
                              toggleOptionalField('cacheTokenPrice', checked)
                            }
                            size='small'
                          />
                        </div>
                        <Input
                          value={editingModel.cacheTokenPrice ?? ''}
                          placeholder='0'
                          onChange={(value) =>
                            updateEditingField('cacheTokenPrice', value)
                          }
                          suffix='$/1M tokens'
                          disabled={!hasValue(editingModel.cacheTokenPrice)}
                        />
                      </div>

                      <div style={{ marginBottom: 0 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('缓存创建价格')}
                          </span>
                          <Switch
                            checked={hasValue(
                              editingModel.createCacheTokenPrice,
                            )}
                            onChange={(checked) =>
                              toggleOptionalField(
                                'createCacheTokenPrice',
                                checked,
                              )
                            }
                            size='small'
                          />
                        </div>
                        <Input
                          value={editingModel.createCacheTokenPrice ?? ''}
                          placeholder='0'
                          onChange={(value) =>
                            updateEditingField('createCacheTokenPrice', value)
                          }
                          suffix='$/1M tokens'
                          disabled={
                            !hasValue(editingModel.createCacheTokenPrice)
                          }
                        />
                      </div>
                    </div>

                    {/* 扩展价格 */}
                    <div style={{ marginBottom: 24 }}>
                      <div
                        style={{
                          fontWeight: 600,
                          marginBottom: 4,
                          fontSize: 14,
                        }}
                      >
                        {t('扩展价格')}
                      </div>
                      <div
                        style={{
                          color: 'var(--semi-color-text-2)',
                          marginBottom: 12,
                          fontSize: 13,
                        }}
                      >
                        {t('这些价格都是可选项，不填也可以。')}
                      </div>

                      <div style={{ marginBottom: 12 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('图片输入价格')}
                          </span>
                          <Switch size='small' />
                        </div>
                        <Input value='0' disabled suffix='$/1M tokens' />
                      </div>

                      <div style={{ marginBottom: 12 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('音频输入价格')}
                          </span>
                          <Switch
                            checked={hasValue(editingModel.audioTokenPrice)}
                            onChange={(checked) =>
                              toggleOptionalField('audioTokenPrice', checked)
                            }
                            size='small'
                          />
                        </div>
                        <Input
                          value={editingModel.audioTokenPrice ?? ''}
                          placeholder='250'
                          onChange={(value) =>
                            updateEditingField('audioTokenPrice', value)
                          }
                          suffix='$/1M tokens'
                          disabled={!hasValue(editingModel.audioTokenPrice)}
                        />
                      </div>

                      <div style={{ marginBottom: 0 }}>
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                            marginBottom: 4,
                          }}
                        >
                          <span
                            style={{
                              color: 'var(--semi-color-text-2)',
                              fontSize: 13,
                            }}
                          >
                            {t('音频补全价格')}
                          </span>
                          <Switch
                            checked={hasValue(
                              editingModel.audioCompletionTokenPrice,
                            )}
                            onChange={(checked) =>
                              toggleOptionalField(
                                'audioCompletionTokenPrice',
                                checked,
                              )
                            }
                            size='small'
                          />
                        </div>
                        <Input
                          value={editingModel.audioCompletionTokenPrice ?? ''}
                          placeholder='250'
                          onChange={(value) =>
                            updateEditingField(
                              'audioCompletionTokenPrice',
                              value,
                            )
                          }
                          suffix='$/1M tokens'
                          disabled={
                            !hasValue(editingModel.audioCompletionTokenPrice)
                          }
                        />
                      </div>
                    </div>
                  </>
                )}
              </>
            )}

            {/* ====== 按次计费表单 ====== */}
            {editingModel.pricingMode === 'per-request' && (
              <>
                <div style={{ marginBottom: 24 }}>
                  <div
                    style={{
                      fontWeight: 600,
                      marginBottom: 12,
                      fontSize: 14,
                    }}
                  >
                    {t('固定价格')}
                  </div>
                  <Input
                    value={editingModel.price ?? ''}
                    placeholder='19.9'
                    onChange={(value) => updateEditingField('price', value)}
                    suffix='$/次'
                  />
                  <div
                    style={{
                      color: 'var(--semi-color-text-2)',
                      marginTop: 8,
                      fontSize: 13,
                    }}
                  >
                    {t('适合 MJ / 任务类等按次收费模型。')}
                  </div>
                </div>
              </>
            )}

            {/* ====== 保存预览 ====== */}
            {previewData && (
              <div
                style={{
                  backgroundColor: 'var(--semi-color-fill-0)',
                  borderRadius: 8,
                  padding: 16,
                  marginTop: 'auto',
                }}
              >
                <div
                  style={{
                    fontWeight: 600,
                    marginBottom: 4,
                    fontSize: 14,
                  }}
                >
                  {t('保存预览')}
                </div>
                <div
                  style={{
                    color: 'var(--semi-color-text-2)',
                    marginBottom: 12,
                    fontSize: 13,
                  }}
                >
                  {t(
                    '下面展示这个模型保存后会写入哪些后端字段，便于和原始 JSON 编辑器保持一致。',
                  )}
                </div>
                {Object.entries(previewData).map(([key, val]) => (
                  <div
                    key={key}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      padding: '6px 0',
                      borderBottom: '1px solid var(--semi-color-border)',
                      fontSize: 13,
                    }}
                  >
                    <span
                      style={{
                        fontFamily: 'monospace',
                        fontWeight: 500,
                      }}
                    >
                      {key}
                    </span>
                    <span
                      style={{
                        color: 'var(--semi-color-text-2)',
                        fontFamily: 'monospace',
                      }}
                    >
                      {val}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </>
        ) : (
          /* 空状态提示 */
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: 'var(--semi-color-text-3)',
              fontSize: 14,
            }}
          >
            <div style={{ textAlign: 'center' }}>
              <IconSearch
                style={{ fontSize: 48, marginBottom: 16, opacity: 0.3 }}
              />
              <div>{t('请从左侧选择一个模型进行配置')}</div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// --- 表格列辅助函数：倍率 ↔ 显示价格的转换 ---

// 补全倍率 → 显示价格：completionRatio 是相对于 ratio 的倍数，所以价格 = ratio * 2 * completionRatio
const completionRatioToDisplayPrice = (completionRatio, baseRatio) => {
  if (!hasValue(completionRatio) || !hasValue(baseRatio)) return '';
  const cr = Number(completionRatio);
  const br = Number(baseRatio);
  if (!Number.isFinite(cr) || !Number.isFinite(br)) return '';
  return formatNumberValue(br * 2 * cr);
};

// 显示价格 → 补全倍率
const displayPriceToCompletionRatio = (price, baseRatio) => {
  if (!hasValue(price) || !hasValue(baseRatio)) return '';
  const p = Number(price);
  const br = Number(baseRatio);
  if (!Number.isFinite(p) || !Number.isFinite(br) || br === 0) return '';
  return formatNumberValue(p / (br * 2));
};

// 相对倍率（缓存/图片/音频输入等）→ 显示价格
const relativeRatioToDisplayPrice = (relativeRatio, baseRatio) => {
  if (!hasValue(relativeRatio) || !hasValue(baseRatio)) return '';
  const rr = Number(relativeRatio);
  const br = Number(baseRatio);
  if (!Number.isFinite(rr) || !Number.isFinite(br)) return '';
  return formatNumberValue(br * 2 * rr);
};

// 显示价格 → 相对倍率
const displayPriceToRelativeRatio = (price, baseRatio) => {
  if (!hasValue(price) || !hasValue(baseRatio)) return '';
  const p = Number(price);
  const br = Number(baseRatio);
  if (!Number.isFinite(p) || !Number.isFinite(br) || br === 0) return '';
  return formatNumberValue(p / (br * 2));
};

// 音频补全倍率 → 显示价格：audioCompletionRatio 相对于 audioRatio，价格 = ratio * 2 * audioRatio * audioCompletionRatio
const audioCompletionRatioToDisplayPrice = (
  audioCompletionRatio,
  baseRatio,
  audioRatio,
) => {
  if (
    !hasValue(audioCompletionRatio) ||
    !hasValue(baseRatio) ||
    !hasValue(audioRatio)
  )
    return '';
  const acr = Number(audioCompletionRatio);
  const br = Number(baseRatio);
  const ar = Number(audioRatio);
  if (!Number.isFinite(acr) || !Number.isFinite(br) || !Number.isFinite(ar))
    return '';
  return formatNumberValue(br * 2 * ar * acr);
};

// 显示价格 → 音频补全倍率
const displayPriceToAudioCompletionRatio = (price, baseRatio, audioRatio) => {
  if (!hasValue(price) || !hasValue(baseRatio) || !hasValue(audioRatio))
    return '';
  const p = Number(price);
  const br = Number(baseRatio);
  const ar = Number(audioRatio);
  if (
    !Number.isFinite(p) ||
    !Number.isFinite(br) ||
    !Number.isFinite(ar) ||
    br * ar === 0
  )
    return '';
  return formatNumberValue(p / (br * 2 * ar));
};
