import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Modal, Tag, Tooltip } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import { createSectionTitle } from '../../helpers/dashboard';
import { TrendingUp, Eye } from 'lucide-react';

const WEEKDAY_MAP = {
  0: '周日',
  1: '周一',
  2: '周二',
  3: '周三',
  4: '周四',
  5: '周五',
  6: '周六',
};

/**
 * 将服务器时区的 HH:MM 时间转换为浏览器本地时区的 HH:MM
 * 服务器时区格式示例: "Asia/Shanghai", "UTC", "America/New_York"
 */
const convertTimeToLocal = (timeStr, serverTimezone) => {
  if (!timeStr || !serverTimezone) return timeStr;
  const [hours, minutes] = timeStr.split(':').map(Number);
  if (isNaN(hours) || isNaN(minutes)) return timeStr;

  try {
    // 构造一个"今天在服务器时区"的时间，然后转成本地时间
    const now = new Date();
    const serverDate = new Date(
      now.toLocaleString('en-US', { timeZone: serverTimezone }),
    );
    const localDate = new Date(
      now.toLocaleString('en-US', { timeZone: Intl.DateTimeFormat().resolvedOptions().timeZone }),
    );
    const offsetMs = localDate.getTime() - serverDate.getTime();

    const serverMinutes = hours * 60 + minutes;
    const localMinutes = serverMinutes + Math.round(offsetMs / 60000);

    const normalized = ((localMinutes % 1440) + 1440) % 1440;
    const h = Math.floor(normalized / 60);
    const m = normalized % 60;
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
  } catch {
    return timeStr;
  }
};

const DynamicRatioCard = () => {
  const { t } = useTranslation();
  const [status, setStatus] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);

  const loadStatus = useCallback(async () => {
    try {
      const res = await API.get('/api/dynamic_ratio/status');
      const { success, data, message } = res.data;
      if (success) {
        setStatus(data);
      } else {
        showError(message);
      }
    } catch {
      showError(t('动态倍率状态加载失败'));
    }
  }, [t]);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const formatWeekdays = useCallback(
    (weekdays) => {
      if (!weekdays) return '';
      try {
        const values = JSON.parse(weekdays);
        if (!Array.isArray(values) || values.length === 0) {
          return t('每天');
        }
        return values.map((day) => t(WEEKDAY_MAP[day] || day)).join('、');
      } catch {
        return weekdays;
      }
    },
    [t],
  );

  // 计算所有规则中的最高倍率
  const maxRatio = useMemo(() => {
    if (!status?.rules || status.rules.length === 0) return null;
    let max = 0;
    for (const rule of status.rules) {
      if (rule.ratio > max) max = rule.ratio;
    }
    return max > 0 ? max : null;
  }, [status]);

  // 当前生效倍率
  const activeRatio = status ? Number(status.active_ratio) : 1;
  const hasActiveRatio = Number.isFinite(activeRatio) && activeRatio !== 1;

  // 判断本地时区是否与服务器时区不同
  const localTimezone = useMemo(() => {
    try {
      return Intl.DateTimeFormat().resolvedOptions().timeZone;
    } catch {
      return '';
    }
  }, []);
  const timezoneDiffers =
    status?.timezone && localTimezone && status.timezone !== localTimezone;

  if (!status || !status.enabled) {
    return null;
  }

  const ratioColor = hasActiveRatio
    ? activeRatio >= 2
      ? '#ef4444'
      : '#f59e0b'
    : '#10b981';

  return (
    <>
      <Card
        className='bg-orange-50 border-0 !rounded-2xl w-full'
        title={createSectionTitle(TrendingUp, t('动态倍率'))}
        style={{ boxShadow: 'none', bordered: true, headerLine: true }}
      >
        <div className='space-y-4'>
          {/* 第一行：当前最高倍率 */}
          <div className='flex items-center justify-between'>
            <div className='flex items-center'>
              <div
                className='mr-3 flex items-center justify-center w-8 h-8 rounded-full'
                style={{ backgroundColor: ratioColor + '20' }}
              >
                <span
                  className='inline-block w-3 h-3 rounded-full'
                  style={{ backgroundColor: ratioColor }}
                />
              </div>
              <div>
                <div className='text-xs text-gray-500'>
                  {t('当前最高倍率可达')}
                </div>
                <div
                  className='text-lg font-semibold'
                  style={{ color: ratioColor }}
                >
                  {maxRatio != null ? `${maxRatio}x` : '--'}
                </div>
              </div>
            </div>
            {hasActiveRatio && (
              <Tooltip content={t('当前倍率')}>
                <Tag
                  color={activeRatio >= 2 ? 'red' : 'orange'}
                  size='large'
                  shape='circle'
                >
                  {activeRatio}x
                </Tag>
              </Tooltip>
            )}
          </div>

          {/* 第二行：查看全部规则按钮 */}
          {status.rules_count > 0 && (
            <div>
              <Button
                icon={<Eye size={14} />}
                size='small'
                onClick={() => setModalVisible(true)}
                block
              >
                {t('查看全部规则')}
              </Button>
            </div>
          )}
        </div>
      </Card>

      {/* 查看全部规则 Modal */}
      <Modal
        title={t('动态倍率规则')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={560}
      >
        <div className='space-y-4'>
          {/* 时区信息 */}
          {status.timezone && (
            <div
              className='flex items-center justify-between px-4 py-3 rounded-xl'
              style={{
                backgroundColor: 'var(--semi-color-fill-0)',
                border: '1px solid var(--semi-color-border)',
              }}
            >
              <span
                className='text-sm'
                style={{ color: 'var(--semi-color-text-2)' }}
              >
                {t('触发时区')}
              </span>
              <div className='flex items-center gap-2'>
                <Tag size='small' shape='circle'>
                  {status.timezone}
                </Tag>
                {timezoneDiffers && (
                  <Tag size='small' shape='circle' color='blue'>
                    {t('已转换为本地时区')}
                  </Tag>
                )}
              </div>
            </div>
          )}

          {/* 规则列表 */}
          {status.rules && status.rules.length > 0 ? (
            <div className='space-y-3'>
              {status.rules.map((rule, idx) => {
                const localStart =
                  timezoneDiffers && rule.start_time
                    ? convertTimeToLocal(rule.start_time, status.timezone)
                    : rule.start_time;
                const localEnd =
                  timezoneDiffers && rule.end_time
                    ? convertTimeToLocal(rule.end_time, status.timezone)
                    : rule.end_time;
                const hasTimeRange = rule.start_time || rule.end_time;
                const weekdayText = formatWeekdays(rule.weekdays);
                const isEveryDay = weekdayText === t('每天');

                return (
                  <div
                    key={idx}
                    className='px-4 py-3 rounded-xl'
                    style={{
                      backgroundColor: 'var(--semi-color-fill-0)',
                      border: '1px solid var(--semi-color-border)',
                    }}
                  >
                    {/* 规则头部：分组 + 倍率 */}
                    <div className='flex items-center justify-between mb-2'>
                      <div className='flex items-center gap-2'>
                        <Tag size='small' shape='circle'>
                          {rule.group}
                        </Tag>
                        <span
                          className='text-xs'
                          style={{ color: 'var(--semi-color-text-2)' }}
                        >
                          {t('优先级')} {rule.priority}
                        </span>
                      </div>
                      <Tag
                        color={
                          rule.ratio >= 2
                            ? 'red'
                            : rule.ratio > 1
                              ? 'orange'
                              : 'green'
                        }
                        size='large'
                      >
                        {rule.ratio}x
                      </Tag>
                    </div>

                    {/* 规则条件标签 */}
                    <div className='flex flex-wrap items-center gap-2'>
                      {rule.concurrency && (
                        <Tag color='blue' size='small' shape='circle'>
                          {t('并发')} &gt; {rule.concurrency}
                        </Tag>
                      )}
                      {!isEveryDay && (
                        <Tag color='cyan' size='small' shape='circle'>
                          {weekdayText}
                        </Tag>
                      )}
                      {hasTimeRange && (
                        <Tag color='purple' size='small' shape='circle'>
                          {localStart || '00:00'} - {localEnd || '23:59'}
                        </Tag>
                      )}
                      {!rule.concurrency && isEveryDay && !hasTimeRange && (
                        <span
                          className='text-xs'
                          style={{ color: 'var(--semi-color-text-2)' }}
                        >
                          {t('不限')}
                        </span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div
              className='text-center py-6'
              style={{ color: 'var(--semi-color-text-2)' }}
            >
              {t('暂无启用的规则')}
            </div>
          )}
        </div>
      </Modal>
    </>
  );
};

export default DynamicRatioCard;
