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

import React, { useState, useEffect, useRef, useMemo } from 'react';
import {
  SideSheet,
  Form,
  Button,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Avatar,
  Col,
  Row,
} from '@douyinfe/semi-ui';
import {
  IconSave,
  IconClose,
  IconLayers,
} from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const endpointOptions = [
  { label: 'OpenAI', value: 'openai' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Gemini', value: 'gemini' },
  { label: 'Image Generation', value: 'image-generation' },
  { label: 'Jina Rerank', value: 'jina-rerank' },
];

const { Text, Title } = Typography;

const EditModelModal = (props) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const isEdit = props.editingModel && props.editingModel.id !== undefined;
  const placement = useMemo(() => (isEdit ? 'right' : 'left'), [isEdit]);

  // 供应商列表
  const [vendors, setVendors] = useState([]);

  // 获取供应商列表
  const fetchVendors = async () => {
    try {
      const res = await API.get('/api/vendors/?page_size=1000'); // 获取全部供应商
      if (res.data.success) {
        const items = res.data.data.items || res.data.data || [];
        setVendors(Array.isArray(items) ? items : []);
      }
    } catch (error) {
      // ignore
    }
  };

  useEffect(() => {
    if (props.visiable) {
      fetchVendors();
    }
  }, [props.visiable]);

  const getInitValues = () => ({
    model_name: '',
    description: '',
    tags: [],
    vendor_id: undefined,
    vendor: '',
    vendor_icon: '',
    endpoints: [],
    status: true,
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const loadModel = async () => {
    if (!isEdit || !props.editingModel.id) return;

    setLoading(true);
    try {
      const res = await API.get(`/api/models/${props.editingModel.id}`);
      const { success, message, data } = res.data;
      if (success) {
        // 处理tags
        if (data.tags) {
          data.tags = data.tags.split(',').filter(Boolean);
        } else {
          data.tags = [];
        }
        // 处理endpoints
        if (data.endpoints) {
          try {
            data.endpoints = JSON.parse(data.endpoints);
          } catch (e) {
            data.endpoints = [];
          }
        } else {
          data.endpoints = [];
        }
        // 处理status，将数字转为布尔值
        data.status = data.status === 1;
        if (formApiRef.current) {
          formApiRef.current.setValues({ ...getInitValues(), ...data });
        }
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('加载模型信息失败'));
    }
    setLoading(false);
  };

  useEffect(() => {
    if (formApiRef.current) {
      if (!isEdit) {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [props.editingModel?.id]);

  useEffect(() => {
    if (props.visiable) {
      if (isEdit) {
        loadModel();
      } else {
        formApiRef.current?.setValues(getInitValues());
      }
    } else {
      formApiRef.current?.reset();
    }
  }, [props.visiable, props.editingModel?.id]);

  const submit = async (values) => {
    setLoading(true);
    try {
      const submitData = {
        ...values,
        tags: Array.isArray(values.tags) ? values.tags.join(',') : values.tags,
        endpoints: JSON.stringify(values.endpoints || []),
        status: values.status ? 1 : 0,
      };

      if (isEdit) {
        submitData.id = props.editingModel.id;
        const res = await API.put('/api/models/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('模型更新成功！'));
          props.refresh();
          props.handleClose();
        } else {
          showError(t(message));
        }
      } else {
        const res = await API.post('/api/models/', submitData);
        const { success, message } = res.data;
        if (success) {
          showSuccess(t('模型创建成功！'));
          props.refresh();
          props.handleClose();
        } else {
          showError(t(message));
        }
      }
    } catch (error) {
      showError(error.response?.data?.message || t('操作失败'));
    }
    setLoading(false);
    formApiRef.current?.setValues(getInitValues());
  };

  return (
    <SideSheet
      placement={placement}
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>
              {t('更新')}
            </Tag>
          ) : (
            <Tag color='green' shape='circle'>
              {t('新建')}
            </Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('更新模型信息') : t('创建新的模型')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={props.visiable}
      width={isMobile ? '100%' : 600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button
              theme='solid'
              className='!rounded-lg'
              onClick={() => formApiRef.current?.submitForm()}
              icon={<IconSave />}
              loading={loading}
            >
              {t('提交')}
            </Button>
            <Button
              theme='light'
              className='!rounded-lg'
              type='primary'
              onClick={handleCancel}
              icon={<IconClose />}
            >
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={() => handleCancel()}
    >
      <Spin spinning={loading}>
        <Form
          key={isEdit ? 'edit' : 'new'}
          initValues={getInitValues()}
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={submit}
        >
          {({ values }) => (
            <div className='p-2'>
              {/* 基本信息 */}
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='green' className='mr-2 shadow-md'>
                    <IconLayers size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('基本信息')}</Text>
                    <div className='text-xs text-gray-600'>{t('设置模型的基本信息')}</div>
                  </div>
                </div>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input
                      field='model_name'
                      label={t('模型名称')}
                      placeholder={t('请输入模型名称，如：gpt-4')}
                      rules={[{ required: true, message: t('请输入模型名称') }]}
                      disabled={isEdit}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TextArea
                      field='description'
                      label={t('描述')}
                      placeholder={t('请输入模型描述')}
                      rows={3}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    <Form.TagInput
                      field='tags'
                      label={t('标签')}
                      placeholder={t('输入标签后按回车添加')}
                      addOnBlur
                      showClear
                      style={{ width: '100%' }}
                    />
                  </Col>
                </Row>
              </Card>

              {/* 供应商信息 */}
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconLayers size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('供应商信息')}</Text>
                    <div className='text-xs text-gray-600'>{t('设置模型的供应商相关信息')}</div>
                  </div>
                </div>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Select
                      field='vendor_id'
                      label={t('供应商')}
                      placeholder={t('选择模型供应商')}
                      optionList={vendors.map(v => ({ label: v.name, value: v.id }))}
                      filter
                      showClear
                      style={{ width: '100%' }}
                      onChange={(value) => {
                        const vendorInfo = vendors.find(v => v.id === value);
                        if (vendorInfo && formApiRef.current) {
                          formApiRef.current.setValue('vendor', vendorInfo.name);
                        }
                      }}
                    />
                  </Col>
                </Row>
              </Card>

              {/* 功能配置 */}
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='purple' className='mr-2 shadow-md'>
                    <IconLayers size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('功能配置')}</Text>
                    <div className='text-xs text-gray-600'>{t('设置模型的功能和状态')}</div>
                  </div>
                </div>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Select
                      field='endpoints'
                      label={t('支持端点')}
                      placeholder={t('选择模型支持的端点类型')}
                      optionList={endpointOptions}
                      multiple
                      showClear
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={24}>
                    <Form.Switch
                      field='status'
                      label={t('状态')}
                      size="large"
                    />
                  </Col>
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

export default EditModelModal;