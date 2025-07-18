import React from 'react';
import { Skeleton, Typography } from '@douyinfe/semi-ui';
import { IconEyeOpened } from '@douyinfe/semi-icons';
import CompactModeToggle from '../../common/ui/CompactModeToggle';

const { Text } = Typography;

const MjLogsActions = ({
  loading,
  showBanner,
  isAdminUser,
  compactMode,
  setCompactMode,
  t,
}) => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full">
      <div className="flex items-center text-orange-500 mb-2 md:mb-0">
        <IconEyeOpened className="mr-2" />
        {loading ? (
          <Skeleton.Title
            style={{
              width: 300,
              marginBottom: 0,
              marginTop: 0
            }}
          />
        ) : (
          <Text>
            {isAdminUser && showBanner
              ? t('当前未开启Midjourney回调，部分项目可能无法获得绘图结果，可在运营设置中开启。')
              : t('Midjourney 任务记录')}
          </Text>
        )}
      </div>
      <CompactModeToggle
        compactMode={compactMode}
        setCompactMode={setCompactMode}
        t={t}
      />
    </div>
  );
};

export default MjLogsActions; 