import React from 'react';
import { Layout } from '@douyinfe/semi-ui';
import CardPro from '../../common/ui/CardPro.js';
import MjLogsTable from './MjLogsTable.jsx';
import MjLogsActions from './MjLogsActions.jsx';
import MjLogsFilters from './MjLogsFilters.jsx';
import ColumnSelectorModal from './modals/ColumnSelectorModal.jsx';
import ContentModal from './modals/ContentModal.jsx';
import { useMjLogsData } from '../../../hooks/mj-logs/useMjLogsData.js';

const MjLogsPage = () => {
  const mjLogsData = useMjLogsData();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...mjLogsData} />
      <ContentModal {...mjLogsData} />

      <Layout>
        <CardPro
          type="type2"
          statsArea={<MjLogsActions {...mjLogsData} />}
          searchArea={<MjLogsFilters {...mjLogsData} />}
          t={mjLogsData.t}
        >
          <MjLogsTable {...mjLogsData} />
        </CardPro>
      </Layout>
    </>
  );
};

export default MjLogsPage; 