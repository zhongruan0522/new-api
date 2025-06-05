import React, { useEffect, useState, useContext } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  renderQuota,
  renderQuotaWithAmount,
  stringToColor,
  copy,
  getQuotaPerUnit
} from '../../helpers';
import {
  Layout,
  Typography,
  Card,
  Button,
  Modal,
  Toast,
  Input,
  InputNumber,
  Banner,
  Skeleton,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconGift,
  IconPlus,
  IconLink,
} from '@douyinfe/semi-icons';
import { SiAlipay, SiWechat } from 'react-icons/si';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status/index.js';

const { Text } = Typography;

const TopUp = () => {
  const { t } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const [redemptionCode, setRedemptionCode] = useState('');
  const [topUpCode, setTopUpCode] = useState('');
  const [amount, setAmount] = useState(0.0);
  const [minTopUp, setMinTopUp] = useState(statusState?.status?.min_topup || 1);
  const [topUpCount, setTopUpCount] = useState(statusState?.status?.min_topup || 1);
  const [topUpLink, setTopUpLink] = useState(statusState?.status?.top_up_link || '');
  const [enableOnlineTopUp, setEnableOnlineTopUp] = useState(statusState?.status?.enable_online_topup || false);
  const [userQuota, setUserQuota] = useState(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [payWay, setPayWay] = useState('');
  const [userDataLoading, setUserDataLoading] = useState(true);
  const [amountLoading, setAmountLoading] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  const [confirmLoading, setConfirmLoading] = useState(false);

  // 邀请相关状态
  const [affLink, setAffLink] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);

  const getUsername = () => {
    if (userState.user) {
      return userState.user.username;
    } else {
      return 'null';
    }
  };

  const getUserRole = () => {
    if (!userState.user) return t('普通用户');

    switch (userState.user.role) {
      case 100:
        return t('超级管理员');
      case 10:
        return t('管理员');
      case 0:
      default:
        return t('普通用户');
    }
  };

  const topUp = async () => {
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const res = await API.post('/api/user/topup', {
        key: redemptionCode,
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        Modal.success({
          title: t('兑换成功！'),
          content: t('成功兑换额度：') + renderQuota(data),
          centered: true,
        });
        setUserQuota((quota) => {
          return quota + data;
        });
        if (userState.user) {
          const updatedUser = {
            ...userState.user,
            quota: userState.user.quota + data
          };
          userDispatch({ type: 'login', payload: updatedUser });
        }
        setRedemptionCode('');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = () => {
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const preTopUp = async (payment) => {
    if (!enableOnlineTopUp) {
      showError(t('管理员未开启在线充值！'));
      return;
    }
    setPaymentLoading(true);
    try {
      await getAmount();
      if (topUpCount < minTopUp) {
        showError(t('充值数量不能小于') + minTopUp);
        return;
      }
      setPayWay(payment);
      setOpen(true);
    } catch (error) {
      showError(t('获取金额失败'));
    } finally {
      setPaymentLoading(false);
    }
  };

  const onlineTopUp = async () => {
    if (amount === 0) {
      await getAmount();
    }
    if (topUpCount < minTopUp) {
      showError('充值数量不能小于' + minTopUp);
      return;
    }
    setConfirmLoading(true);
    setOpen(false);
    try {
      const res = await API.post('/api/user/pay', {
        amount: parseInt(topUpCount),
        top_up_code: topUpCode,
        payment_method: payWay,
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          let params = data;
          let url = res.data.url;
          let form = document.createElement('form');
          form.action = url;
          form.method = 'POST';
          let isSafari =
            navigator.userAgent.indexOf('Safari') > -1 &&
            navigator.userAgent.indexOf('Chrome') < 1;
          if (!isSafari) {
            form.target = '_blank';
          }
          for (let key in params) {
            let input = document.createElement('input');
            input.type = 'hidden';
            input.name = key;
            input.value = params[key];
            form.appendChild(input);
          }
          document.body.appendChild(form);
          form.submit();
          document.body.removeChild(form);
        } else {
          showError(data);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
      showError(t('支付请求失败'));
    } finally {
      setConfirmLoading(false);
    }
  };

  const getUserQuota = async () => {
    setUserDataLoading(true);
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      setUserQuota(data.quota);
      userDispatch({ type: 'login', payload: data });
    } else {
      showError(message);
    }
    setUserDataLoading(false);
  };

  // 获取邀请链接
  const getAffLink = async () => {
    const res = await API.get('/api/user/aff');
    const { success, message, data } = res.data;
    if (success) {
      let link = `${window.location.origin}/register?aff=${data}`;
      setAffLink(link);
    } else {
      showError(message);
    }
  };

  // 划转邀请额度
  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post(`/api/user/aff_transfer`, {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota().then();
    } else {
      showError(message);
    }
  };

  // 复制邀请链接
  const handleAffLinkClick = async (e) => {
    e.target.select();
    await copy(e.target.value);
    showSuccess(t('邀请链接已复制到剪切板'));
  };

  useEffect(() => {
    if (userState?.user?.id) {
      setUserDataLoading(false);
      setUserQuota(userState.user.quota);
    } else {
      getUserQuota().then();
    }
    getAffLink().then();
    setTransferAmount(getQuotaPerUnit());
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      setMinTopUp(statusState.status.min_topup || 1);
      setTopUpCount(statusState.status.min_topup || 1);
      setTopUpLink(statusState.status.top_up_link || '');
      setEnableOnlineTopUp(statusState.status.enable_online_topup || false);
    }
  }, [statusState?.status]);

  const renderAmount = () => {
    return amount + ' ' + t('元');
  };

  const getAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/amount', {
        amount: parseFloat(value),
        top_up_code: topUpCode,
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setAmount(parseFloat(data));
        } else {
          setAmount(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      console.log(err);
    }
    setAmountLoading(false);
  };

  const handleCancel = () => {
    setOpen(false);
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  return (
    <div className="bg-gray-50">
      <Layout>
        <Layout.Content>
          {/* 划转模态框 */}
          <Modal
            title={
              <div className="flex items-center">
                <IconCreditCard className="mr-2" />
                {t('请输入要划转的数量')}
              </div>
            }
            visible={openTransfer}
            onOk={transfer}
            onCancel={handleTransferCancel}
            maskClosable={false}
            size={'small'}
            centered={true}
          >
            <div className="space-y-4 py-4">
              <div>
                <Typography.Text strong className="block mb-2">
                  {t('可用额度')} {renderQuota(userState?.user?.aff_quota)}
                </Typography.Text>
                <Input
                  value={userState?.user?.aff_quota}
                  disabled={true}
                  size="large"
                  className="!rounded-lg"
                />
              </div>
              <div>
                <Typography.Text strong className="block mb-2">
                  {t('划转额度')} {renderQuota(transferAmount)}{' '}
                  {t('最低') + renderQuota(getQuotaPerUnit())}
                </Typography.Text>
                <InputNumber
                  min={0}
                  value={transferAmount}
                  onChange={(value) => setTransferAmount(value)}
                  disabled={false}
                  size="large"
                  className="!rounded-lg w-full"
                />
              </div>
            </div>
          </Modal>

          <Modal
            title={
              <div className="flex items-center">
                <IconGift className="mr-2" />
                {t('充值确认')}
              </div>
            }
            visible={open}
            onOk={onlineTopUp}
            onCancel={handleCancel}
            maskClosable={false}
            size={'small'}
            centered={true}
            confirmLoading={confirmLoading}
          >
            <div className="space-y-3 py-4">
              <div className="flex justify-between">
                <Text strong>{t('充值数量')}：</Text>
                <Text>{topUpCount}</Text>
              </div>
              <div className="flex justify-between">
                <Text strong>{t('实付金额')}：</Text>
                {amountLoading ? (
                  <Skeleton.Title style={{ width: '60px', height: '16px' }} />
                ) : (
                  <Text type="danger">{renderAmount()}</Text>
                )}
              </div>
            </div>
          </Modal>

          <div className="flex justify-center">
            <div className="w-full">
              <Card className="!rounded-2xl shadow-lg border-0">
                <Card
                  className="!rounded-2xl !border-0 !shadow-2xl overflow-hidden"
                  style={{
                    background: 'linear-gradient(135deg, #1e3a8a 0%, #1e40af 25%, #2563eb 50%, #3b82f6 75%, #60a5fa 100%)',
                    position: 'relative'
                  }}
                  bodyStyle={{ padding: 0 }}
                >
                  <div className="absolute inset-0 overflow-hidden">
                    <div className="absolute -top-10 -right-10 w-40 h-40 bg-white opacity-5 rounded-full"></div>
                    <div className="absolute -bottom-16 -left-16 w-48 h-48 bg-white opacity-3 rounded-full"></div>
                    <div className="absolute top-1/2 right-1/4 w-24 h-24 bg-yellow-400 opacity-10 rounded-full"></div>
                  </div>

                  <div className="relative p-4 sm:p-6 md:p-8" style={{ color: 'white' }}>
                    <div className="flex justify-between items-start mb-4 sm:mb-6">
                      <div className="flex-1 min-w-0">
                        {userDataLoading ? (
                          <Skeleton.Title style={{ width: '200px', height: '20px' }} />
                        ) : (
                          <div className="text-base sm:text-lg font-semibold truncate" style={{ color: 'white' }}>
                            {t('尊敬的')} {getUsername()}
                          </div>
                        )}
                      </div>
                      <div
                        className="w-10 h-10 sm:w-12 sm:h-12 rounded-lg flex items-center justify-center shadow-lg flex-shrink-0 ml-2"
                        style={{
                          background: `linear-gradient(135deg, ${stringToColor(getUsername())} 0%, #f59e0b 100%)`
                        }}
                      >
                        <IconCreditCard size="default" style={{ color: 'white' }} />
                      </div>
                    </div>

                    <div className="mb-4 sm:mb-6">
                      <div className="text-xs sm:text-sm mb-1 sm:mb-2" style={{ color: 'rgba(255, 255, 255, 0.7)' }}>
                        {t('当前余额')}
                      </div>
                      {userDataLoading ? (
                        <Skeleton.Title style={{ width: '180px', height: '32px' }} />
                      ) : (
                        <div className="text-2xl sm:text-3xl md:text-4xl font-bold tracking-wide" style={{ color: 'white' }}>
                          {renderQuota(userState?.user?.quota || userQuota)}
                        </div>
                      )}
                    </div>

                    <div className="flex flex-col sm:flex-row sm:justify-between sm:items-end">
                      <div className="grid grid-cols-3 gap-2 sm:flex sm:space-x-6 lg:space-x-8 mb-3 sm:mb-0">
                        <div className="text-center sm:text-left">
                          <div className="text-xs" style={{ color: 'rgba(255, 255, 255, 0.6)' }}>
                            {t('历史消耗')}
                          </div>
                          {userDataLoading ? (
                            <Skeleton.Title style={{ width: '60px', height: '14px' }} />
                          ) : (
                            <div className="text-xs sm:text-sm font-medium truncate" style={{ color: 'white' }}>
                              {renderQuota(userState?.user?.used_quota || 0)}
                            </div>
                          )}
                        </div>
                        <div className="text-center sm:text-left">
                          <div className="text-xs" style={{ color: 'rgba(255, 255, 255, 0.6)' }}>
                            {t('用户分组')}
                          </div>
                          {userDataLoading ? (
                            <Skeleton.Title style={{ width: '50px', height: '14px' }} />
                          ) : (
                            <div className="text-xs sm:text-sm font-medium truncate" style={{ color: 'white' }}>
                              {userState?.user?.group || t('默认')}
                            </div>
                          )}
                        </div>
                        <div className="text-center sm:text-left">
                          <div className="text-xs" style={{ color: 'rgba(255, 255, 255, 0.6)' }}>
                            {t('用户角色')}
                          </div>
                          {userDataLoading ? (
                            <Skeleton.Title style={{ width: '60px', height: '14px' }} />
                          ) : (
                            <div className="text-xs sm:text-sm font-medium truncate" style={{ color: 'white' }}>
                              {getUserRole()}
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="self-end sm:self-auto">
                        {userDataLoading ? (
                          <Skeleton.Title style={{ width: '50px', height: '24px' }} />
                        ) : (
                          <div
                            className="px-2 py-1 sm:px-3 rounded-md text-xs sm:text-sm font-medium inline-block"
                            style={{
                              backgroundColor: 'rgba(255, 255, 255, 0.2)',
                              color: 'white',
                              backdropFilter: 'blur(10px)'
                            }}
                          >
                            ID: {userState?.user?.id || '---'}
                          </div>
                        )}
                      </div>
                    </div>

                    <div className="absolute top-0 left-0 w-full h-2 bg-gradient-to-r from-yellow-400 via-orange-400 to-red-400" style={{ opacity: 0.6 }}></div>
                  </div>
                </Card>

                <div className="p-6">
                  <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* 邀请信息部分 */}
                    <div>
                      <div className="flex items-center justify-between mb-6">
                        <div className="flex items-center">
                          <div className="w-12 h-12 rounded-full bg-orange-50 flex items-center justify-center mr-4">
                            <IconLink size="large" className="text-orange-500" />
                          </div>
                          <div>
                            <div className="flex items-center gap-3">
                              <Text className="text-xl font-semibold">{t('邀请信息')}</Text>
                              <Button
                                type="primary"
                                theme="solid"
                                onClick={() => setOpenTransfer(true)}
                                size="small"
                                className="!rounded-lg !bg-blue-500 hover:!bg-blue-600"
                                icon={<IconCreditCard />}
                              >
                                {t('划转')}
                              </Button>
                            </div>
                            <div className="text-gray-500 text-sm">{t('管理您的邀请链接和收益')}</div>
                          </div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                          <Card
                            className="!rounded-2xl text-center"
                            bodyStyle={{ padding: '16px' }}
                            shadows='hover'
                          >
                            <div className="text-gray-600 text-xs font-medium">{t('待使用收益')}</div>
                            <div className="text-gray-900 text-lg font-bold mt-1">
                              {renderQuota(userState?.user?.aff_quota)}
                            </div>

                          </Card>
                          <Card
                            className="!rounded-2xl text-center"
                            bodyStyle={{ padding: '16px' }}
                            shadows='hover'
                          >
                            <div className="text-gray-600 text-xs font-medium">{t('总收益')}</div>
                            <div className="text-gray-900 text-lg font-bold mt-1">
                              {renderQuota(userState?.user?.aff_history_quota)}
                            </div>
                          </Card>
                          <Card
                            className="!rounded-2xl text-center"
                            bodyStyle={{ padding: '16px' }}
                            shadows='hover'
                          >
                            <div className="text-gray-600 text-xs font-medium">{t('邀请人数')}</div>
                            <div className="text-gray-900 text-lg font-bold mt-1">
                              {userState?.user?.aff_count || 0}
                            </div>
                          </Card>
                        </div>

                        <div className="bg-white rounded-lg p-3">
                          <Typography.Text strong className="block mb-2 text-sm">{t('邀请链接')}</Typography.Text>
                          <Input
                            value={affLink}
                            onClick={handleAffLinkClick}
                            readOnly
                            size="large"
                            className="!rounded-lg"
                            prefix={<IconLink />}
                          />
                        </div>
                      </div>
                    </div>
                    <div>
                      <div className="flex items-center mb-6">
                        <div className="w-12 h-12 rounded-full bg-green-50 flex items-center justify-center mr-4">
                          <IconGift size="large" className="text-green-500" />
                        </div>
                        <div>
                          <Text className="text-xl font-semibold">{t('兑换余额')}</Text>
                          <div className="text-gray-500 text-sm">{t('使用兑换码充值余额')}</div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div>
                          <Text strong className="block mb-2">{t('兑换码')}</Text>
                          <Input
                            placeholder={t('请输入兑换码')}
                            value={redemptionCode}
                            onChange={(value) => setRedemptionCode(value)}
                            size="large"
                            className="!rounded-lg"
                            prefix={<IconGift />}
                          />
                        </div>

                        <div className="flex flex-col sm:flex-row gap-3">
                          {topUpLink && (
                            <Button
                              type="primary"
                              theme="solid"
                              onClick={openTopUpLink}
                              size="large"
                              className="!rounded-lg flex-1"
                              icon={<IconLink />}
                            >
                              {t('获取兑换码')}
                            </Button>
                          )}
                          <Button
                            type="warning"
                            theme="solid"
                            onClick={topUp}
                            disabled={isSubmitting}
                            loading={isSubmitting}
                            size="large"
                            className="!rounded-lg flex-1"
                          >
                            {isSubmitting ? t('兑换中...') : t('兑换')}
                          </Button>
                        </div>
                      </div>
                    </div>

                    <div>
                      <div className="flex items-center mb-6">
                        <div className="w-12 h-12 rounded-full bg-blue-50 flex items-center justify-center mr-4">
                          <IconPlus size="large" className="text-blue-500" />
                        </div>
                        <div>
                          <Text className="text-xl font-semibold">{t('在线充值')}</Text>
                          <div className="text-gray-500 text-sm">{t('支持多种支付方式')}</div>
                        </div>
                      </div>

                      <div className="space-y-4">
                        <div>
                          <div className="flex justify-between mb-2">
                            <Text strong>{t('充值数量')}</Text>
                            {amountLoading ? (
                              <Skeleton.Title style={{ width: '80px', height: '14px' }} />
                            ) : (
                              <Text type="tertiary">{t('实付金额：') + ' ' + renderAmount()}</Text>
                            )}
                          </div>
                          <InputNumber
                            disabled={!enableOnlineTopUp}
                            placeholder={
                              t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)
                            }
                            value={topUpCount}
                            min={minTopUp}
                            max={999999999}
                            step={1}
                            precision={0}
                            onChange={async (value) => {
                              if (value && value >= 1) {
                                setTopUpCount(value);
                                await getAmount(value);
                              }
                            }}
                            onBlur={(e) => {
                              const value = parseInt(e.target.value);
                              if (!value || value < 1) {
                                setTopUpCount(1);
                                getAmount(1);
                              }
                            }}
                            size="large"
                            className="!rounded-lg w-full"
                            prefix={<IconCreditCard />}
                            formatter={(value) => value ? `${value}` : ''}
                            parser={(value) => value ? parseInt(value.replace(/[^\d]/g, '')) : 0}
                          />
                        </div>

                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                          <Button
                            type="primary"
                            theme="solid"
                            onClick={async () => {
                              preTopUp('zfb');
                            }}
                            size="large"
                            className="!rounded-lg !bg-blue-500 hover:!bg-blue-600 h-14"
                            disabled={!enableOnlineTopUp}
                            loading={paymentLoading}
                            icon={<SiAlipay size={20} />}
                          >
                            <span className="ml-2">{t('支付宝')}</span>
                          </Button>
                          <Button
                            type="primary"
                            theme="solid"
                            onClick={async () => {
                              preTopUp('wx');
                            }}
                            size="large"
                            className="!rounded-lg !bg-green-500 hover:!bg-green-600 h-14"
                            disabled={!enableOnlineTopUp}
                            loading={paymentLoading}
                            icon={<SiWechat size={20} />}
                          >
                            <span className="ml-2">{t('微信')}</span>
                          </Button>
                        </div>

                        {!enableOnlineTopUp && (
                          <Banner
                            fullMode={false}
                            type="warning"
                            icon={null}
                            closeIcon={null}
                            className="!rounded-lg"
                            title={
                              <div style={{ fontWeight: 600, fontSize: '14px', lineHeight: '20px' }}>
                                {t('在线充值功能未开启')}
                              </div>
                            }
                            description={
                              <div>
                                {t('管理员未开启在线充值功能，请联系管理员开启或使用兑换码充值。')}
                              </div>
                            }
                          />
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </Card>
            </div>
          </div>
        </Layout.Content>
      </Layout>
    </div>
  );
};

export default TopUp;
