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

import React, { useEffect, useState, useRef } from 'react';
import {
  Table,
  Button,
  Input,
  Modal,
  Form,
  Space,
  RadioGroup,
  Radio,
  Checkbox,
  Tag,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconPlus,
  IconSearch,
  IconSave,
  IconEdit,
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
  hasConflict: false,
  isUnset: true,
});

const hasAnyPricingConfig = (model) =>
  MODEL_PRICING_FIELDS.some((field) => hasValue(model?.[field]));

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

// 从倍率字段构建价格显示值（用于弹窗编辑时的价格字段回显）
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

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [models, setModels] = useState([]);
  const [enabledModels, setEnabledModels] = useState([]);
  const [visible, setVisible] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  const [currentModel, setCurrentModel] = useState(null);
  const [searchText, setSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [pricingMode, setPricingMode] = useState('per-token'); // 'per-token' or 'per-request'
  const [conflictOnly, setConflictOnly] = useState(false);
  const [editingValues, setEditingValues] = useState({});
  const formRef = useRef(null);
  const pageSize = 10;

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

      const configuredModelNames = new Set([
        ...Object.keys(modelPrice),
        ...Object.keys(modelRatio),
        ...Object.keys(completionRatio),
        ...Object.keys(cacheRatio),
        ...Object.keys(createCacheRatio),
        ...Object.keys(audioRatio),
        ...Object.keys(audioCompletionRatio),
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

          return {
            name,
            price: normalizeNumericEditableValue(price),
            ratio: normalizeNumericEditableValue(ratio),
            completionRatio: normalizeNumericEditableValue(comp),
            cacheRatio: normalizeNumericEditableValue(cache),
            createCacheRatio: normalizeNumericEditableValue(createCache),
            audioRatio: normalizeNumericEditableValue(audio),
            audioCompletionRatio: normalizeNumericEditableValue(audioComp),
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

  const getPagedData = (data, currentPage, pageSize) => {
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    return data.slice(start, end);
  };

  const filteredModels = models.filter((model) => {
    const keywordMatch = searchText ? model.name.includes(searchText) : true;
    const conflictMatch = conflictOnly ? model.hasConflict : true;
    return keywordMatch && conflictMatch;
  });

  const pagedData = getPagedData(filteredModels, currentPage, pageSize);

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
    };
    try {
      models.forEach((model) => {
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

  // 表格列：统一以价格（$/1M tokens）展示，内部仍存倍率
  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'name',
      key: 'name',
      width: 220,
      render: (text, record) => (
        <span>
          {text}
          {record.hasConflict && (
            <Tag color='red' shape='circle' className='ml-2'>
              {t('矛盾')}
            </Tag>
          )}
        </span>
      ),
    },
    {
      title: t('模型固定价格'),
      dataIndex: 'price',
      key: 'price',
      width: 100,
      render: (text, record) => (
        <Input
          value={editingValues[`${record.name}:price`] ?? text}
          placeholder={t('按量计费')}
          onChange={(value) =>
            handleInlineInputChange(record.name, 'price', value, (current) =>
              formatNumberValue(Number(current)),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'price',
              editingValues[`${record.name}:price`] ?? text,
              (current) => formatNumberValue(Number(current)),
            )
          }
        />
      ),
    },
    {
      title: `${t('输入价格')} ($/1M)`,
      dataIndex: 'ratio',
      key: 'ratio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:ratio`] ?? ratioToDisplayPrice(text)
          }
          placeholder={record.price !== '' ? '-' : t('默认补全倍率')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'ratio',
              value,
              displayPriceToRatio,
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'ratio',
              editingValues[`${record.name}:ratio`] ??
                ratioToDisplayPrice(text),
              displayPriceToRatio,
            )
          }
        />
      ),
    },
    {
      title: `${t('输出价格')} ($/1M)`,
      dataIndex: 'completionRatio',
      key: 'completionRatio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:completionRatio`] ??
            completionRatioToDisplayPrice(text, record.ratio)
          }
          placeholder={record.price !== '' ? '-' : t('默认补全倍率')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'completionRatio',
              value,
              (current) => displayPriceToCompletionRatio(current, record.ratio),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'completionRatio',
              editingValues[`${record.name}:completionRatio`] ??
                completionRatioToDisplayPrice(text, record.ratio),
              (current) => displayPriceToCompletionRatio(current, record.ratio),
            )
          }
        />
      ),
    },
    {
      title: `${t('缓存创建价格')} ($/1M)`,
      dataIndex: 'createCacheRatio',
      key: 'createCacheRatio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:createCacheRatio`] ??
            relativeRatioToDisplayPrice(text, record.ratio)
          }
          placeholder={t('缓存创建')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'createCacheRatio',
              value,
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'createCacheRatio',
              editingValues[`${record.name}:createCacheRatio`] ??
                relativeRatioToDisplayPrice(text, record.ratio),
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
        />
      ),
    },
    {
      title: `${t('缓存读取价格')} ($/1M)`,
      dataIndex: 'cacheRatio',
      key: 'cacheRatio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:cacheRatio`] ??
            relativeRatioToDisplayPrice(text, record.ratio)
          }
          placeholder={t('缓存读取')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'cacheRatio',
              value,
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'cacheRatio',
              editingValues[`${record.name}:cacheRatio`] ??
                relativeRatioToDisplayPrice(text, record.ratio),
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
        />
      ),
    },
    {
      title: `${t('音频输入价格')} ($/1M)`,
      dataIndex: 'audioRatio',
      key: 'audioRatio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:audioRatio`] ??
            relativeRatioToDisplayPrice(text, record.ratio)
          }
          placeholder={t('音频输入')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'audioRatio',
              value,
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'audioRatio',
              editingValues[`${record.name}:audioRatio`] ??
                relativeRatioToDisplayPrice(text, record.ratio),
              (current) => displayPriceToRelativeRatio(current, record.ratio),
            )
          }
        />
      ),
    },
    {
      title: `${t('音频输出价格')} ($/1M)`,
      dataIndex: 'audioCompletionRatio',
      key: 'audioCompletionRatio',
      render: (text, record) => (
        <Input
          value={
            editingValues[`${record.name}:audioCompletionRatio`] ??
            audioCompletionRatioToDisplayPrice(
              text,
              record.ratio,
              record.audioRatio,
            )
          }
          placeholder={t('音频输出')}
          disabled={record.price !== ''}
          onChange={(value) =>
            handleInlineInputChange(
              record.name,
              'audioCompletionRatio',
              value,
              (current) =>
                displayPriceToAudioCompletionRatio(
                  current,
                  record.ratio,
                  record.audioRatio,
                ),
            )
          }
          onBlur={() =>
            handleInlineInputBlur(
              record.name,
              'audioCompletionRatio',
              editingValues[`${record.name}:audioCompletionRatio`] ??
                audioCompletionRatioToDisplayPrice(
                  text,
                  record.ratio,
                  record.audioRatio,
                ),
              (current) =>
                displayPriceToAudioCompletionRatio(
                  current,
                  record.ratio,
                  record.audioRatio,
                ),
            )
          }
        />
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            type='primary'
            icon={<IconEdit />}
            onClick={() => editModel(record)}
          ></Button>
          <Button
            icon={<IconDelete />}
            type='danger'
            onClick={() => deleteModel(record.name)}
          />
        </Space>
      ),
    },
  ];

  const handleInlineInputChange = (
    name,
    field,
    rawValue,
    converter = (current) => current,
  ) => {
    if (rawValue !== '' && !isEditableNumberInput(rawValue)) {
      showError('请输入数字');
      return;
    }

    const cacheKey = `${name}:${field}`;
    setEditingValues((prev) => ({ ...prev, [cacheKey]: rawValue }));

    if (shouldDeferNumericSync(rawValue)) {
      return;
    }

    const normalizedValue = rawValue === '' ? '' : converter(rawValue);
    setModels((prev) =>
      prev.map((model) => {
        if (model.name !== name) return model;
        const updated = { ...model, [field]: normalizedValue };
        updated.hasConflict = buildConflictState(updated);
        return updated;
      }),
    );
  };

  const handleInlineInputBlur = (
    name,
    field,
    value,
    converter = (current) => current,
  ) => {
    const cacheKey = `${name}:${field}`;
    const normalizedValue = hasValue(value) ? converter(value) : '';

    setModels((prev) =>
      prev.map((model) => {
        if (model.name !== name) return model;
        const updated = { ...model, [field]: normalizedValue };
        updated.hasConflict = buildConflictState(updated);
        return updated;
      }),
    );

    setEditingValues((prev) => {
      const next = { ...prev };
      delete next[cacheKey];
      return next;
    });
  };

  const deleteModel = (name) => {
    setEditingValues((prev) => {
      const next = { ...prev };
      Object.keys(next).forEach((key) => {
        if (key.startsWith(`${name}:`)) {
          delete next[key];
        }
      });
      return next;
    });
    setModels((prev) => prev.filter((model) => model.name !== name));
  };

  // 弹窗中价格输入的 onChange 处理（输入价格 → 自动转为倍率存入 currentModel）
  const handleTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      tokenPrice: value,
      ratio:
        hasValue(value) && !isNaN(value)
          ? formatNumberValue(priceToRatio(Number(value)))
          : '',
    };
    setCurrentModel(syncRatioFieldsFromPrices(newState));
  };

  const handleCompletionTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      completionTokenPrice: value,
      completionRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(
              Number(value),
              Number(currentModel.tokenPrice),
            )
          : '',
    };
    setCurrentModel(newState);
  };

  const handleCacheTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      cacheTokenPrice: value,
      cacheRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(
              Number(value),
              Number(currentModel.tokenPrice),
            )
          : '',
    };
    setCurrentModel(newState);
  };

  const handleCreateCacheTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      createCacheTokenPrice: value,
      createCacheRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(
              Number(value),
              Number(currentModel.tokenPrice),
            )
          : '',
    };
    setCurrentModel(newState);
  };

  const handleAudioTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      audioTokenPrice: value,
      audioRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(
              Number(value),
              Number(currentModel.tokenPrice),
            )
          : '',
    };
    setCurrentModel(syncRatioFieldsFromPrices(newState));
  };

  const handleAudioCompletionTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      audioCompletionTokenPrice: value,
      audioCompletionRatio:
        hasValue(value) && hasValue(currentModel?.audioTokenPrice)
          ? calculateRelativeRatio(
              Number(value),
              Number(currentModel.audioTokenPrice),
            )
          : '',
    };
    setCurrentModel(newState);
  };

  const addOrUpdateModel = (values) => {
    const existingModelIndex = models.findIndex(
      (model) => model.name === values.name,
    );

    if (existingModelIndex >= 0) {
      setModels((prev) =>
        sortModels(
          prev.map((model, index) => {
            if (index !== existingModelIndex) return model;
            const updated = {
              ...model,
              name: values.name,
              price: normalizeNumericEditableValue(values.price),
              ratio: normalizeNumericEditableValue(values.ratio),
              completionRatio: normalizeNumericEditableValue(
                values.completionRatio,
              ),
              cacheRatio: normalizeNumericEditableValue(values.cacheRatio),
              createCacheRatio: normalizeNumericEditableValue(
                values.createCacheRatio,
              ),
              audioRatio: normalizeNumericEditableValue(values.audioRatio),
              audioCompletionRatio: normalizeNumericEditableValue(
                values.audioCompletionRatio,
              ),
            };
            updated.hasConflict = buildConflictState(updated);
            return updated;
          }),
        ),
      );
      setVisible(false);
      showSuccess(t('更新成功'));
    } else {
      if (models.some((model) => model.name === values.name)) {
        showError(t('模型名称已存在'));
        return;
      }

      setModels((prev) => {
        const newModel = {
          name: values.name,
          price: normalizeNumericEditableValue(values.price),
          ratio: normalizeNumericEditableValue(values.ratio),
          completionRatio: normalizeNumericEditableValue(
            values.completionRatio,
          ),
          cacheRatio: normalizeNumericEditableValue(values.cacheRatio),
          createCacheRatio: normalizeNumericEditableValue(
            values.createCacheRatio,
          ),
          audioRatio: normalizeNumericEditableValue(values.audioRatio),
          audioCompletionRatio: normalizeNumericEditableValue(
            values.audioCompletionRatio,
          ),
          isUnset: false,
        };
        newModel.hasConflict = buildConflictState(newModel);
        return sortModels([newModel, ...prev]);
      });
      setVisible(false);
      showSuccess(t('添加成功'));
    }
  };

  const resetModalState = () => {
    setCurrentModel(null);
    setPricingMode('per-token');
    setIsEditMode(false);
  };

  const editModel = (record) => {
    setIsEditMode(true);

    let initialPricingMode = 'per-token';
    if (record.price !== '') {
      initialPricingMode = 'per-request';
    }

    setPricingMode(initialPricingMode);

    // 从倍率构建价格字段用于弹窗回显
    const modelCopy = {
      ...record,
      ...buildPriceFieldsFromRatios(record),
    };

    setCurrentModel(modelCopy);
    setVisible(true);

    setTimeout(() => {
      if (formRef.current) {
        const formValues = { name: modelCopy.name };

        if (initialPricingMode === 'per-request') {
          formValues.priceInput = modelCopy.price;
        } else {
          // 按量计费统一以价格展示
          formValues.modelTokenPrice = modelCopy.tokenPrice;
          formValues.completionTokenPrice = modelCopy.completionTokenPrice;
          formValues.cacheTokenPrice = modelCopy.cacheTokenPrice;
          formValues.createCacheTokenPrice = modelCopy.createCacheTokenPrice;
          formValues.audioTokenPrice = modelCopy.audioTokenPrice;
          formValues.audioCompletionTokenPrice =
            modelCopy.audioCompletionTokenPrice;
        }

        formRef.current.setValues(formValues);
      }
    }, 0);
  };

  return (
    <>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space className='mt-2'>
          <Button
            icon={<IconPlus />}
            onClick={() => {
              resetModalState();
              setVisible(true);
            }}
          >
            {t('添加模型')}
          </Button>
          <Button type='primary' icon={<IconSave />} onClick={SubmitData}>
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
        </Space>
        <Table
          columns={columns}
          dataSource={pagedData}
          rowKey='name'
          pagination={{
            currentPage: currentPage,
            pageSize: pageSize,
            total: filteredModels.length,
            onPageChange: (page) => setCurrentPage(page),
            showTotal: true,
            showSizeChanger: false,
          }}
        />
      </Space>

      <Modal
        title={isEditMode ? t('编辑模型') : t('添加模型')}
        visible={visible}
        onCancel={() => {
          resetModalState();
          setVisible(false);
        }}
        onOk={() => {
          if (currentModel) {
            let valuesToSave = { ...currentModel };

            // 按量计费：将价格字段同步回倍率字段
            if (pricingMode === 'per-token') {
              valuesToSave = syncRatioFieldsFromPrices(valuesToSave);
              valuesToSave.price = '';
            } else {
              // 按次计费：清除按量字段
              valuesToSave = clearPerTokenPricing(valuesToSave);
            }

            addOrUpdateModel(valuesToSave);
          }
        }}
      >
        <Form getFormApi={(api) => (formRef.current = api)}>
          <Form.Input
            field='name'
            label={t('模型名称')}
            placeholder='strawberry'
            required
            disabled={isEditMode}
            onChange={(value) =>
              setCurrentModel((prev) => ({ ...prev, name: value }))
            }
          />

          <Form.Section text={t('定价模式')}>
            <div style={{ marginBottom: '16px' }}>
              <RadioGroup
                type='button'
                value={pricingMode}
                onChange={(e) => {
                  const newMode = e.target.value;
                  setPricingMode(newMode);

                  if (currentModel) {
                    const updatedModel =
                      newMode === 'per-token'
                        ? {
                            ...currentModel,
                            ...buildPriceFieldsFromRatios(currentModel),
                          }
                        : { ...currentModel };

                    if (formRef.current) {
                      const formValues = { name: updatedModel.name };

                      if (newMode === 'per-request') {
                        formValues.priceInput = normalizeEditableValue(
                          updatedModel.price,
                        );
                      } else {
                        formValues.modelTokenPrice = normalizeEditableValue(
                          updatedModel.tokenPrice,
                        );
                        formValues.completionTokenPrice =
                          normalizeEditableValue(
                            updatedModel.completionTokenPrice,
                          );
                        formValues.cacheTokenPrice = normalizeEditableValue(
                          updatedModel.cacheTokenPrice,
                        );
                        formValues.createCacheTokenPrice =
                          normalizeEditableValue(
                            updatedModel.createCacheTokenPrice,
                          );
                        formValues.audioTokenPrice = normalizeEditableValue(
                          updatedModel.audioTokenPrice,
                        );
                        formValues.audioCompletionTokenPrice =
                          normalizeEditableValue(
                            updatedModel.audioCompletionTokenPrice,
                          );
                      }

                      formRef.current.setValues(formValues);
                    }

                    setCurrentModel(updatedModel);
                  }
                }}
              >
                <Radio value='per-token'>{t('按量计费')}</Radio>
                <Radio value='per-request'>{t('按次计费')}</Radio>
              </RadioGroup>
            </div>
          </Form.Section>

          {/* 按量计费：统一以价格（$/1M tokens）输入，保存时自动转倍率 */}
          {pricingMode === 'per-token' && (
            <>
              <Form.Input
                field='modelTokenPrice'
                label={t('输入价格')}
                onChange={(value) => {
                  handleTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(currentModel?.tokenPrice)}
                suffix={t('$/1M tokens')}
              />
              <Form.Input
                field='completionTokenPrice'
                label={t('输出价格')}
                onChange={(value) => {
                  handleCompletionTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(
                  currentModel?.completionTokenPrice,
                )}
                suffix={t('$/1M tokens')}
              />
              <Form.Input
                field='createCacheTokenPrice'
                label={t('缓存创建价格')}
                onChange={(value) => {
                  handleCreateCacheTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(
                  currentModel?.createCacheTokenPrice,
                )}
                suffix={t('$/1M tokens')}
              />
              <Form.Input
                field='cacheTokenPrice'
                label={t('缓存读取价格')}
                onChange={(value) => {
                  handleCacheTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(
                  currentModel?.cacheTokenPrice,
                )}
                suffix={t('$/1M tokens')}
              />
              <Form.Input
                field='audioTokenPrice'
                label={t('音频输入价格')}
                onChange={(value) => {
                  handleAudioTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(
                  currentModel?.audioTokenPrice,
                )}
                suffix={t('$/1M tokens')}
              />
              <Form.Input
                field='audioCompletionTokenPrice'
                label={t('音频输出价格')}
                onChange={(value) => {
                  handleAudioCompletionTokenPriceChange(value);
                }}
                initValue={normalizeEditableValue(
                  currentModel?.audioCompletionTokenPrice,
                )}
                suffix={t('$/1M tokens')}
              />
            </>
          )}

          {pricingMode === 'per-request' && (
            <Form.Input
              field='priceInput'
              label={t('固定价格(每次)')}
              placeholder={t('输入每次价格')}
              onChange={(value) =>
                setCurrentModel((prev) => ({
                  ...(prev || {}),
                  price: value,
                }))
              }
              initValue={normalizeEditableValue(currentModel?.price)}
            />
          )}
        </Form>
      </Modal>
    </>
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
