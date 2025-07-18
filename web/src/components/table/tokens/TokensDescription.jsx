import React from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { Key } from 'lucide-react';

const { Text } = Typography;

const TokensDescription = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full">
      <div className="flex items-center text-blue-500">
        <Key size={16} className="mr-2" />
        <Text>{t('令牌用于API访问认证，可以设置额度限制和模型权限。')}</Text>
      </div>

      <Button
        type="tertiary"
        className="w-full md:w-auto"
        onClick={() => setCompactMode(!compactMode)}
        size="small"
      >
        {compactMode ? t('自适应列表') : t('紧凑列表')}
      </Button>
    </div>
  );
};

export default TokensDescription; 