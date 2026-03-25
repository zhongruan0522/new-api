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

import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  Calendar,
  Button,
  Typography,
  Avatar,
  Spin,
  Tooltip,
  Modal,
} from '@douyinfe/semi-ui';
import { CalendarCheck, Gift, Check } from 'lucide-react';
import Turnstile from 'react-turnstile';
import { API, showError, showSuccess, renderQuota } from '../../../../helpers';

const CheckinCalendar = ({ t, status, turnstileEnabled, turnstileSiteKey }) => {
  const [loading, setLoading] = useState(false);
  const [checkinLoading, setCheckinLoading] = useState(false);
  const [turnstileModalVisible, setTurnstileModalVisible] = useState(false);
  const [turnstileWidgetKey, setTurnstileWidgetKey] = useState(0);
  const [checkinData, setCheckinData] = useState({
    enabled: false,
    stats: {
      checked_in_today: false,
      total_checkins: 0,
      total_quota: 0,
      checkin_count: 0,
      records: [],
    },
  });
  const [currentMonth, setCurrentMonth] = useState(
    new Date().toISOString().slice(0, 7),
  );
  const [initialLoaded, setInitialLoaded] = useState(false);

  const checkinRecordsMap = useMemo(() => {
    const map = {};
    const records = checkinData.stats?.records || [];
    records.forEach((record) => {
      map[record.checkin_date] = record.quota_awarded;
    });
    return map;
  }, [checkinData.stats?.records]);

  const monthlyQuota = useMemo(() => {
    const records = checkinData.stats?.records || [];
    return records.reduce(
      (sum, record) => sum + (record.quota_awarded || 0),
      0,
    );
  }, [checkinData.stats?.records]);

  const fetchCheckinStatus = async (month) => {
    const isFirstLoad = !initialLoaded;
    setLoading(true);
    try {
      const res = await API.get(`/api/user/checkin?month=${month}`);
      const { success, data, message } = res.data;
      if (success) {
        setCheckinData(data);
      } else {
        showError(message || t('获取签到状态失败'));
      }
    } catch (error) {
      showError(t('获取签到状态失败'));
    } finally {
      setLoading(false);
      if (isFirstLoad) setInitialLoaded(true);
    }
  };

  const postCheckin = async (token) => {
    const url = token
      ? `/api/user/checkin?turnstile=${encodeURIComponent(token)}`
      : '/api/user/checkin';
    return API.post(url);
  };

  const shouldTriggerTurnstile = (message) => {
    if (!turnstileEnabled) return false;
    if (typeof message !== 'string') return true;
    return message.includes('Turnstile');
  };

  const doCheckin = async (token) => {
    setCheckinLoading(true);
    try {
      const res = await postCheckin(token);
      const { success, data, message } = res.data;
      if (success) {
        showSuccess(
          t('签到成功！获得') + ' ' + renderQuota(data.quota_awarded),
        );
        fetchCheckinStatus(currentMonth);
        setTurnstileModalVisible(false);
      } else {
        if (!token && shouldTriggerTurnstile(message)) {
          if (!turnstileSiteKey) {
            showError('Turnstile is enabled but site key is empty.');
            return;
          }
          setTurnstileModalVisible(true);
          return;
        }
        if (token && shouldTriggerTurnstile(message)) {
          setTurnstileWidgetKey((v) => v + 1);
        }
        showError(message || t('签到失败'));
      }
    } catch (error) {
      showError(t('签到失败'));
    } finally {
      setCheckinLoading(false);
    }
  };

  useEffect(() => {
    if (status?.checkin_enabled) {
      fetchCheckinStatus(currentMonth);
    }
  }, [status?.checkin_enabled, currentMonth]);

  if (!status?.checkin_enabled) {
    return null;
  }

  // 日历日期渲染 - 显示签到标记和额度
  const dateRender = (dateString) => {
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return null;
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const formattedDate = `${year}-${month}-${day}`;
    const quotaAwarded = checkinRecordsMap[formattedDate];
    const isCheckedIn = quotaAwarded !== undefined;

    if (isCheckedIn) {
      return (
        <Tooltip
          content={`${t('获得')} ${renderQuota(quotaAwarded)}`}
          position='top'
        >
          <div className='absolute inset-0 flex flex-col items-center justify-center cursor-pointer'>
            <div className='w-6 h-6 rounded-full bg-green-500 flex items-center justify-center mb-0.5 shadow-sm'>
              <Check size={14} className='text-white' strokeWidth={3} />
            </div>
            <div className='text-[10px] font-medium text-green-600 dark:text-green-400 leading-none'>
              {renderQuota(quotaAwarded)}
            </div>
          </div>
        </Tooltip>
      );
    }
    return null;
  };

  const handleMonthChange = (date) => {
    const month = date.toISOString().slice(0, 7);
    setCurrentMonth(month);
  };

  const isCheckedInToday = checkinData.stats?.checked_in_today;
  const totalCheckins = checkinData.stats?.total_checkins || 0;

  return (
    <Card className='!rounded-2xl'>
      <Modal
        title='Security Check'
        visible={turnstileModalVisible}
        footer={null}
        centered
        onCancel={() => {
          setTurnstileModalVisible(false);
          setTurnstileWidgetKey((v) => v + 1);
        }}
      >
        <div className='flex justify-center py-2'>
          <Turnstile
            key={turnstileWidgetKey}
            sitekey={turnstileSiteKey}
            onVerify={(token) => {
              doCheckin(token);
            }}
            onExpire={() => {
              setTurnstileWidgetKey((v) => v + 1);
            }}
          />
        </div>
      </Modal>

      {/* 卡片头部 - 与其他卡片统一 */}
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-3 shadow-md'>
          <CalendarCheck size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('每日签到')}
          </Typography.Text>
          <div className='text-xs text-gray-600 dark:text-gray-400'>
            {!initialLoaded
              ? t('正在加载签到状态...')
              : isCheckedInToday
                ? t('今日已签到，累计签到') + ` ${totalCheckins} ` + t('天')
                : t('每日签到可获得随机额度奖励')}
          </div>
        </div>
      </div>

      {/* 签到操作卡片 - 与安全设置等卡片统一的子卡片风格 */}
      <Card className='!rounded-xl border dark:border-gray-700 mb-4'>
        <div className='flex flex-col sm:flex-row items-start sm:items-center sm:justify-between gap-4'>
          <div className='flex items-start w-full sm:w-auto'>
            <div className='w-12 h-12 rounded-full bg-green-50 dark:bg-green-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
              <Gift
                size={22}
                className='text-green-600 dark:text-green-400'
              />
            </div>
            <div>
              <Typography.Title heading={6} className='mb-1'>
                {isCheckedInToday ? t('今日已签到') : t('立即签到')}
              </Typography.Title>
              <Typography.Text type='tertiary' className='text-sm'>
                {t('每日签到可获得随机额度奖励')}
              </Typography.Text>
            </div>
          </div>
          <Button
            type='primary'
            theme='solid'
            icon={<Gift size={16} />}
            onClick={() => doCheckin()}
            loading={checkinLoading || !initialLoaded}
            disabled={!initialLoaded || isCheckedInToday}
            className='!bg-green-600 hover:!bg-green-700 w-full sm:w-auto'
          >
            {!initialLoaded
              ? t('加载中...')
              : isCheckedInToday
                ? t('今日已签到')
                : t('立即签到')}
          </Button>
        </div>
      </Card>

      {/* 统计数据 - 使用与安全设置卡片统一的列表布局 */}
      <div className='space-y-4'>
        <Card className='!rounded-xl border dark:border-gray-700'>
          <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
            <div className='flex items-start w-full sm:w-auto'>
              <div className='w-12 h-12 rounded-full bg-green-50 dark:bg-green-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
                <CalendarCheck
                  size={20}
                  className='text-green-600 dark:text-green-400'
                />
              </div>
              <div>
                <Typography.Title heading={6} className='mb-1'>
                  {t('累计签到')}
                </Typography.Title>
                <Typography.Text type='tertiary' className='text-sm'>
                  {t('签到奖励将直接添加到您的账户余额')}
                </Typography.Text>
              </div>
            </div>
            <div className='text-right flex-shrink-0'>
              <div className='text-2xl font-bold text-green-600 dark:text-green-400'>
                {totalCheckins}
              </div>
              <div className='text-xs text-gray-500'>{t('天')}</div>
            </div>
          </div>
        </Card>

        <Card className='!rounded-xl border dark:border-gray-700'>
          <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
            <div className='flex items-start w-full sm:w-auto'>
              <div className='w-12 h-12 rounded-full bg-orange-50 dark:bg-orange-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
                <Gift
                  size={20}
                  className='text-orange-600 dark:text-orange-400'
                />
              </div>
              <div>
                <Typography.Title heading={6} className='mb-1'>
                  {t('本月获得')}
                </Typography.Title>
                <Typography.Text type='tertiary' className='text-sm'>
                  {t('签到奖励将直接添加到您的账户余额')}
                </Typography.Text>
              </div>
            </div>
            <div className='text-right flex-shrink-0'>
              <div className='text-2xl font-bold text-orange-600 dark:text-orange-400'>
                {renderQuota(monthlyQuota, 6)}
              </div>
              <div className='text-xs text-gray-500'>{t('额度')}</div>
            </div>
          </div>
        </Card>

        <Card className='!rounded-xl border dark:border-gray-700'>
          <div className='flex flex-col sm:flex-row items-start sm:justify-between gap-4'>
            <div className='flex items-start w-full sm:w-auto'>
              <div className='w-12 h-12 rounded-full bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center mr-4 flex-shrink-0'>
                <Gift
                  size={20}
                  className='text-blue-600 dark:text-blue-400'
                />
              </div>
              <div>
                <Typography.Title heading={6} className='mb-1'>
                  {t('累计获得')}
                </Typography.Title>
                <Typography.Text type='tertiary' className='text-sm'>
                  {t('签到奖励将直接添加到您的账户余额')}
                </Typography.Text>
              </div>
            </div>
            <div className='text-right flex-shrink-0'>
              <div className='text-2xl font-bold text-blue-600 dark:text-blue-400'>
                {renderQuota(checkinData.stats?.total_quota || 0, 6)}
              </div>
              <div className='text-xs text-gray-500'>{t('额度')}</div>
            </div>
          </div>
        </Card>

        {/* 签到日历 */}
        <Card className='!rounded-xl border dark:border-gray-700'>
          <Spin spinning={loading}>
            <div className='checkin-calendar'>
              <Calendar
                mode='month'
                onChange={handleMonthChange}
                dateGridRender={(dateString, date) => dateRender(dateString)}
              />
            </div>
          </Spin>
        </Card>

        {/* 签到说明 */}
        <div className='px-1'>
          <Typography.Text type='tertiary' className='text-xs'>
            <ul className='list-disc list-inside space-y-0.5'>
              <li>{t('每日签到可获得随机额度奖励')}</li>
              <li>{t('签到奖励将直接添加到您的账户余额')}</li>
              <li>{t('每日仅可签到一次，请勿重复签到')}</li>
            </ul>
          </Typography.Text>
        </div>
      </div>
    </Card>
  );
};

export default CheckinCalendar;
