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
import { Button, Modal, Typography } from '@douyinfe/semi-ui';
import { showError } from '../../../helpers';

const StoredMediaActions = ({ selectedKeys, batchDelete, t }) => {
  const hasSelected = selectedKeys && selectedKeys.length > 0;

  return (
    <div className='flex flex-col gap-2'>
      <div className='flex flex-col md:flex-row justify-between gap-2'>
        <div className='flex flex-wrap items-center gap-2 w-full md:w-auto'>
          <Button
            size='small'
            type='danger'
            className='w-full md:w-auto'
            disabled={!hasSelected}
            onClick={() => {
              if (!hasSelected) {
                showError(t('请先选择要删除的文件'));
                return;
              }
              Modal.confirm({
                title: t('批量删除'),
                content: t('确定要删除所选的 {{count}} 个文件吗？', {
                  count: selectedKeys.length,
                }),
                onOk: () => batchDelete(),
                centered: true,
              });
            }}
          >
            {t('批量删除')}
          </Button>
        </div>

        <div className='w-full md:w-auto flex justify-end'>
          <Typography.Text type='tertiary' className='select-none'>
            {hasSelected
              ? t('已选择 {{count}} 项', { count: selectedKeys.length })
              : t('未选择')}
          </Typography.Text>
        </div>
      </div>
    </div>
  );
};

export default StoredMediaActions;

