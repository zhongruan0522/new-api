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

import React, { useContext, useEffect, useRef, useState } from 'react';
import { Avatar, Banner, Button, Card, Col, Form, Row } from '@douyinfe/semi-ui';
import { Fingerprint } from 'lucide-react';
import { API, showError, showSuccess } from '../../../../helpers';
import { UserContext } from '../../../../context/User';

const UpstreamIdentitySettings = ({ t }) => {
  const [userState, userDispatch] = useContext(UserContext);
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    upstream_user_agent: '',
    upstream_x_title: '',
    upstream_http_referer: '',
  });

  useEffect(() => {
    const raw = userState?.user?.setting;
    if (!raw) return;

    try {
      const settings = JSON.parse(raw);
      const next = {
        upstream_user_agent: settings.upstream_user_agent || '',
        upstream_x_title: settings.upstream_x_title || '',
        upstream_http_referer: settings.upstream_http_referer || '',
      };
      setInputs(next);
      if (formApiRef.current) {
        formApiRef.current.setValues(next);
      }
    } catch (error) {
      // ignore parse errors
    }
  }, [userState?.user?.setting]);

  const save = async () => {
    setLoading(true);
    try {
      const payload = {
        upstream_user_agent: inputs.upstream_user_agent || '',
        upstream_x_title: inputs.upstream_x_title || '',
        upstream_http_referer: inputs.upstream_http_referer || '',
      };
      const res = await API.put('/api/user/self', payload);
      if (!res?.data?.success) {
        showError(res?.data?.message || t('保存失败'));
        return;
      }

      showSuccess(t('保存成功'));

      if (userState?.user) {
        let settings = {};
        try {
          settings = userState.user.setting ? JSON.parse(userState.user.setting) : {};
        } catch (e) {
          settings = {};
        }

        const ua = String(payload.upstream_user_agent || '').trim();
        const xTitle = String(payload.upstream_x_title || '').trim();
        const httpReferer = String(payload.upstream_http_referer || '').trim();

        if (ua) settings.upstream_user_agent = ua;
        else delete settings.upstream_user_agent;

        if (xTitle) settings.upstream_x_title = xTitle;
        else delete settings.upstream_x_title;

        if (httpReferer) settings.upstream_http_referer = httpReferer;
        else delete settings.upstream_http_referer;

        userDispatch({
          type: 'login',
          payload: {
            ...userState.user,
            setting: JSON.stringify(settings),
          },
        });
      }
    } catch (error) {
      showError(t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='blue' className='mr-3 shadow-md'>
          <Fingerprint size={16} />
        </Avatar>
        <div>
          <div className='text-lg font-medium'>{t('上游身份标识')}</div>
          <div className='text-xs text-gray-600 dark:text-gray-400'>
            {t('用于 OpenRouter 等渠道的请求头标识')}
          </div>
        </div>
      </div>

      <Banner
        type='info'
        description={t(
          '说明：这里配置的 User-Agent / X-Title / HTTP-Referer 会作为默认请求头写入上游请求；如果渠道已配置 Header Override 或请求中已携带同名 Header，则不会覆盖。',
        )}
        className='!rounded-xl'
      />

      <Form
        initValues={inputs}
        onValueChange={(values) => setInputs(values)}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Row style={{ marginTop: 12 }} gutter={16}>
          <Col xs={24} sm={24} md={24} lg={24} xl={24}>
            <Form.Input
              field='upstream_user_agent'
              label='User-Agent'
              placeholder={t('可选，不填则不设置')}
              showClear
            />
          </Col>
        </Row>

        <Row style={{ marginTop: 12 }} gutter={16}>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Form.Input
              field='upstream_x_title'
              label='X-Title'
              placeholder={t('可选，不填则不设置')}
              showClear
            />
          </Col>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Form.Input
              field='upstream_http_referer'
              label='HTTP-Referer / Referer'
              placeholder={t('可选，不填则不设置')}
              showClear
            />
          </Col>
        </Row>

        <div style={{ marginTop: 12 }}>
          <Button type='primary' onClick={save} loading={loading}>
            {t('保存')}
          </Button>
        </div>
      </Form>
    </Card>
  );
};

export default UpstreamIdentitySettings;

