import React from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { renderQuota, renderNumber } from '../../../../helpers';

const UserInfoModal = ({
  showUserInfo,
  setShowUserInfoModal,
  userInfoData,
  t,
}) => {
  return (
    <Modal
      title={t('用户信息')}
      visible={showUserInfo}
      onCancel={() => setShowUserInfoModal(false)}
      footer={null}
      centered={true}
    >
      {userInfoData && (
        <div style={{ padding: 12 }}>
          <p>
            {t('用户名')}: {userInfoData.username}
          </p>
          <p>
            {t('余额')}: {renderQuota(userInfoData.quota)}
          </p>
          <p>
            {t('已用额度')}：{renderQuota(userInfoData.used_quota)}
          </p>
          <p>
            {t('请求次数')}：{renderNumber(userInfoData.request_count)}
          </p>
        </div>
      )}
    </Modal>
  );
};

export default UserInfoModal; 