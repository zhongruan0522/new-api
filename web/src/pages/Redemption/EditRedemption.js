import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  downloadTextAsFile,
  isMobile,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaWithPrompt
} from '../../helpers';
import {
  AutoComplete,
  Button,
  Input,
  Modal,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  DatePicker,
  Avatar,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconSave,
  IconClose,
  IconPlusCircle,
  IconGift,
} from '@douyinfe/semi-icons';

const { Text, Title } = Typography;

const EditRedemption = (props) => {
  const { t } = useTranslation();
  const isEdit = props.editingRedemption.id !== undefined;
  const [loading, setLoading] = useState(isEdit);

  const originInputs = {
    name: '',
    quota: 100000,
    count: 1,
    expired_time: 0,
  };
  const [inputs, setInputs] = useState(originInputs);
  const { name, quota, count, expired_time } = inputs;

  const handleCancel = () => {
    props.handleClose();
  };

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const loadRedemption = async () => {
    setLoading(true);
    let res = await API.get(`/api/redemption/${props.editingRedemption.id}`);
    const { success, message, data } = res.data;
    if (success) {
      setInputs(data);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (isEdit) {
      loadRedemption().then(() => {
        // console.log(inputs);
      });
    } else {
      setInputs(originInputs);
    }
  }, [props.editingRedemption.id]);

  const submit = async () => {
    let name = inputs.name;
    if (!isEdit && inputs.name === '') {
      // set default name
      name = renderQuota(quota);
    }
    setLoading(true);
    let localInputs = inputs;
    localInputs.count = parseInt(localInputs.count);
    localInputs.quota = parseInt(localInputs.quota);
    localInputs.name = name;
    if (localInputs.expired_time === null || localInputs.expired_time === undefined) {
      localInputs.expired_time = 0;
    }
    let res;
    if (isEdit) {
      res = await API.put(`/api/redemption/`, {
        ...localInputs,
        id: parseInt(props.editingRedemption.id),
      });
    } else {
      res = await API.post(`/api/redemption/`, {
        ...localInputs,
      });
    }
    const { success, message, data } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('兑换码更新成功！'));
        props.refresh();
        props.handleClose();
      } else {
        showSuccess(t('兑换码创建成功！'));
        setInputs(originInputs);
        props.refresh();
        props.handleClose();
      }
    } else {
      showError(message);
    }
    if (!isEdit && data) {
      let text = '';
      for (let i = 0; i < data.length; i++) {
        text += data[i] + '\n';
      }
      Modal.confirm({
        title: t('兑换码创建成功'),
        content: (
          <div>
            <p>{t('兑换码创建成功，是否下载兑换码？')}</p>
            <p>{t('兑换码将以文本文件的形式下载，文件名为兑换码的名称。')}</p>
          </div>
        ),
        onOk: () => {
          downloadTextAsFile(text, `${inputs.name}.txt`);
        },
      });
    }
    setLoading(false);
  };

  return (
    <>
      <SideSheet
        placement={isEdit ? 'right' : 'left'}
        title={
          <Space>
            {isEdit ?
              <Tag color="blue" shape="circle">{t('更新')}</Tag> :
              <Tag color="green" shape="circle">{t('新建')}</Tag>
            }
            <Title heading={4} className="m-0">
              {isEdit ? t('更新兑换码信息') : t('创建新的兑换码')}
            </Title>
          </Space>
        }
        headerStyle={{
          borderBottom: '1px solid var(--semi-color-border)',
          padding: '24px'
        }}
        bodyStyle={{ padding: '0' }}
        visible={props.visiable}
        width={isMobile() ? '100%' : 600}
        footer={
          <div className="flex justify-end bg-white">
            <Space>
              <Button
                theme="solid"
                className="!rounded-full"
                onClick={submit}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme="light"
                className="!rounded-full"
                type="primary"
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
          <div className="p-6">
            <Card className="!rounded-2xl shadow-sm border-0 mb-6">
              {/* Header: Basic Info */}
              <div className="flex items-center mb-2">
                <Avatar size="small" color="blue" className="mr-2 shadow-md">
                  <IconGift size={16} />
                </Avatar>
                <div>
                  <Text className="text-lg font-medium">{t('基本信息')}</Text>
                  <div className="text-xs text-gray-600">{t('设置兑换码的基本信息')}</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <Text strong className="block mb-2">{t('名称')}</Text>
                  <Input
                    placeholder={t('请输入名称')}
                    onChange={(value) => handleInputChange('name', value)}
                    value={name}
                    autoComplete="new-password"
                    className="!rounded-lg"
                    showClear
                    required={!isEdit}
                  />
                </div>
                <div>
                  <Text strong className="block mb-2">{t('过期时间')}</Text>
                  <DatePicker
                    type="dateTime"
                    placeholder={t('选择过期时间（可选，留空为永久）')}
                    showClear
                    value={expired_time ? new Date(expired_time * 1000) : null}
                    onChange={(value) => {
                      if (value === null || value === undefined) {
                        handleInputChange('expired_time', 0);
                      } else {
                        const timestamp = Math.floor(value.getTime() / 1000);
                        handleInputChange('expired_time', timestamp);
                      }
                    }}
                    className="!rounded-lg w-full"
                  />
                </div>
              </div>
            </Card>

            <Card className="!rounded-2xl shadow-sm border-0">
              {/* Header: Quota Settings */}
              <div className="flex items-center mb-2">
                <Avatar size="small" color="green" className="mr-2 shadow-md">
                  <IconCreditCard size={16} />
                </Avatar>
                <div>
                  <Text className="text-lg font-medium">{t('额度设置')}</Text>
                  <div className="text-xs text-gray-600">{t('设置兑换码的额度和数量')}</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <div className="flex justify-between mb-2">
                    <Text strong>{t('额度')}</Text>
                    <Text type="tertiary">{renderQuotaWithPrompt(quota)}</Text>
                  </div>
                  <AutoComplete
                    placeholder={t('请输入额度')}
                    onChange={(value) => handleInputChange('quota', value)}
                    value={quota}
                    autoComplete="new-password"
                    type="number"
                    className="w-full !rounded-lg"
                    prefix={<IconCreditCard />}
                    data={[
                      { value: 500000, label: '1$' },
                      { value: 5000000, label: '10$' },
                      { value: 25000000, label: '50$' },
                      { value: 50000000, label: '100$' },
                      { value: 250000000, label: '500$' },
                      { value: 500000000, label: '1000$' },
                    ]}
                  />
                </div>

                {!isEdit && (
                  <div>
                    <Text strong className="block mb-2">{t('生成数量')}</Text>
                    <Input
                      placeholder={t('请输入生成数量')}
                      onChange={(value) => handleInputChange('count', value)}
                      value={count}
                      autoComplete="new-password"
                      type="number"
                      className="!rounded-lg"
                      prefix={<IconPlusCircle />}
                    />
                  </div>
                )}
              </div>
            </Card>
          </div>
        </Spin>
      </SideSheet>
    </>
  );
};

export default EditRedemption;
