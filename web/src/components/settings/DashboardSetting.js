import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import SettingsAPIInfo from '../../pages/Setting/Dashboard/SettingsAPIInfo.js';
import SettingsAnnouncements from '../../pages/Setting/Dashboard/SettingsAnnouncements.js';
import SettingsFAQ from '../../pages/Setting/Dashboard/SettingsFAQ.js';

const DashboardSetting = () => {
  let [inputs, setInputs] = useState({
    ApiInfo: '',
    Announcements: '',
    FAQ: '',
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
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
        {/* API信息管理 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsAPIInfo options={inputs} refresh={onRefresh} />
        </Card>

        {/* 系统公告管理 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsAnnouncements options={inputs} refresh={onRefresh} />
        </Card>

        {/* 常见问答管理 */}
        <Card style={{ marginTop: '10px' }}>
          <SettingsFAQ options={inputs} refresh={onRefresh} />
        </Card>
      </Spin>
    </>
  );
};

export default DashboardSetting; 