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

import React from 'react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Input,
  InputNumber,
  Banner,
  Skeleton,
  Divider,
  Tabs,
  TabPane,
} from '@douyinfe/semi-ui';
import { SiAlipay, SiWechat, SiStripe } from 'react-icons/si';
import { CreditCard, Gift, Link as LinkIcon, Coins } from 'lucide-react';
import { IconGift } from '@douyinfe/semi-icons';
import RightStatsCard from './RightStatsCard';

const { Text } = Typography;

const RechargeCard = ({
  t,
  enableOnlineTopUp,
  enableStripeTopUp,
  presetAmounts,
  selectedPreset,
  selectPresetAmount,
  formatLargeNumber,
  priceRatio,
  topUpCount,
  minTopUp,
  renderQuotaWithAmount,
  getAmount,
  setTopUpCount,
  setSelectedPreset,
  renderAmount,
  amountLoading,
  payMethods,
  preTopUp,
  paymentLoading,
  payWay,
  redemptionCode,
  setRedemptionCode,
  topUp,
  isSubmitting,
  topUpLink,
  openTopUpLink,
  // 新增：用于右侧统计卡片
  userState,
  renderQuota,
  statusLoading,
}) => {
  return (
    <Card className="!rounded-2xl shadow-sm border-0">
      {/* 卡片头部 */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-4 gap-3">
        <div className="flex items-center">
          <Avatar size="small" color="blue" className="mr-3 shadow-md">
            <CreditCard size={16} />
          </Avatar>
          <div>
            <Typography.Text className="text-lg font-medium">{t('账户充值')}</Typography.Text>
            <div className="text-xs text-gray-600 dark:text-gray-400">{t('多种充值方式，安全便捷')}</div>
          </div>
        </div>
        <RightStatsCard t={t} userState={userState} renderQuota={renderQuota} />
      </div>

      <Tabs type="card" defaultActiveKey="online">
        {/* 在线充值 Tab */}
        <TabPane
          tab={
            <div className="flex items-center">
              <CreditCard size={16} className="mr-2" />
              {t('在线充值')}
            </div>
          }
          itemKey="online"
        >
          <div className="py-4">
            {statusLoading ? (
              <div className='py-8 flex justify-center'>
                <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500'></div>
              </div>
            ) : (enableOnlineTopUp || enableStripeTopUp) ? (
              <div className='space-y-6'>
                {/* 预设充值额度选择 */}
                {(enableOnlineTopUp || enableStripeTopUp) && (
                  <div>
                    <Text strong className='block mb-3'>
                      {t('选择充值额度')}
                    </Text>
                    <div className='grid grid-cols-2 sm:grid-cols-4 gap-3'>
                      {presetAmounts.map((preset, index) => (
                        <Card
                          key={index}
                          onClick={() => selectPresetAmount(preset)}
                          className={`cursor-pointer !rounded-xl transition-all hover:shadow-md ${selectedPreset === preset.value
                            ? 'border-blue-500 shadow-md'
                            : 'border-slate-200 hover:border-slate-300 dark:border-slate-600 dark:hover:border-slate-500'
                            }`}
                          bodyStyle={{ textAlign: 'center', padding: '12px' }}
                        >
                          <div className='font-medium text-lg flex items-center justify-center mb-1'>
                            <Coins size={16} className='mr-1' />
                            {formatLargeNumber(preset.value)}
                          </div>
                          <div className='text-xs text-gray-500 dark:text-gray-400'>
                            {t('实付')} ￥{(preset.value * priceRatio).toFixed(2)}
                          </div>
                        </Card>
                      ))}
                    </div>
                  </div>
                )}

                {/* 自定义充值金额 */}
                {(enableOnlineTopUp || enableStripeTopUp) && (
                  <div className='space-y-4'>
                    <Divider style={{ margin: '24px 0' }}>
                      <Text className='text-sm font-medium text-slate-600 dark:text-slate-400'>
                        {t('或输入自定义金额')}
                      </Text>
                    </Divider>

                    <div>
                      <div className='flex justify-between mb-2'>
                        <Text strong className='text-slate-700 dark:text-slate-200'>{t('充值数量')}</Text>
                        {amountLoading ? (
                          <Skeleton.Title style={{ width: '80px', height: '16px' }} />
                        ) : (
                          <Text className='text-red-600 font-semibold'>
                            {t('实付金额：')}<span className='font-bold' style={{ color: 'red' }}>{renderAmount()}</span>
                          </Text>
                        )}
                      </div>
                      <InputNumber
                        disabled={!enableOnlineTopUp && !enableStripeTopUp}
                        placeholder={t('充值数量，最低 ') + renderQuotaWithAmount(minTopUp)}
                        value={topUpCount}
                        min={minTopUp}
                        max={999999999}
                        step={1}
                        precision={0}
                        onChange={async (value) => {
                          if (value && value >= 1) {
                            setTopUpCount(value);
                            setSelectedPreset(null);
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
                        className='w-full !rounded-lg'
                        formatter={(value) => (value ? `${value}` : '')}
                        parser={(value) => value ? parseInt(value.replace(/[^\d]/g, '')) : 0}
                      />
                    </div>

                    {/* 支付方式选择 */}
                    <div>
                      <Text strong className='block mb-3 text-slate-700 dark:text-slate-200'>
                        {t('选择支付方式')}
                      </Text>
                      <div className={`grid gap-3 ${payMethods.length <= 2
                        ? 'grid-cols-1 sm:grid-cols-2'
                        : payMethods.length === 3
                          ? 'grid-cols-1 sm:grid-cols-3'
                          : 'grid-cols-2 sm:grid-cols-4'
                        }`}>
                        {payMethods.map((payMethod) => (
                          <Card
                            key={payMethod.type}
                            onClick={() => preTopUp(payMethod.type)}
                            className={`cursor-pointer !rounded-xl transition-all hover:shadow-md ${paymentLoading && payWay === payMethod.type
                              ? 'border-blue-500 shadow-md'
                              : 'border-slate-200 hover:border-slate-300 dark:border-slate-600 dark:hover:border-slate-500'
                              } ${(!enableOnlineTopUp && payMethod.type !== 'stripe') ||
                                (!enableStripeTopUp && payMethod.type === 'stripe')
                                ? 'opacity-50 cursor-not-allowed'
                                : ''
                              }`}
                            bodyStyle={{ padding: '12px', textAlign: 'center' }}
                          >
                            {paymentLoading && payWay === payMethod.type ? (
                              <div className='flex flex-col items-center justify-center'>
                                <div className='mb-2'>
                                  <div className='animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500'></div>
                                </div>
                                <div className='text-xs text-slate-500 dark:text-slate-400'>{t('处理中')}</div>
                              </div>
                            ) : (
                              <>
                                <div className='flex items-center justify-center mb-2'>
                                  {payMethod.type === 'zfb' ? (
                                    <SiAlipay size={24} color="#1677FF" />
                                  ) : payMethod.type === 'wx' ? (
                                    <SiWechat size={24} color="#07C160" />
                                  ) : payMethod.type === 'stripe' ? (
                                    <SiStripe size={24} color="#635BFF" />
                                  ) : (
                                    <CreditCard size={24} color={payMethod.color || 'var(--semi-color-text-2)'} />
                                  )}
                                </div>
                                <div className='text-sm font-medium text-slate-700 dark:text-slate-200'>{payMethod.name}</div>
                              </>
                            )}
                          </Card>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Banner
                type='warning'
                description={t('管理员未开启在线充值功能，请联系管理员开启或使用兑换码充值。')}
                className='!rounded-xl'
                closeIcon={null}
              />
            )}
          </div>
        </TabPane>

        {/* 兑换码充值 Tab */}
        <TabPane
          tab={
            <div className="flex items-center">
              <Gift size={16} className="mr-2" />
              {t('兑换码充值')}
            </div>
          }
          itemKey="redeem"
        >
          <div className="py-4">
            <div className='space-y-4'>
              <Input
                placeholder={t('请输入兑换码')}
                value={redemptionCode}
                onChange={(value) => setRedemptionCode(value)}
                className='!rounded-lg'
                prefix={<IconGift />}
                showClear
              />

              <div className='flex flex-col sm:flex-row gap-2'>
                {topUpLink && (
                  <Button
                    type='secondary'
                    theme='outline'
                    onClick={openTopUpLink}
                    className='flex-1 !rounded-lg !border-slate-300 !text-slate-600 hover:!border-slate-400 hover:!text-slate-700'
                    icon={<LinkIcon size={16} />}
                  >
                    {t('获取兑换码')}
                  </Button>
                )}
                <Button
                  type='primary'
                  theme='solid'
                  onClick={topUp}
                  disabled={isSubmitting || !redemptionCode}
                  loading={isSubmitting}
                  className='flex-1 !rounded-lg !bg-slate-600 hover:!bg-slate-700'
                >
                  {isSubmitting ? t('兑换中...') : t('兑换')}
                </Button>
              </div>
            </div>
          </div>
        </TabPane>
      </Tabs>
    </Card>
  );
};

export default RechargeCard;
