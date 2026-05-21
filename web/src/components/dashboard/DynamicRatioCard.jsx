import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Tag, Button, Modal } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import { AlertTriangle, CheckCircle, Eye } from 'lucide-react';

const DynamicRatioCard = () => {
  const { t } = useTranslation();

  const WEEKDAY_LABELS = {
    0: t('周日'),
    1: t('周一'),
    2: t('周二'),
    3: t('周三'),
    4: t('周四'),
    5: t('周五'),
    6: t('周六'),
  };
  const [status, setStatus] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);

  const formatWeekdays = useCallback(
    (weekdays) => {
      if (!weekdays) {
        return '';
      }
      try {
        const values = JSON.parse(weekdays);
        if (!Array.isArray(values) || values.length === 0) {
          return t('每天');
        }
        return values.map((day) => WEEKDAY_LABELS[day] || day).join('、');
      } catch {
        return weekdays;
      }
    },
    [t],
  );

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

  if (!status || !status.enabled) {
    return null;
  }

  const activeRatio = Number(status.active_ratio);
  const hasActiveRatio = Number.isFinite(activeRatio) && activeRatio !== 1;
  const isHighRatio = Number.isFinite(activeRatio) && activeRatio >= 2;

  const cardStyle = hasActiveRatio
    ? {
        background: isHighRatio
          ? 'linear-gradient(135deg, #fff5f5 0%, #ffe0e0 100%)'
          : 'linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%)',
        border: `1px solid ${isHighRatio ? '#fca5a5' : '#fcd34d'}`,
      }
    : {
        background: 'linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%)',
        border: '1px solid #86efac',
      };

  return (
    <>
      <Card
        className='!rounded-2xl border-0'
        style={cardStyle}
        bodyStyle={{ padding: '16px 20px' }}
      >
        <div className='flex items-center justify-between'>
          <div className='flex items-center gap-3'>
            {hasActiveRatio ? (
              <AlertTriangle
                size={20}
                color={isHighRatio ? '#dc2626' : '#d97706'}
              />
            ) : (
              <CheckCircle size={20} color='#16a34a' />
            )}
            <div>
              <div
                className='text-sm font-semibold'
                style={{ color: 'var(--semi-color-text-0)' }}
              >
                {hasActiveRatio
                  ? t('当前动态倍率生效中')
                  : t('当前无动态倍率生效')}
              </div>
              {hasActiveRatio && (
                <div className='flex items-center gap-2 mt-1'>
                  <Tag
                    color={isHighRatio ? 'red' : 'orange'}
                    size='large'
                    shape='circle'
                  >
                    {status.active_ratio}x
                  </Tag>
                  <span
                    className='text-xs'
                    style={{ color: 'var(--semi-color-text-2)' }}
                  >
                    {t('当前倍率')}
                  </span>
                </div>
              )}
            </div>
          </div>
          {status.rules_count > 0 && (
            <Button
              icon={<Eye size={14} />}
              size='small'
              onClick={() => setModalVisible(true)}
            >
              {t('查看全部规则')}
            </Button>
          )}
        </div>
      </Card>

      <Modal
        title={t('动态倍率规则')}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={520}
      >
        <div className='space-y-3'>
          {status.rules && status.rules.length > 0 ? (
            status.rules.map((rule, idx) => (
              <div
                key={idx}
                className='flex items-center justify-between p-3 rounded-lg'
                style={{
                  backgroundColor: 'var(--semi-color-fill-0)',
                  border: '1px solid var(--semi-color-border)',
                }}
              >
                <div className='flex-1'>
                  <div className='flex items-center gap-2'>
                    <Tag size='small' shape='circle'>
                      {rule.group}
                    </Tag>
                    {rule.concurrency && (
                      <Tag color='blue' size='small' shape='circle'>
                        {t('并发')} &gt; {rule.concurrency}
                      </Tag>
                    )}
                    {rule.weekdays &&
                      formatWeekdays(rule.weekdays) !== t('每天') && (
                        <Tag color='cyan' size='small' shape='circle'>
                          {formatWeekdays(rule.weekdays)}
                        </Tag>
                      )}
                    {(rule.start_time || rule.end_time) && (
                      <Tag color='purple' size='small' shape='circle'>
                        {rule.start_time || '00:00'} -{' '}
                        {rule.end_time || '23:59'}
                      </Tag>
                    )}
                  </div>
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
            ))
          ) : (
            <div
              className='text-center py-4'
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
