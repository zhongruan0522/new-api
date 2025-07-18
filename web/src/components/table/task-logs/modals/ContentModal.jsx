import React from 'react';
import { Modal } from '@douyinfe/semi-ui';

const ContentModal = ({
  isModalOpen,
  setIsModalOpen,
  modalContent,
}) => {
  return (
    <Modal
      visible={isModalOpen}
      onOk={() => setIsModalOpen(false)}
      onCancel={() => setIsModalOpen(false)}
      closable={null}
      bodyStyle={{ height: '400px', overflow: 'auto' }}
      width={800}
    >
      <p style={{ whiteSpace: 'pre-line' }}>{modalContent}</p>
    </Modal>
  );
};

export default ContentModal; 