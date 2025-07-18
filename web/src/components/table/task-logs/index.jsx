/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

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
          t={taskLogsData.t}
        >
          <TaskLogsTable {...taskLogsData} />
        </CardPro>
      </Layout>
    </>
  );
};

export default TaskLogsPage; 