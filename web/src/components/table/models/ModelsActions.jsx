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

import React, { useState } from 'react';
import MissingModelsModal from './modals/MissingModelsModal.jsx';
import { Button, Space, Modal } from '@douyinfe/semi-ui';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { showError } from '../../../helpers';

const ModelsActions = ({
  selectedKeys,
  setEditingModel,
  setShowEdit,
  batchDeleteModels,
  compactMode,
  setCompactMode,
  t,
}) => {
  // Modal states
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showMissingModal, setShowMissingModal] = useState(false);

  // Handle delete selected models with confirmation
  const handleDeleteSelectedModels = () => {
    if (selectedKeys.length === 0) {
      showError(t('请至少选择一个模型！'));
      return;
    }
    setShowDeleteModal(true);
  };

  // Handle delete confirmation
  const handleConfirmDelete = () => {
    batchDeleteModels();
    setShowDeleteModal(false);
  };

  return (
    <>
      <div className="flex flex-wrap gap-2 w-full md:w-auto order-2 md:order-1">
        <Button
          type="primary"
          className="flex-1 md:flex-initial"
          onClick={() => {
            setEditingModel({
              id: undefined,
            });
            setShowEdit(true);
          }}
          size="small"
        >
          {t('添加模型')}
        </Button>

        <Button
          type='danger'
          className="flex-1 md:flex-initial"
          onClick={handleDeleteSelectedModels}
          size="small"
        >
          {t('删除所选模型')}
        </Button>

        <Button
          type="secondary"
          className="flex-1 md:flex-initial"
          size="small"
          onClick={() => setShowMissingModal(true)}
        >
          {t('未配置模型')}
        </Button>

        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </div>

      <Modal
        title={t('批量删除模型')}
        visible={showDeleteModal}
        onCancel={() => setShowDeleteModal(false)}
        onOk={handleConfirmDelete}
        type="warning"
      >
        <div>
          {t('确定要删除所选的 {{count}} 个模型吗？', { count: selectedKeys.length })}
        </div>
      </Modal>

      <MissingModelsModal
        visible={showMissingModal}
        onClose={() => setShowMissingModal(false)}
        onConfigureModel={(name) => {
          setEditingModel({ id: undefined, model_name: name });
          setShowEdit(true);
          setShowMissingModal(false);
        }}
        t={t}
      />
    </>
  );
};

export default ModelsActions;