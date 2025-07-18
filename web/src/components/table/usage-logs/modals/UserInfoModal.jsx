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