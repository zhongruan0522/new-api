import React, { useMemo } from 'react';
import { Table, Empty } from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { getMjLogsColumns } from './MjLogsColumnDefs.js';

const MjLogsTable = (mjLogsData) => {
  const {
    logs,
    loading,
    activePage,
    pageSize,
    logCount,
    compactMode,
    visibleColumns,
    handlePageChange,
    handlePageSizeChange,
    copyText,
    openContentModal,
    openImageModal,
    isAdminUser,
    t,
    COLUMN_KEYS,
  } = mjLogsData;

  // Get all columns
  const allColumns = useMemo(() => {
    return getMjLogsColumns({
      t,
      COLUMN_KEYS,
      copyText,
      openContentModal,
      openImageModal,
      isAdminUser,
    });
  }, [
    t,
    COLUMN_KEYS,
    copyText,
    openContentModal,
    openImageModal,
    isAdminUser,
  ]);

  // Filter columns based on visibility settings
  const getVisibleColumns = () => {
    return allColumns.filter((column) => visibleColumns[column.key]);
  };

  const visibleColumnsList = useMemo(() => {
    return getVisibleColumns();
  }, [visibleColumns, allColumns]);

  const tableColumns = useMemo(() => {
    return compactMode
      ? visibleColumnsList.map(({ fixed, ...rest }) => rest)
      : visibleColumnsList;
  }, [compactMode, visibleColumnsList]);

  return (
    <Table
      columns={tableColumns}
      dataSource={logs}
      rowKey='key'
      loading={loading}
      scroll={compactMode ? undefined : { x: 'max-content' }}
      className="rounded-xl overflow-hidden"
      size="middle"
      empty={
        <Empty
          image={
            <IllustrationNoResult style={{ width: 150, height: 150 }} />
          }
          darkModeImage={
            <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
          }
          description={t('搜索无结果')}
          style={{ padding: 30 }}
        />
      }
      pagination={{
        currentPage: activePage,
        pageSize: pageSize,
        total: logCount,
        pageSizeOptions: [10, 20, 50, 100],
        showSizeChanger: true,
        onPageSizeChange: handlePageSizeChange,
        onPageChange: handlePageChange,
      }}
    />
  );
};

export default MjLogsTable; 