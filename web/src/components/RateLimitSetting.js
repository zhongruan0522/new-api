import React, { useEffect, useState } from 'react';
import { Card, Spin, Tabs } from '@douyinfe/semi-ui';

import { API, showError, showSuccess } from '../helpers';
import SettingsChats from '../pages/Setting/Operation/SettingsChats.js';
import { useTranslation } from 'react-i18next';
import RequestRateLimit from '../pages/Setting/RateLimit/SettingsRequestRateLimit.js';

const RateLimitSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
  	ModelRequestRateLimitEnabled: false,
  	ModelRequestRateLimitCount: 0,
  	ModelRequestRateLimitSuccessCount: 1000,
  	ModelRequestRateLimitDurationMinutes: 1,
  	ModelRequestRateLimitGroup: '{}',
  });
 
  let [loading, setLoading] = useState(false);
 
  const getOptions = async () => {
  	const res = await API.get('/api/option/');
  	const { success, message, data } = res.data;
  	if (success) {
  		let newInputs = {};
  		data.forEach((item) => {
  			// 检查 key 是否在初始 inputs 中定义
  			if (Object.prototype.hasOwnProperty.call(inputs, item.key)) {
  				if (item.key.endsWith('Enabled')) {
  					newInputs[item.key] = item.value === 'true';
  				} else {
  					newInputs[item.key] = item.value;
  				}
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
  			<Card style={{ marginTop: '10px' }}>
  				<RequestRateLimit options={inputs} refresh={onRefresh} />
  			</Card>
  		</Spin>
  	</>
  );
 };
 
 export default RateLimitSetting;
