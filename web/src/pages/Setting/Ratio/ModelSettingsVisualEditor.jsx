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

const PER_TOKEN_RATIO_FIELDS = [
  'ratio',
  'completionRatio',
  'cacheRatio',
  'createCacheRatio',
  'imageRatio',
  'audioRatio',
  'audioCompletionRatio',
];

// 统一处理可视化计费编辑里的倍率/价格换算，避免不同入口出现计算口径不一致。
const hasValue = (value) => value !== '' && value !== undefined && value !== null;

const normalizeEditableValue = (value) => (hasValue(value) ? `${value}` : '');

const parseInputNumber = (value) => {
  if (!hasValue(value)) {
    return null;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

const calculateTokenPriceFromRatio = (ratio) => ratio * 2;

const calculateRelativeRatio = (targetPrice, basePrice) => {
  if (
    !Number.isFinite(targetPrice) ||
    !Number.isFinite(basePrice) ||
    basePrice <= 0
  ) {
    return '';
  }
  return (targetPrice / basePrice).toString();
};

const hasPerTokenPricing = (model) =>
  PER_TOKEN_RATIO_FIELDS.some((field) => hasValue(model?.[field]));

const buildConflictState = (model) =>
  hasValue(model?.price) && hasPerTokenPricing(model);

const buildTokenPriceFieldsFromRatios = (model) => {
  const ratio = parseInputNumber(model?.ratio);
  const completionRatio = parseInputNumber(model?.completionRatio);
  const cacheRatio = parseInputNumber(model?.cacheRatio);
  const createCacheRatio = parseInputNumber(model?.createCacheRatio);
  const imageRatio = parseInputNumber(model?.imageRatio);
  const audioRatio = parseInputNumber(model?.audioRatio);
  const audioCompletionRatio = parseInputNumber(model?.audioCompletionRatio);

  const tokenPrice = ratio !== null ? calculateTokenPriceFromRatio(ratio) : null;
  const audioTokenPrice =
    tokenPrice !== null && audioRatio !== null ? tokenPrice * audioRatio : null;

  return {
    tokenPrice: tokenPrice !== null ? tokenPrice.toString() : '',
    completionTokenPrice:
      tokenPrice !== null && completionRatio !== null
        ? (tokenPrice * completionRatio).toString()
        : '',
    cacheTokenPrice:
      tokenPrice !== null && cacheRatio !== null
        ? (tokenPrice * cacheRatio).toString()
        : '',
    createCacheTokenPrice:
      tokenPrice !== null && createCacheRatio !== null
        ? (tokenPrice * createCacheRatio).toString()
        : '',
    audioTokenPrice: audioTokenPrice !== null ? audioTokenPrice.toString() : '',
    audioCompletionTokenPrice:
      audioTokenPrice !== null && audioCompletionRatio !== null
        ? (audioTokenPrice * audioCompletionRatio).toString()
        : '',
    imageTokenPrice:
      tokenPrice !== null && imageRatio !== null
        ? (tokenPrice * imageRatio).toString()
        : '',
  };
};

const syncRatioFieldsFromTokenPrices = (model) => {
  const tokenPrice = parseInputNumber(model?.tokenPrice);
  const completionTokenPrice = parseInputNumber(model?.completionTokenPrice);
  const cacheTokenPrice = parseInputNumber(model?.cacheTokenPrice);
  const createCacheTokenPrice = parseInputNumber(model?.createCacheTokenPrice);
  const audioTokenPrice = parseInputNumber(model?.audioTokenPrice);
  const audioCompletionTokenPrice = parseInputNumber(
    model?.audioCompletionTokenPrice,
  );
  const imageTokenPrice = parseInputNumber(model?.imageTokenPrice);

  const updatedModel = {
    ...(model || {}),
    ratio:
      tokenPrice !== null
        ? (tokenPrice / 2).toString()
        : hasValue(model?.tokenPrice)
          ? ''
          : normalizeEditableValue(model?.ratio),
    completionRatio: hasValue(model?.completionTokenPrice)
      ? calculateRelativeRatio(completionTokenPrice, tokenPrice)
      : normalizeEditableValue(model?.completionRatio),
    cacheRatio: hasValue(model?.cacheTokenPrice)
      ? calculateRelativeRatio(cacheTokenPrice, tokenPrice)
      : normalizeEditableValue(model?.cacheRatio),
    createCacheRatio: hasValue(model?.createCacheTokenPrice)
      ? calculateRelativeRatio(createCacheTokenPrice, tokenPrice)
      : normalizeEditableValue(model?.createCacheRatio),
    audioRatio: hasValue(model?.audioTokenPrice)
      ? calculateRelativeRatio(audioTokenPrice, tokenPrice)
      : normalizeEditableValue(model?.audioRatio),
    audioCompletionRatio: hasValue(model?.audioCompletionTokenPrice)
      ? calculateRelativeRatio(audioCompletionTokenPrice, audioTokenPrice)
      : normalizeEditableValue(model?.audioCompletionRatio),
    imageRatio: hasValue(model?.imageTokenPrice)
      ? calculateRelativeRatio(imageTokenPrice, tokenPrice)
      : normalizeEditableValue(model?.imageRatio),
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
  imageRatio: '',
  audioRatio: '',
  audioCompletionRatio: '',
  hasConflict: false,
});

export default function ModelSettingsVisualEditor(props) {
  const { t } = useTranslation();
  const [models, setModels] = useState([]);
  const [visible, setVisible] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  const [currentModel, setCurrentModel] = useState(null);
  const [searchText, setSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [pricingMode, setPricingMode] = useState('per-token'); // 'per-token' or 'per-request'
  const [pricingSubMode, setPricingSubMode] = useState('ratio'); // 'ratio' or 'token-price'
  const [conflictOnly, setConflictOnly] = useState(false);
  const formRef = useRef(null);
  const pageSize = 10;

  useEffect(() => {
    try {
      const modelPrice = JSON.parse(props.options.ModelPrice || '{}');
      const modelRatio = JSON.parse(props.options.ModelRatio || '{}');
      const completionRatio = JSON.parse(props.options.CompletionRatio || '{}');
      const cacheRatio = JSON.parse(props.options.CacheRatio || '{}');
      const createCacheRatio = JSON.parse(props.options.CreateCacheRatio || '{}');
      const imageRatio = JSON.parse(props.options.ImageRatio || '{}');
      const audioRatio = JSON.parse(props.options.AudioRatio || '{}');
      const audioCompletionRatio = JSON.parse(props.options.AudioCompletionRatio || '{}');

      // 合并所有模型名称
      const modelNames = new Set([
        ...Object.keys(modelPrice),
        ...Object.keys(modelRatio),
        ...Object.keys(completionRatio),
        ...Object.keys(cacheRatio),
        ...Object.keys(createCacheRatio),
        ...Object.keys(imageRatio),
        ...Object.keys(audioRatio),
        ...Object.keys(audioCompletionRatio),
      ]);

      const modelData = Array.from(modelNames).map((name) => {
        const price = modelPrice[name] === undefined ? '' : modelPrice[name];
        const ratio = modelRatio[name] === undefined ? '' : modelRatio[name];
        const comp =
          completionRatio[name] === undefined ? '' : completionRatio[name];
        const cache = cacheRatio[name] === undefined ? '' : cacheRatio[name];
        const createCache = createCacheRatio[name] === undefined ? '' : createCacheRatio[name];
        const image = imageRatio[name] === undefined ? '' : imageRatio[name];
        const audio = audioRatio[name] === undefined ? '' : audioRatio[name];
        const audioComp = audioCompletionRatio[name] === undefined ? '' : audioCompletionRatio[name];

        return {
          name,
          price,
          ratio,
          completionRatio: comp,
          cacheRatio: cache,
          createCacheRatio: createCache,
          imageRatio: image,
          audioRatio: audio,
          audioCompletionRatio: audioComp,
          hasConflict: buildConflictState({
            price,
            ratio,
            completionRatio: comp,
            cacheRatio: cache,
            createCacheRatio: createCache,
            imageRatio: image,
            audioRatio: audio,
            audioCompletionRatio: audioComp,
          }),
        };
      });

      setModels(modelData);
    } catch (error) {
      console.error('JSON解析错误:', error);
    }
  }, [props.options]);

  // 首先声明分页相关的工具函数
  const getPagedData = (data, currentPage, pageSize) => {
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    return data.slice(start, end);
  };

  // 在 return 语句之前，先处理过滤和分页逻辑
  const filteredModels = models.filter((model) => {
    const keywordMatch = searchText ? model.name.includes(searchText) : true;
    const conflictMatch = conflictOnly ? model.hasConflict : true;
    return keywordMatch && conflictMatch;
  });

  // 然后基于过滤后的数据计算分页数据
  const pagedData = getPagedData(filteredModels, currentPage, pageSize);

  const SubmitData = async () => {
    setLoading(true);
    const output = {
      ModelPrice: {},
      ModelRatio: {},
      CompletionRatio: {},
      CacheRatio: {},
      CreateCacheRatio: {},
      ImageRatio: {},
      AudioRatio: {},
      AudioCompletionRatio: {},
    };
    let currentConvertModelName = '';

    try {
      // 数据转换
      models.forEach((model) => {
        currentConvertModelName = model.name;
        if (model.price !== '') {
          output.ModelPrice[model.name] = parseFloat(model.price);
        } else {
          if (model.ratio !== '')
            output.ModelRatio[model.name] = parseFloat(model.ratio);
          if (model.completionRatio !== '')
            output.CompletionRatio[model.name] = parseFloat(
              model.completionRatio,
            );
        }
        if (model.cacheRatio !== '')
          output.CacheRatio[model.name] = parseFloat(model.cacheRatio);
        if (model.createCacheRatio !== '')
          output.CreateCacheRatio[model.name] = parseFloat(model.createCacheRatio);
        if (model.imageRatio !== '')
          output.ImageRatio[model.name] = parseFloat(model.imageRatio);
        if (model.audioRatio !== '')
          output.AudioRatio[model.name] = parseFloat(model.audioRatio);
        if (model.audioCompletionRatio !== '')
          output.AudioCompletionRatio[model.name] = parseFloat(model.audioCompletionRatio);
      });

      // 准备API请求数组
      const finalOutput = {
        ModelPrice: JSON.stringify(output.ModelPrice, null, 2),
        ModelRatio: JSON.stringify(output.ModelRatio, null, 2),
        CompletionRatio: JSON.stringify(output.CompletionRatio, null, 2),
        CacheRatio: JSON.stringify(output.CacheRatio, null, 2),
        CreateCacheRatio: JSON.stringify(output.CreateCacheRatio, null, 2),
        ImageRatio: JSON.stringify(output.ImageRatio, null, 2),
        AudioRatio: JSON.stringify(output.AudioRatio, null, 2),
        AudioCompletionRatio: JSON.stringify(output.AudioCompletionRatio, null, 2),
      };

      const requestQueue = Object.entries(finalOutput).map(([key, value]) => {
        return API.put('/api/option/', {
          key,
          value,
        });
      });

      // 批量处理请求
      const results = await Promise.all(requestQueue);

      // 验证结果
      if (requestQueue.length === 1) {
        if (results.includes(undefined)) return;
      } else if (requestQueue.length > 1) {
        if (results.includes(undefined)) {
          return showError('部分保存失败，请重试');
        }
      }

      // 检查每个请求的结果
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

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'name',
      key: 'name',
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
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('按量计费')}
          onChange={(value) => updateModel(record.name, 'price', value)}
        />
      ),
    },
    {
      title: t('模型倍率'),
      dataIndex: 'ratio',
      key: 'ratio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={record.price !== '' ? t('模型倍率') : t('默认补全倍率')}
          disabled={record.price !== ''}
          onChange={(value) => updateModel(record.name, 'ratio', value)}
        />
      ),
    },
    {
      title: t('补全倍率'),
      dataIndex: 'completionRatio',
      key: 'completionRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={record.price !== '' ? t('补全倍率') : t('默认补全倍率')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'completionRatio', value)
          }
        />
      ),
    },
    {
      title: t('缓存倍率'),
      dataIndex: 'cacheRatio',
      key: 'cacheRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('缓存读取')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'cacheRatio', value)
          }
        />
      ),
    },
    {
      title: t('缓存创建倍率'),
      dataIndex: 'createCacheRatio',
      key: 'createCacheRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('缓存创建')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'createCacheRatio', value)
          }
        />
      ),
    },
    {
      title: t('音频倍率'),
      dataIndex: 'audioRatio',
      key: 'audioRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('音频输入')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'audioRatio', value)
          }
        />
      ),
    },
    {
      title: t('音频补全倍率'),
      dataIndex: 'audioCompletionRatio',
      key: 'audioCompletionRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('音频输出')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'audioCompletionRatio', value)
          }
        />
      ),
    },
    {
      title: t('图片倍率'),
      dataIndex: 'imageRatio',
      key: 'imageRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('图片输入')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'imageRatio', value)
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

  const updateModel = (name, field, value) => {
    if (value !== '' && isNaN(value)) {
      showError('请输入数字');
      return;
    }
    setModels((prev) =>
      prev.map((model) => {
        if (model.name !== name) return model;
        const updated = { ...model, [field]: value };
        updated.hasConflict = buildConflictState(updated);
        return updated;
      }),
    );
  };

  const deleteModel = (name) => {
    setModels((prev) => prev.filter((model) => model.name !== name));
  };

  const handleTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      tokenPrice: value,
      ratio: hasValue(value) && !isNaN(value)
        ? (Number(value) / 2).toString()
        : '',
    };

    setCurrentModel(syncRatioFieldsFromTokenPrices(newState));
  };

  const handleCompletionTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      completionTokenPrice: value,
      completionRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(Number(value), Number(currentModel.tokenPrice))
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
          ? calculateRelativeRatio(Number(value), Number(currentModel.tokenPrice))
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
          ? calculateRelativeRatio(Number(value), Number(currentModel.tokenPrice))
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
          ? calculateRelativeRatio(Number(value), Number(currentModel.tokenPrice))
          : '',
    };

    setCurrentModel(syncRatioFieldsFromTokenPrices(newState));
  };

  const handleAudioCompletionTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      audioCompletionTokenPrice: value,
      audioCompletionRatio:
        hasValue(value) && hasValue(currentModel?.audioTokenPrice)
          ? calculateRelativeRatio(Number(value), Number(currentModel.audioTokenPrice))
          : '',
    };

    setCurrentModel(newState);
  };

  const handleImageTokenPriceChange = (value) => {
    const newState = {
      ...(currentModel || {}),
      imageTokenPrice: value,
      imageRatio:
        hasValue(value) && hasValue(currentModel?.tokenPrice)
          ? calculateRelativeRatio(Number(value), Number(currentModel.tokenPrice))
          : '',
    };

    setCurrentModel(newState);
  };

  const addOrUpdateModel = (values) => {
    // Check if we're editing an existing model or adding a new one
    const existingModelIndex = models.findIndex(
      (model) => model.name === values.name,
    );

    if (existingModelIndex >= 0) {
      // Update existing model
      setModels((prev) =>
        prev.map((model, index) => {
          if (index !== existingModelIndex) return model;
          const updated = {
            name: values.name,
            price: normalizeEditableValue(values.price),
            ratio: normalizeEditableValue(values.ratio),
            completionRatio: normalizeEditableValue(values.completionRatio),
            cacheRatio: normalizeEditableValue(values.cacheRatio),
            createCacheRatio: normalizeEditableValue(values.createCacheRatio),
            imageRatio: normalizeEditableValue(values.imageRatio),
            audioRatio: normalizeEditableValue(values.audioRatio),
            audioCompletionRatio: normalizeEditableValue(
              values.audioCompletionRatio,
            ),
          };
          updated.hasConflict = buildConflictState(updated);
          return updated;
        }),
      );
      setVisible(false);
      showSuccess(t('更新成功'));
    } else {
      // Add new model
      // Check if model name already exists
      if (models.some((model) => model.name === values.name)) {
        showError(t('模型名称已存在'));
        return;
      }

      setModels((prev) => {
        const newModel = {
          name: values.name,
          price: normalizeEditableValue(values.price),
          ratio: normalizeEditableValue(values.ratio),
          completionRatio: normalizeEditableValue(values.completionRatio),
          cacheRatio: normalizeEditableValue(values.cacheRatio),
          createCacheRatio: normalizeEditableValue(values.createCacheRatio),
          imageRatio: normalizeEditableValue(values.imageRatio),
          audioRatio: normalizeEditableValue(values.audioRatio),
          audioCompletionRatio: normalizeEditableValue(
            values.audioCompletionRatio,
          ),
        };
        newModel.hasConflict = buildConflictState(newModel);
        return [newModel, ...prev];
      });
      setVisible(false);
      showSuccess(t('添加成功'));
    }
  };

  const resetModalState = () => {
    setCurrentModel(null);
    setPricingMode('per-token');
    setPricingSubMode('ratio');
    setIsEditMode(false);
  };

  const editModel = (record) => {
    setIsEditMode(true);
    // Determine which pricing mode to use based on the model's current configuration
    let initialPricingMode = 'per-token';
    let initialPricingSubMode = 'ratio';

    if (record.price !== '') {
      initialPricingMode = 'per-request';
    } else {
      initialPricingMode = 'per-token';
      // We default to ratio mode, but could set to token-price if needed
    }

    // Set the pricing modes for the form
    setPricingMode(initialPricingMode);
    setPricingSubMode(initialPricingSubMode);

    // Create a copy of the model data to avoid modifying the original
    const modelCopy = {
      ...record,
      ...buildTokenPriceFieldsFromRatios(record),
    };

    // Set the current model
    setCurrentModel(modelCopy);

    // Open the modal
    setVisible(true);

    // Use setTimeout to ensure the form is rendered before setting values
    setTimeout(() => {
      if (formRef.current) {
        // Update the form fields based on pricing mode
        const formValues = {
          name: modelCopy.name,
        };

        if (initialPricingMode === 'per-request') {
          formValues.priceInput = modelCopy.price;
        } else if (initialPricingMode === 'per-token') {
          formValues.ratioInput = modelCopy.ratio;
          formValues.completionRatioInput = modelCopy.completionRatio;
          formValues.cacheRatioInput = modelCopy.cacheRatio;
          formValues.createCacheRatioInput = modelCopy.createCacheRatio;
          formValues.audioRatioInput = modelCopy.audioRatio;
          formValues.audioCompletionRatioInput = modelCopy.audioCompletionRatio;
          formValues.imageRatioInput = modelCopy.imageRatio;
          formValues.modelTokenPrice = modelCopy.tokenPrice;
          formValues.completionTokenPrice = modelCopy.completionTokenPrice;
          formValues.cacheTokenPrice = modelCopy.cacheTokenPrice;
          formValues.createCacheTokenPrice = modelCopy.createCacheTokenPrice;
          formValues.audioTokenPrice = modelCopy.audioTokenPrice;
          formValues.audioCompletionTokenPrice = modelCopy.audioCompletionTokenPrice;
          formValues.imageTokenPrice = modelCopy.imageTokenPrice;
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

            if (pricingMode === 'per-token' && pricingSubMode === 'token-price') {
              valuesToSave = syncRatioFieldsFromTokenPrices(valuesToSave);
            }

            if (pricingMode === 'per-token') {
              valuesToSave.price = '';
            } else {
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

                  // Instead of resetting all values, convert between modes
                  if (currentModel) {
                    const updatedModel =
                      newMode === 'per-token'
                        ? {
                            ...currentModel,
                            ...buildTokenPriceFieldsFromRatios(currentModel),
                          }
                        : { ...currentModel };

                    // Update formRef with converted values
                    if (formRef.current) {
                      const formValues = {
                        name: updatedModel.name,
                      };

                      if (newMode === 'per-request') {
                        formValues.priceInput = normalizeEditableValue(
                          updatedModel.price,
                        );
                      } else if (newMode === 'per-token') {
                        formValues.ratioInput = normalizeEditableValue(
                          updatedModel.ratio,
                        );
                        formValues.completionRatioInput =
                          normalizeEditableValue(updatedModel.completionRatio);
                        formValues.cacheRatioInput =
                          normalizeEditableValue(updatedModel.cacheRatio);
                        formValues.createCacheRatioInput =
                          normalizeEditableValue(updatedModel.createCacheRatio);
                        formValues.audioRatioInput =
                          normalizeEditableValue(updatedModel.audioRatio);
                        formValues.audioCompletionRatioInput =
                          normalizeEditableValue(
                            updatedModel.audioCompletionRatio,
                          );
                        formValues.imageRatioInput =
                          normalizeEditableValue(updatedModel.imageRatio);
                        formValues.modelTokenPrice =
                          normalizeEditableValue(updatedModel.tokenPrice);
                        formValues.completionTokenPrice =
                          normalizeEditableValue(updatedModel.completionTokenPrice);
                        formValues.cacheTokenPrice =
                          normalizeEditableValue(updatedModel.cacheTokenPrice);
                        formValues.createCacheTokenPrice =
                          normalizeEditableValue(
                            updatedModel.createCacheTokenPrice,
                          );
                        formValues.audioTokenPrice =
                          normalizeEditableValue(updatedModel.audioTokenPrice);
                        formValues.audioCompletionTokenPrice =
                          normalizeEditableValue(
                            updatedModel.audioCompletionTokenPrice,
                          );
                        formValues.imageTokenPrice =
                          normalizeEditableValue(updatedModel.imageTokenPrice);
                      }

                      formRef.current.setValues(formValues);
                    }

                    // Update the model state
                    setCurrentModel(updatedModel);
                  }
                }}
              >
                <Radio value='per-token'>{t('按量计费')}</Radio>
                <Radio value='per-request'>{t('按次计费')}</Radio>
              </RadioGroup>
            </div>
          </Form.Section>

          {pricingMode === 'per-token' && (
            <>
              <Form.Section text={t('价格设置方式')}>
                <div style={{ marginBottom: '16px' }}>
                  <RadioGroup
                    type='button'
                    value={pricingSubMode}
                    onChange={(e) => {
                      const newSubMode = e.target.value;
                      const oldSubMode = pricingSubMode;
                      setPricingSubMode(newSubMode);

                      // Handle conversion between submodes
                      if (currentModel) {
                        const updatedModel =
                          oldSubMode === 'ratio' && newSubMode === 'token-price'
                            ? {
                                ...currentModel,
                                ...buildTokenPriceFieldsFromRatios(currentModel),
                              }
                            : { ...currentModel };

                        // Update the form values
                        if (formRef.current) {
                          const formValues = {};

                          if (newSubMode === 'ratio') {
                            formValues.ratioInput = normalizeEditableValue(
                              updatedModel.ratio,
                            );
                            formValues.completionRatioInput =
                              normalizeEditableValue(
                                updatedModel.completionRatio,
                              );
                            formValues.cacheRatioInput =
                              normalizeEditableValue(updatedModel.cacheRatio);
                            formValues.createCacheRatioInput =
                              normalizeEditableValue(
                                updatedModel.createCacheRatio,
                              );
                            formValues.audioRatioInput =
                              normalizeEditableValue(updatedModel.audioRatio);
                            formValues.audioCompletionRatioInput =
                              normalizeEditableValue(
                                updatedModel.audioCompletionRatio,
                              );
                            formValues.imageRatioInput =
                              normalizeEditableValue(updatedModel.imageRatio);
                          } else if (newSubMode === 'token-price') {
                            formValues.modelTokenPrice =
                              normalizeEditableValue(updatedModel.tokenPrice);
                            formValues.completionTokenPrice =
                              normalizeEditableValue(
                                updatedModel.completionTokenPrice,
                              );
                            formValues.cacheTokenPrice =
                              normalizeEditableValue(updatedModel.cacheTokenPrice);
                            formValues.createCacheTokenPrice =
                              normalizeEditableValue(
                                updatedModel.createCacheTokenPrice,
                              );
                            formValues.audioTokenPrice =
                              normalizeEditableValue(updatedModel.audioTokenPrice);
                            formValues.audioCompletionTokenPrice =
                              normalizeEditableValue(
                                updatedModel.audioCompletionTokenPrice,
                              );
                            formValues.imageTokenPrice =
                              normalizeEditableValue(updatedModel.imageTokenPrice);
                          }

                          formRef.current.setValues(formValues);
                        }

                        setCurrentModel(updatedModel);
                      }
                    }}
                  >
                    <Radio value='ratio'>{t('按倍率设置')}</Radio>
                    <Radio value='token-price'>{t('按价格设置')}</Radio>
                  </RadioGroup>
                </div>
              </Form.Section>

              {pricingSubMode === 'ratio' && (
                <>
                  <Form.Input
                    field='ratioInput'
                    label={t('模型倍率')}
                    placeholder={t('输入模型倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        ratio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(currentModel?.ratio)}
                  />
                  <Form.Input
                    field='completionRatioInput'
                    label={t('补全倍率')}
                    placeholder={t('输入补全倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        completionRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(
                      currentModel?.completionRatio,
                    )}
                  />
                  <Form.Input
                    field='cacheRatioInput'
                    label={t('缓存倍率')}
                    placeholder={t('缓存读取倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        cacheRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(currentModel?.cacheRatio)}
                  />
                  <Form.Input
                    field='createCacheRatioInput'
                    label={t('缓存创建倍率')}
                    placeholder={t('缓存创建倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        createCacheRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(
                      currentModel?.createCacheRatio,
                    )}
                  />
                  <Form.Input
                    field='audioRatioInput'
                    label={t('音频倍率')}
                    placeholder={t('音频输入倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        audioRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(currentModel?.audioRatio)}
                  />
                  <Form.Input
                    field='audioCompletionRatioInput'
                    label={t('音频补全倍率')}
                    placeholder={t('音频输出倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        audioCompletionRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(
                      currentModel?.audioCompletionRatio,
                    )}
                  />
                  <Form.Input
                    field='imageRatioInput'
                    label={t('图片倍率')}
                    placeholder={t('图片输入倍率')}
                    onChange={(value) =>
                      setCurrentModel((prev) => ({
                        ...(prev || {}),
                        imageRatio: value,
                      }))
                    }
                    initValue={normalizeEditableValue(currentModel?.imageRatio)}
                  />
                </>
              )}

              {pricingSubMode === 'token-price' && (
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
                  <Form.Input
                    field='imageTokenPrice'
                    label={t('图片输入价格')}
                    onChange={(value) => {
                      handleImageTokenPriceChange(value);
                    }}
                    initValue={normalizeEditableValue(
                      currentModel?.imageTokenPrice,
                    )}
                    suffix={t('$/1M tokens')}
                  />
                </>
              )}
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
