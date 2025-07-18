import React from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { IconUserAdd } from '@douyinfe/semi-icons';

const { Text } = Typography;

const UsersDescription = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full">
      <div className="flex items-center text-blue-500">
        <IconUserAdd className="mr-2" />
        <Text>{t('用户管理页面，可以查看和管理所有注册用户的信息、权限和状态。')}</Text>
      </div>
      <Button
        type='tertiary'
        className="w-full md:w-auto"
        onClick={() => setCompactMode(!compactMode)}
        size="small"
      >
        {compactMode ? t('自适应列表') : t('紧凑列表')}
      </Button>
    </div>
  );
};

export default UsersDescription; 