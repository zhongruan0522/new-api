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

import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';

import { API, showError, toBoolean } from '../../helpers';
import SettingGlobalModel from '../../pages/Setting/Model/SettingGlobalModel';
import SettingsChannelAffinity from '../../pages/Setting/Operation/SettingsChannelAffinity';

const ModelSetting = () => {
  let [inputs, setInputs] = useState({
    'global.pass_through_request_enabled': false,
    'global.thinking_model_blacklist': '[]',
    'general_setting.ping_interval_enabled': false,
    'general_setting.ping_interval_seconds': 60,
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key === 'global.thinking_model_blacklist') {
          if (item.value !== '') {
            try {
              item.value = JSON.stringify(JSON.parse(item.value), null, 2);
            } catch (e) {
              // Keep raw value so user can fix it, and avoid crashing the page.
              console.error(`Invalid JSON for option ${item.key}:`, e);
            }
          }
        }
        // Keep boolean config keys ending with enabled/Enabled so UI parses correctly.
        if (item.key.endsWith('Enabled') || item.key.endsWith('enabled')) {
          newInputs[item.key] = toBoolean(item.value);
        } else {
          newInputs[item.key] = item.value;
        }
      });

      setInputs(newInputs);
    } else {
      showError(message);
    }
  };
  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
      // showSuccess('刷新成功');
    } catch (error) {
      showError('刷新失败');
      console.error(error);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <>
      <Spin spinning={loading} size='large'>
        {/* OpenAI */}
        <Card style={{ marginTop: '10px' }}>
          <SettingGlobalModel options={inputs} refresh={onRefresh} />
        </Card>
        {/* Channel affinity */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsChannelAffinity options={inputs} refresh={onRefresh} />
        </Card>
      </Spin>
    </>
  );
};

export default ModelSetting;
