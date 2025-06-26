import React, { useState } from 'react';
import { API, isMobile, showError, showSuccess } from '../../helpers';
import {
  Button,
  Input,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Avatar
} from '@douyinfe/semi-ui';
import {
  IconUser,
  IconSave,
  IconClose,
  IconKey,
  IconUserAdd,
  IconEdit,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const AddUser = (props) => {
  const { t } = useTranslation();
  const originInputs = {
    username: '',
    display_name: '',
    password: '',
    remark: '',
  };
  const [inputs, setInputs] = useState(originInputs);
  const [loading, setLoading] = useState(false);
  const { username, display_name, password, remark } = inputs;

  const handleInputChange = (name, value) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const submit = async () => {
    setLoading(true);
    if (inputs.username === '' || inputs.password === '') {
      setLoading(false);
      showError(t('用户名和密码不能为空！'));
      return;
    }
    const res = await API.post(`/api/user/`, inputs);
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('用户账户创建成功！'));
      setInputs(originInputs);
      props.refresh();
      props.handleClose();
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleCancel = () => {
    props.handleClose();
  };

  return (
    <>
      <SideSheet
        placement={'left'}
        title={
          <Space>
            <Tag color="green" shape="circle">{t('新建')}</Tag>
            <Title heading={4} className="m-0">
              {t('添加用户')}
            </Title>
          </Space>
        }
        headerStyle={{
          borderBottom: '1px solid var(--semi-color-border)',
          padding: '24px'
        }}
        bodyStyle={{ padding: '0' }}
        visible={props.visible}
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
            <Card className="!rounded-2xl shadow-sm border-0">
              <div className="flex items-center mb-2">
                <Avatar size="small" color="blue" className="mr-2 shadow-md">
                  <IconUserAdd size={16} />
                </Avatar>
                <div>
                  <Text className="text-lg font-medium">{t('用户信息')}</Text>
                  <div className="text-xs text-gray-600">{t('创建新用户账户')}</div>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <Text strong className="block mb-2">{t('用户名')}</Text>
                  <Input
                    placeholder={t('请输入用户名')}
                    onChange={(value) => handleInputChange('username', value)}
                    value={username}
                    autoComplete="off"
                    className="!rounded-lg"
                    prefix={<IconUser />}
                    showClear
                    required
                  />
                </div>

                <div>
                  <Text strong className="block mb-2">{t('显示名称')}</Text>
                  <Input
                    placeholder={t('请输入显示名称')}
                    onChange={(value) => handleInputChange('display_name', value)}
                    value={display_name}
                    autoComplete="off"
                    className="!rounded-lg"
                    prefix={<IconUser />}
                    showClear
                  />
                </div>

                <div>
                  <Text strong className="block mb-2">{t('密码')}</Text>
                  <Input
                    type="password"
                    placeholder={t('请输入密码')}
                    onChange={(value) => handleInputChange('password', value)}
                    value={password}
                    autoComplete="off"
                    className="!rounded-lg"
                    prefix={<IconKey />}
                    required
                  />
                </div>

                <div>
                  <Text strong className="block mb-2">{t('备注')}</Text>
                  <Input
                    placeholder={t('请输入备注（仅管理员可见）')}
                    onChange={(value) => handleInputChange('remark', value)}
                    value={remark}
                    autoComplete="off"
                    className="!rounded-lg"
                    prefix={<IconEdit />}
                    showClear
                  />
                </div>
              </div>
            </Card>
          </div>
        </Spin>
      </SideSheet>
    </>
  );
};

export default AddUser;
