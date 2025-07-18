import React from 'react';
import { Layout } from '@douyinfe/semi-ui';
import CardPro from '../../common/ui/CardPro.js';
import TaskLogsTable from './TaskLogsTable.jsx';
import TaskLogsActions from './TaskLogsActions.jsx';
import TaskLogsFilters from './TaskLogsFilters.jsx';
import ColumnSelectorModal from './modals/ColumnSelectorModal.jsx';
import ContentModal from './modals/ContentModal.jsx';
import { useTaskLogsData } from '../../../hooks/task-logs/useTaskLogsData.js';

const TaskLogsPage = () => {
  const taskLogsData = useTaskLogsData();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...taskLogsData} />
      <ContentModal {...taskLogsData} />

      <Layout>
        <CardPro
          type="type2"
          statsArea={<TaskLogsActions {...taskLogsData} />}
          searchArea={<TaskLogsFilters {...taskLogsData} />}
        >
          <TaskLogsTable {...taskLogsData} />
        </CardPro>
      </Layout>
    </>
  );
};

export default TaskLogsPage; 