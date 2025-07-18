import React from 'react';
import { Button, Tag, Space, Spin } from '@douyinfe/semi-ui';
import { renderQuota } from '../../../helpers';

const LogsActions = ({
  stat,
  loadingStat,
  compactMode,
  setCompactMode,
  t,
}) => {
  return (
    <Spin spinning={loadingStat}>
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full">
        <Space>
          <Tag
            color='blue'
            style={{
              fontWeight: 500,
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
              padding: 13,
            }}
            className='!rounded-lg'
          >
            {t('消耗额度')}: {renderQuota(stat.quota)}
          </Tag>
          <Tag
            color='pink'
            style={{
              fontWeight: 500,
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
              padding: 13,
            }}
            className='!rounded-lg'
          >
            RPM: {stat.rpm}
          </Tag>
          <Tag
            color='white'
            style={{
              border: 'none',
              boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
              fontWeight: 500,
              padding: 13,
            }}
            className='!rounded-lg'
          >
            TPM: {stat.tpm}
          </Tag>
        </Space>

        <Button
          type='tertiary'
          className="w-full md:w-auto"
          onClick={() => setCompactMode(!compactMode)}
          size="small"
        >
          {compactMode ? t('自适应列表') : t('紧凑列表')}
        </Button>
      </div>
    </Spin>
  );
};

export default LogsActions; 