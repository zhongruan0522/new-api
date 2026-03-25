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

import React, { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Input,
  Modal,
  Form,
  Space,
  Typography,
  Radio,
  Notification,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconPlus,
  IconSearch,
  IconSave,
  IconBolt,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const EXTRA_RATIO_FIELDS = [
  'cacheRatio',
  'createCacheRatio',
  'imageRatio',
  'audioRatio',
  'audioCompletionRatio',
];

// 未配置模型页与可视化编辑器共用同一组额外计费字段，避免保存时遗漏。
const createEmptyExtraPricing = () => ({
  cacheRatio: '',
  createCacheRatio: '',
  imageRatio: '',
  audioRatio: '',
  audioCompletionRatio: '',
});

const normalizeEditableValue = (value) =>
  value === '' || value === undefined || value === null ? '' : `${value}`;

const getBatchFillTypeLabel = (type, t) => {
  switch (type) {
    case 'price':
      return t('固定价格');
    case 'ratio':
      return t('模型倍率');
    case 'completionRatio':
      return t('补全倍率');
    case 'cacheRatio':
      return t('缓存读取倍率');
    case 'createCacheRatio':
      return t('缓存创建倍率');
    case 'audioRatio':
      return t('音频倍率');
    case 'audioCompletionRatio':
      return t('音频补全倍率');
    case 'imageRatio':
      return t('图片倍率');
    default:
      return t('模型倍率和补全倍率');
  }
};

const getBatchFillValueLabel = (type, t) => {
  switch (type) {
    case 'price':
      return t('固定价格值');
    case 'ratio':
      return t('模型倍率值');
    case 'cacheRatio':
      return t('缓存读取倍率值');
    case 'createCacheRatio':
      return t('缓存创建倍率值');
    case 'audioRatio':
      return t('音频倍率值');
    case 'audioCompletionRatio':
      return t('音频补全倍率值');
    case 'imageRatio':
      return t('图片倍率值');
    default:
      return t('补全倍率值');
  }
};

export default function ModelRatioNotSetEditor(props) {
  const { t } = useTranslation();
  const [models, setModels] = useState([]);
  const [visible, setVisible] = useState(false);
  const [batchVisible, setBatchVisible] = useState(false);
  const [currentModel, setCurrentModel] = useState(null);
  const [searchText, setSearchText] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [enabledModels, setEnabledModels] = useState([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [batchFillType, setBatchFillType] = useState('ratio');
  const [batchFillValue, setBatchFillValue] = useState('');
  const [batchRatioValue, setBatchRatioValue] = useState('');
  const [batchCompletionRatioValue, setBatchCompletionRatioValue] =
    useState('');
  const { Text } = Typography;
  // 定义可选的每页显示条数
  const pageSizeOptions = [10, 20, 50, 100];

  const getAllEnabledModels = async () => {
    try {
      const res = await API.get('/api/channel/models_enabled');
      const { success, message, data } = res.data;
      if (success) {
        setEnabledModels(data);
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('获取启用模型失败:'), error);
      showError(t('获取启用模型失败'));
    }
  };

  useEffect(() => {
    // 获取所有启用的模型
    getAllEnabledModels();
  }, []);

  useEffect(() => {
    try {
      const modelPrice = JSON.parse(props.options.ModelPrice || '{}');
      const modelRatio = JSON.parse(props.options.ModelRatio || '{}');
      const completionRatio = JSON.parse(props.options.CompletionRatio || '{}');
      const cacheRatio = JSON.parse(props.options.CacheRatio || '{}');
      const createCacheRatio = JSON.parse(props.options.CreateCacheRatio || '{}');
      const imageRatio = JSON.parse(props.options.ImageRatio || '{}');
      const audioRatio = JSON.parse(props.options.AudioRatio || '{}');
      const audioCompletionRatio = JSON.parse(
        props.options.AudioCompletionRatio || '{}',
      );

      // 找出所有未设置价格和倍率的模型
      const unsetModels = enabledModels.filter((modelName) => {
        const hasPrice = modelPrice[modelName] !== undefined;
        const hasRatio = modelRatio[modelName] !== undefined;

        // 如果模型没有价格或者没有倍率设置，则显示
        return !hasPrice && !hasRatio;
      });

      // 创建模型数据
      const modelData = unsetModels.map((name) => ({
        name,
        price: normalizeEditableValue(modelPrice[name]),
        ratio: normalizeEditableValue(modelRatio[name]),
        completionRatio: normalizeEditableValue(completionRatio[name]),
        cacheRatio: normalizeEditableValue(cacheRatio[name]),
        createCacheRatio: normalizeEditableValue(createCacheRatio[name]),
        imageRatio: normalizeEditableValue(imageRatio[name]),
        audioRatio: normalizeEditableValue(audioRatio[name]),
        audioCompletionRatio: normalizeEditableValue(audioCompletionRatio[name]),
      }));

      setModels(modelData);
      // 清空选择
      setSelectedRowKeys([]);
    } catch (error) {
      console.error(t('JSON解析错误:'), error);
    }
  }, [props.options, enabledModels]);

  // 首先声明分页相关的工具函数
  const getPagedData = (data, currentPage, pageSize) => {
    const start = (currentPage - 1) * pageSize;
    const end = start + pageSize;
    return data.slice(start, end);
  };

  // 处理页面大小变化
  const handlePageSizeChange = (size) => {
    setPageSize(size);
    // 重新计算当前页，避免数据丢失
    const totalPages = Math.ceil(filteredModels.length / size);
    if (currentPage > totalPages) {
      setCurrentPage(totalPages || 1);
    }
  };

  // 在 return 语句之前，先处理过滤和分页逻辑
  const filteredModels = models.filter((model) =>
    searchText ? model.name.includes(searchText) : true,
  );

  // 然后基于过滤后的数据计算分页数据
  const pagedData = getPagedData(filteredModels, currentPage, pageSize);

  const SubmitData = async () => {
    setLoading(true);
    const output = {
      ModelPrice: JSON.parse(props.options.ModelPrice || '{}'),
      ModelRatio: JSON.parse(props.options.ModelRatio || '{}'),
      CompletionRatio: JSON.parse(props.options.CompletionRatio || '{}'),
      CacheRatio: JSON.parse(props.options.CacheRatio || '{}'),
      CreateCacheRatio: JSON.parse(props.options.CreateCacheRatio || '{}'),
      ImageRatio: JSON.parse(props.options.ImageRatio || '{}'),
      AudioRatio: JSON.parse(props.options.AudioRatio || '{}'),
      AudioCompletionRatio: JSON.parse(
        props.options.AudioCompletionRatio || '{}',
      ),
    };

    try {
      // 数据转换 - 只处理已修改的模型
      models.forEach((model) => {
        delete output.ModelPrice[model.name];
        delete output.ModelRatio[model.name];
        delete output.CompletionRatio[model.name];
        delete output.CacheRatio[model.name];
        delete output.CreateCacheRatio[model.name];
        delete output.ImageRatio[model.name];
        delete output.AudioRatio[model.name];
        delete output.AudioCompletionRatio[model.name];

        // 只有当用户设置了值时才更新
        if (model.price !== '') {
          // 如果价格不为空，则转换为浮点数，忽略倍率参数
          output.ModelPrice[model.name] = parseFloat(model.price);
        } else {
          if (model.ratio !== '')
            output.ModelRatio[model.name] = parseFloat(model.ratio);
          if (model.completionRatio !== '')
            output.CompletionRatio[model.name] = parseFloat(
              model.completionRatio,
            );
          if (model.cacheRatio !== '')
            output.CacheRatio[model.name] = parseFloat(model.cacheRatio);
          if (model.createCacheRatio !== '')
            output.CreateCacheRatio[model.name] = parseFloat(
              model.createCacheRatio,
            );
          if (model.imageRatio !== '')
            output.ImageRatio[model.name] = parseFloat(model.imageRatio);
          if (model.audioRatio !== '')
            output.AudioRatio[model.name] = parseFloat(model.audioRatio);
          if (model.audioCompletionRatio !== '')
            output.AudioCompletionRatio[model.name] = parseFloat(
              model.audioCompletionRatio,
            );
        }
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

      // 批量处理请求
      const results = await Promise.all(requestQueue);

      // 验证结果
      if (requestQueue.length === 1) {
        if (results.includes(undefined)) return;
      } else if (requestQueue.length > 1) {
        if (results.includes(undefined)) {
          return showError(t('部分保存失败，请重试'));
        }
      }

      // 检查每个请求的结果
      for (const res of results) {
        if (!res.data.success) {
          return showError(res.data.message);
        }
      }

      showSuccess(t('保存成功'));
      props.refresh();
      // 重新获取未设置的模型
      getAllEnabledModels();
    } catch (error) {
      console.error(t('保存失败:'), error);
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: t('模型名称'),
      dataIndex: 'name',
      key: 'name',
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
          placeholder={record.price !== '' ? t('模型倍率') : t('输入模型倍率')}
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
          placeholder={record.price !== '' ? t('补全倍率') : t('输入补全倍率')}
          disabled={record.price !== ''}
          onChange={(value) =>
            updateModel(record.name, 'completionRatio', value)
          }
        />
      ),
    },
    {
      title: t('缓存读取倍率'),
      dataIndex: 'cacheRatio',
      key: 'cacheRatio',
      render: (text, record) => (
        <Input
          value={text}
          placeholder={t('输入缓存读取倍率')}
          disabled={record.price !== ''}
          onChange={(value) => updateModel(record.name, 'cacheRatio', value)}
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
          placeholder={t('输入缓存创建倍率')}
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
          placeholder={t('输入音频倍率')}
          disabled={record.price !== ''}
          onChange={(value) => updateModel(record.name, 'audioRatio', value)}
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
          placeholder={t('输入音频补全倍率')}
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
          placeholder={t('输入图片倍率')}
          disabled={record.price !== ''}
          onChange={(value) => updateModel(record.name, 'imageRatio', value)}
        />
      ),
    },
  ];

  const updateModel = (name, field, value) => {
    if (value !== '' && isNaN(value)) {
      showError(t('请输入数字'));
      return;
    }
    setModels((prev) =>
      prev.map((model) =>
        model.name === name ? { ...model, [field]: value } : model,
      ),
    );
  };

  const addModel = (values) => {
    // 检查模型名称是否存在, 如果存在则拒绝添加
    if (models.some((model) => model.name === values.name)) {
      showError(t('模型名称已存在'));
      return;
    }
    setModels((prev) => [
      {
        name: values.name,
        price: normalizeEditableValue(values.price),
        ratio: normalizeEditableValue(values.ratio),
        completionRatio: normalizeEditableValue(values.completionRatio),
        cacheRatio: normalizeEditableValue(values.cacheRatio),
        createCacheRatio: normalizeEditableValue(values.createCacheRatio),
        imageRatio: normalizeEditableValue(values.imageRatio),
        audioRatio: normalizeEditableValue(values.audioRatio),
        audioCompletionRatio: normalizeEditableValue(values.audioCompletionRatio),
      },
      ...prev,
    ]);
    setVisible(false);
    showSuccess(t('添加成功'));
  };

  // 批量填充功能
  const handleBatchFill = () => {
    if (selectedRowKeys.length === 0) {
      showError(t('请先选择需要批量设置的模型'));
      return;
    }

    if (batchFillType === 'bothRatio') {
      if (batchRatioValue === '' || batchCompletionRatioValue === '') {
        showError(t('请输入模型倍率和补全倍率'));
        return;
      }
      if (isNaN(batchRatioValue) || isNaN(batchCompletionRatioValue)) {
        showError(t('请输入有效的数字'));
        return;
      }
    } else {
      if (batchFillValue === '') {
        showError(t('请输入填充值'));
        return;
      }
      if (isNaN(batchFillValue)) {
        showError(t('请输入有效的数字'));
        return;
      }
    }

    // 根据选择的类型批量更新模型
    setModels((prev) =>
      prev.map((model) => {
        if (selectedRowKeys.includes(model.name)) {
          if (batchFillType === 'price') {
            return {
              ...model,
              price: batchFillValue,
              ratio: '',
              completionRatio: '',
              ...createEmptyExtraPricing(),
            };
          } else if (batchFillType === 'ratio') {
            return {
              ...model,
              price: '',
              ratio: batchFillValue,
            };
          } else if (batchFillType === 'completionRatio') {
            return {
              ...model,
              price: '',
              completionRatio: batchFillValue,
            };
          } else if (EXTRA_RATIO_FIELDS.includes(batchFillType)) {
            return {
              ...model,
              price: '',
              [batchFillType]: batchFillValue,
            };
          } else if (batchFillType === 'bothRatio') {
            return {
              ...model,
              price: '',
              ratio: batchRatioValue,
              completionRatio: batchCompletionRatioValue,
            };
          }
        }
        return model;
      }),
    );

    setBatchVisible(false);
    Notification.success({
      title: t('批量设置成功'),
      content: t('已为 {{count}} 个模型设置{{type}}', {
        count: selectedRowKeys.length,
        type: getBatchFillTypeLabel(batchFillType, t),
      }),
      duration: 3,
    });
  };

  const handleBatchTypeChange = (value) => {
    setBatchFillType(value);

    // 切换类型时清空对应的值
    if (value !== 'bothRatio') {
      setBatchFillValue('');
    } else {
      setBatchRatioValue('');
      setBatchCompletionRatioValue('');
    }
  };

  const rowSelection = {
    selectedRowKeys,
    onChange: (selectedKeys) => {
      setSelectedRowKeys(selectedKeys);
    },
  };

  return (
    <>
      <Space vertical align='start' style={{ width: '100%' }}>
        <Space className='mt-2'>
          <Button
            icon={<IconPlus />}
            onClick={() => {
              setCurrentModel({ ...createEmptyExtraPricing(), priceMode: false });
              setVisible(true);
            }}
          >
            {t('添加模型')}
          </Button>
          <Button
            icon={<IconBolt />}
            type='secondary'
            onClick={() => setBatchVisible(true)}
            disabled={selectedRowKeys.length === 0}
          >
            {t('批量设置')} ({selectedRowKeys.length})
          </Button>
          <Button
            type='primary'
            icon={<IconSave />}
            onClick={SubmitData}
            loading={loading}
          >
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
          />
        </Space>

        <Text>
          {t('此页面仅显示未设置价格或倍率的模型，设置后将自动从列表中移除')}
        </Text>

        <Table
          columns={columns}
          dataSource={pagedData}
          rowSelection={rowSelection}
          rowKey='name'
          pagination={{
            currentPage: currentPage,
            pageSize: pageSize,
            total: filteredModels.length,
            onPageChange: (page) => setCurrentPage(page),
            onPageSizeChange: handlePageSizeChange,
            pageSizeOptions: pageSizeOptions,
            showTotal: true,
            showSizeChanger: true,
          }}
          empty={
            <div style={{ textAlign: 'center', padding: '20px' }}>
              {t('没有未设置的模型')}
            </div>
          }
        />
      </Space>

      {/* 添加模型弹窗 */}
      <Modal
        title={t('添加模型')}
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setCurrentModel(null);
        }}
        onOk={() => {
          currentModel && addModel(currentModel);
        }}
      >
        <Form>
          <Form.Input
            field='name'
            label={t('模型名称')}
            placeholder='strawberry'
            required
            onChange={(value) =>
              setCurrentModel((prev) => ({ ...prev, name: value }))
            }
          />
          <Form.Switch
            field='priceMode'
            label={
              <>
                {t('定价模式')}：
                {currentModel?.priceMode ? t('固定价格') : t('倍率模式')}
              </>
            }
            onChange={(checked) => {
              setCurrentModel((prev) => ({
                ...prev,
                price: '',
                ratio: '',
                completionRatio: '',
                ...createEmptyExtraPricing(),
                priceMode: checked,
              }));
            }}
          />
          {currentModel?.priceMode ? (
            <Form.Input
              field='price'
              label={t('固定价格(每次)')}
              placeholder={t('输入每次价格')}
              onChange={(value) =>
                setCurrentModel((prev) => ({ ...prev, price: value }))
              }
            />
          ) : (
            <>
              <Form.Input
                field='ratio'
                label={t('模型倍率')}
                placeholder={t('输入模型倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({ ...prev, ratio: value }))
                }
              />
              <Form.Input
                field='completionRatio'
                label={t('补全倍率')}
                placeholder={t('输入补全价格')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({
                    ...prev,
                    completionRatio: value,
                  }))
                }
              />
              <Form.Input
                field='cacheRatio'
                label={t('缓存读取倍率')}
                placeholder={t('输入缓存读取倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({ ...prev, cacheRatio: value }))
                }
              />
              <Form.Input
                field='createCacheRatio'
                label={t('缓存创建倍率')}
                placeholder={t('输入缓存创建倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({
                    ...prev,
                    createCacheRatio: value,
                  }))
                }
              />
              <Form.Input
                field='audioRatio'
                label={t('音频倍率')}
                placeholder={t('输入音频倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({ ...prev, audioRatio: value }))
                }
              />
              <Form.Input
                field='audioCompletionRatio'
                label={t('音频补全倍率')}
                placeholder={t('输入音频补全倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({
                    ...prev,
                    audioCompletionRatio: value,
                  }))
                }
              />
              <Form.Input
                field='imageRatio'
                label={t('图片倍率')}
                placeholder={t('输入图片倍率')}
                onChange={(value) =>
                  setCurrentModel((prev) => ({ ...prev, imageRatio: value }))
                }
              />
            </>
          )}
        </Form>
      </Modal>

      {/* 批量设置弹窗 */}
      <Modal
        title={t('批量设置模型参数')}
        visible={batchVisible}
        onCancel={() => setBatchVisible(false)}
        onOk={handleBatchFill}
        width={500}
      >
        <Form>
          <Form.Section text={t('设置类型')}>
            <div style={{ marginBottom: '16px' }}>
              <Space>
                <Radio
                  checked={batchFillType === 'price'}
                  onChange={() => handleBatchTypeChange('price')}
                >
                  {t('固定价格')}
                </Radio>
                <Radio
                  checked={batchFillType === 'ratio'}
                  onChange={() => handleBatchTypeChange('ratio')}
                >
                  {t('模型倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'completionRatio'}
                  onChange={() => handleBatchTypeChange('completionRatio')}
                >
                  {t('补全倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'cacheRatio'}
                  onChange={() => handleBatchTypeChange('cacheRatio')}
                >
                  {t('缓存读取倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'createCacheRatio'}
                  onChange={() => handleBatchTypeChange('createCacheRatio')}
                >
                  {t('缓存创建倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'audioRatio'}
                  onChange={() => handleBatchTypeChange('audioRatio')}
                >
                  {t('音频倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'audioCompletionRatio'}
                  onChange={() => handleBatchTypeChange('audioCompletionRatio')}
                >
                  {t('音频补全倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'imageRatio'}
                  onChange={() => handleBatchTypeChange('imageRatio')}
                >
                  {t('图片倍率')}
                </Radio>
                <Radio
                  checked={batchFillType === 'bothRatio'}
                  onChange={() => handleBatchTypeChange('bothRatio')}
                >
                  {t('模型倍率和补全倍率同时设置')}
                </Radio>
              </Space>
            </div>
          </Form.Section>

          {batchFillType === 'bothRatio' ? (
            <>
              <Form.Input
                field='batchRatioValue'
                label={t('模型倍率值')}
                placeholder={t('请输入模型倍率')}
                value={batchRatioValue}
                onChange={(value) => setBatchRatioValue(value)}
              />
              <Form.Input
                field='batchCompletionRatioValue'
                label={t('补全倍率值')}
                placeholder={t('请输入补全倍率')}
                value={batchCompletionRatioValue}
                onChange={(value) => setBatchCompletionRatioValue(value)}
              />
            </>
          ) : (
            <Form.Input
              field='batchFillValue'
              label={getBatchFillValueLabel(batchFillType, t)}
              placeholder={t('请输入数值')}
              value={batchFillValue}
              onChange={(value) => setBatchFillValue(value)}
            />
          )}

          <Text type='tertiary'>
            {t('将为选中的 ')} <Text strong>{selectedRowKeys.length}</Text>{' '}
            {t(' 个模型设置相同的值')}
          </Text>
          <div style={{ marginTop: '8px' }}>
            <Text type='tertiary'>
              {t('当前设置类型: ')}{' '}
              <Text strong>{getBatchFillTypeLabel(batchFillType, t)}</Text>
            </Text>
          </div>
        </Form>
      </Modal>
    </>
  );
}
