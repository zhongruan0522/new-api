import React from 'react';
import CardPro from '../../common/ui/CardPro.js';
import LogsTable from './UsageLogsTable.jsx';
import LogsActions from './UsageLogsActions.jsx';
import LogsFilters from './UsageLogsFilters.jsx';
import ColumnSelectorModal from './modals/ColumnSelectorModal.jsx';
import UserInfoModal from './modals/UserInfoModal.jsx';
import { useLogsData } from '../../../hooks/usage-logs/useUsageLogsData.js';

const LogsPage = () => {
  const logsData = useLogsData();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...logsData} />
      <UserInfoModal {...logsData} />

      {/* Main Content */}
      <CardPro
        type="type2"
        statsArea={<LogsActions {...logsData} />}
        searchArea={<LogsFilters {...logsData} />}
        t={logsData.t}
      >
        <LogsTable {...logsData} />
      </CardPro>
    </>
  );
};

export default LogsPage; 