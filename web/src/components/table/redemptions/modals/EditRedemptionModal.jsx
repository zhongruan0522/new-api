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
import { useTranslation } from 'react-i18next';
import {
  API,
  downloadTextAsFile,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaWithPrompt,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  Modal,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
  Progress,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconSave,
  IconClose,
  IconGift,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

// 单次 API 请求的最大创建数量，与后端 controller/redemption.go 保持一致
const BATCH_SIZE = 100;

const EditRedemptionModal = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingRedemption.id !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);

  const getInitValues = () => ({
    name: '',
    quota: 100000,
    count: 1,
    expired_time: null,
  });

  const handleCancel = () => {
    props.handleClose();
  };

  const loadRedemption = async () => {
    setLoading(true);
    let res = await API.get(`/api/redemption/${props.editingRedemption.id}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.expired_time === 0) {
        data.expired_time = null;
      } else {
        data.expired_time = new Date(data.expired_time * 1000);
      }
      formApiRef.current?.setValues({ ...getInitValues(), ...data });
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (formApiRef.current) {
      if (isEdit) {
        loadRedemption();
      } else {
        formApiRef.current.setValues(getInitValues());
      }
    }
  }, [props.editingRedemption.id]);

  const [batchProgress, setBatchProgress] = useState(null); // { current, total }

  const submit = async (values) => {
    let name = values.name;
    if (!isEdit && (!name || name === '')) {
      name = renderQuota(values.quota);
    }
    setLoading(true);
    let localInputs = { ...values };
    localInputs.count = parseInt(localInputs.count) || 0;
    localInputs.quota = parseInt(localInputs.quota) || 0;
    localInputs.name = name;
    if (!localInputs.expired_time) {
      localInputs.expired_time = 0;
    } else {
      localInputs.expired_time = Math.floor(
        localInputs.expired_time.getTime() / 1000,
      );
    }

    if (isEdit) {
      let res = await API.put(`/api/redemption/`, {
        ...localInputs,
        id: parseInt(props.editingRedemption.id),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('兑换码更新成功！'));
        props.refresh();
        props.handleClose();
      } else {
        showError(message);
      }
      setLoading(false);
      return;
    }

    // 新建：分批创建兑换码
    const totalCount = localInputs.count;
    const batchCount = Math.ceil(totalCount / BATCH_SIZE);
    let allKeys = [];

    for (let batch = 0; batch < batchCount; batch++) {
      const batchNum = batch + 1;
      // 最后一批取余数，其余每批 BATCH_SIZE 个
      const currentBatchSize =
        batch === batchCount - 1
          ? totalCount - batch * BATCH_SIZE
          : BATCH_SIZE;

      setBatchProgress({ current: batchNum, total: batchCount });

      let res = await API.post(`/api/redemption/`, {
        ...localInputs,
        count: currentBatchSize,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        // 已有部分创建成功，提示用户下载
        if (allKeys.length > 0) {
          promptDownload(allKeys, localInputs.name, true);
        }
        setLoading(false);
        setBatchProgress(null);
        return;
      }
      if (data) {
        allKeys = allKeys.concat(data);
      }
    }

    setBatchProgress(null);
    showSuccess(
      t('兑换码创建成功！共创建 {{count}} 个', { count: allKeys.length }),
    );
    props.refresh();
    formApiRef.current?.setValues(getInitValues());
    props.handleClose();

    if (allKeys.length > 0) {
      promptDownload(allKeys, localInputs.name, false);
    }
    setLoading(false);
  };

  /**
   * 弹窗提示用户下载兑换码文本文件
   * @param {string[]} keys - 兑换码列表
   * @param {string} name - 文件名
   * @param {boolean} partial - 是否为部分创建（中间有批次失败）
   */
  const promptDownload = (keys, name, partial) => {
    let text = keys.join('\n');
    Modal.confirm({
      title: partial
        ? t('部分兑换码创建成功')
        : t('兑换码创建成功'),
      content: (
        <div>
          <p>
            {partial
              ? t(
                  '部分批次创建失败，已成功创建 {{count}} 个兑换码，是否下载？',
                  { count: keys.length },
                )
              : t('兑换码创建成功，是否下载兑换码？', {
                  count: keys.length,
                })}
          </p>
          <p>{t('兑换码将以文本文件的形式下载，文件名为兑换码的名称。')}</p>
        </div>
      ),
      onOk: () => {
        downloadTextAsFile(text, `${name}.txt`);
      },
    });
  };

  return (
    <>
      <SideSheet
        placement={isEdit ? 'right' : 'left'}
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
              {isEdit ? t('更新兑换码信息') : t('创建新的兑换码')}
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
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme='light'
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
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {({ values }) => (
              <div className='p-2'>
                <Card className='!rounded-2xl shadow-sm border-0 mb-6'>
                  {/* Header: Basic Info */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconGift size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置兑换码的基本信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='name'
                        label={t('名称')}
                        placeholder={t('请输入名称')}
                        style={{ width: '100%' }}
                        rules={
                          !isEdit
                            ? []
                            : [{ required: true, message: t('请输入名称') }]
                        }
                        showClear
                      />
                    </Col>
                    <Col span={24}>
                      <Form.DatePicker
                        field='expired_time'
                        label={t('过期时间')}
                        type='dateTime'
                        placeholder={t('选择过期时间（可选，留空为永久）')}
                        style={{ width: '100%' }}
                        showClear
                      />
                    </Col>
                  </Row>
                </Card>

                <Card className='!rounded-2xl shadow-sm border-0'>
                  {/* Header: Quota Settings */}
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='green'
                      className='mr-2 shadow-md'
                    >
                      <IconCreditCard size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('额度设置')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('设置兑换码的额度和数量')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={12}>
                      <Form.AutoComplete
                        field='quota'
                        label={t('额度')}
                        placeholder={t('请输入额度')}
                        style={{ width: '100%' }}
                        type='number'
                        rules={[
                          { required: true, message: t('请输入额度') },
                          {
                            validator: (rule, v) => {
                              const num = parseInt(v, 10);
                              return num > 0
                                ? Promise.resolve()
                                : Promise.reject(t('额度必须大于0'));
                            },
                          },
                        ]}
                        extraText={renderQuotaWithPrompt(
                          Number(values.quota) || 0,
                        )}
                        data={[
                          { value: 500000, label: '1$' },
                          { value: 5000000, label: '10$' },
                          { value: 25000000, label: '50$' },
                          { value: 50000000, label: '100$' },
                          { value: 250000000, label: '500$' },
                          { value: 500000000, label: '1000$' },
                        ]}
                        showClear
                      />
                    </Col>
                    {!isEdit && (
                      <Col span={12}>
                        <Form.InputNumber
                          field='count'
                          label={t('生成数量')}
                          min={1}
                          rules={[
                            { required: true, message: t('请输入生成数量') },
                            {
                              validator: (rule, v) => {
                                const num = parseInt(v, 10);
                                return num > 0
                                  ? Promise.resolve()
                                  : Promise.reject(t('生成数量必须大于0'));
                              },
                            },
                          ]}
                          extraText={
                            batchProgress
                              ? null
                              : t(
                                  '单次上限 {{max}} 个，超出将自动分批创建',
                                  { max: BATCH_SIZE },
                                )
                          }
                          style={{ width: '100%' }}
                          showClear
                        />
                      </Col>
                    )}
                  </Row>
                  {/* 分批创建进度条 */}
                  {batchProgress && (
                    <div style={{ marginTop: 12 }}>
                      <Progress
                        percent={
                          Math.round(
                            (batchProgress.current /
                              batchProgress.total) *
                              100,
                          )
                        }
                        showInfo
                        style={{ marginBottom: 4 }}
                      />
                      <Text type='tertiary' size='small'>
                        {t('正在创建第 {{current}} 批，共 {{total}} 批', {
                          current: batchProgress.current,
                          total: batchProgress.total,
                        })}
                      </Text>
                    </div>
                  )}
                </Card>
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>
    </>
  );
};

export default EditRedemptionModal;
