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
import CardPro from '../../common/ui/CardPro';
import StoredMediaTable from './StoredMediaTable';
import StoredMediaActions from './StoredMediaActions';
import StoredMediaFilters from './StoredMediaFilters';
import ViewStoredMediaModal from './modals/ViewStoredMediaModal';
import { useStoredMediaData } from '../../../hooks/stored-media/useStoredMediaData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import { createCardProPagination } from '../../../helpers/utils';

const StoredMediaPage = () => {
  const data = useStoredMediaData();
  const isMobile = useIsMobile();

  return (
    <>
      <ViewStoredMediaModal
        visible={data.viewModalVisible}
        onCancel={data.closeViewModal}
        data={data.viewModalData}
        t={data.t}
      />

      <CardPro
        type='type2'
        statsArea={<StoredMediaActions {...data} />}
        searchArea={<StoredMediaFilters {...data} />}
        paginationArea={createCardProPagination({
          currentPage: data.activePage,
          pageSize: data.pageSize,
          total: data.total,
          onPageChange: data.handlePageChange,
          onPageSizeChange: data.handlePageSizeChange,
          isMobile,
          t: data.t,
        })}
        t={data.t}
      >
        <StoredMediaTable {...data} />
      </CardPro>
    </>
  );
};

export default StoredMediaPage;

