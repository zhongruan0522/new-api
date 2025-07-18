import React from 'react';
import { Button, Typography } from '@douyinfe/semi-ui';
import { Ticket } from 'lucide-react';

const { Text } = Typography;

const RedemptionsDescription = ({ compactMode, setCompactMode, t }) => {
  return (
    <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full">
      <div className="flex items-center text-orange-500">
        <Ticket size={16} className="mr-2" />
        <Text>{t('兑换码可以批量生成和分发，适合用于推广活动或批量充值。')}</Text>
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

export default RedemptionsDescription; 