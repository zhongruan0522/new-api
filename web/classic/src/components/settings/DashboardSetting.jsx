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

import React, { useEffect, useState, useMemo, useRef } from 'react';
import {
  Banner,
  Button,
  Card,
  Col,
  Form,
  Modal,
  Row,
  Spin,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  toBoolean,
} from '../../helpers';
import SettingsAPIInfo from '../../pages/Setting/Dashboard/SettingsAPIInfo';
import SettingsAnnouncements from '../../pages/Setting/Dashboard/SettingsAnnouncements';
import SettingsDataDashboard from '../../pages/Setting/Dashboard/SettingsDataDashboard';
import SettingsFAQ from '../../pages/Setting/Dashboard/SettingsFAQ';
import SettingsHeaderNavModules from '../../pages/Setting/Operation/SettingsHeaderNavModules';
import SettingsSidebarModulesAdmin from '../../pages/Setting/Operation/SettingsSidebarModulesAdmin';
import SettingsUptimeKuma from '../../pages/Setting/Dashboard/SettingsUptimeKuma';

const LEGAL_USER_AGREEMENT_KEY = 'legal.user_agreement';
const LEGAL_PRIVACY_POLICY_KEY = 'legal.privacy_policy';

const DashboardSetting = () => {
  const { t } = useTranslation();

  // 仪表盘相关设置
  let [dashboardInputs, setDashboardInputs] = useState({
    'console_setting.api_info': '',
    'console_setting.announcements': '',
    'console_setting.faq': '',
    'console_setting.uptime_kuma_groups': '',
    'console_setting.api_info_enabled': '',
    'console_setting.announcements_enabled': '',
    'console_setting.faq_enabled': '',
    'console_setting.uptime_kuma_enabled': '',

    // 用于迁移检测的旧键，下个版本会删除
    ApiInfo: '',
    Announcements: '',
    FAQ: '',
    UptimeKumaUrl: '',
    UptimeKumaSlug: '',

    /* 数据看板 */
    DataExportEnabled: false,
    DataExportDefaultTime: 'hour',
    DataExportInterval: 5,

    /* 顶栏模块管理 */
    HeaderNavModules: '',

    /* 左侧边栏模块管理（管理员） */
    SidebarModulesAdmin: '',
  });

  // 通用设置 & 个性化设置
  let [otherInputs, setOtherInputs] = useState({
    Notice: '',
    [LEGAL_USER_AGREEMENT_KEY]: '',
    [LEGAL_PRIVACY_POLICY_KEY]: '',
    SystemName: '',
    Logo: '',
    Footer: '',
    About: '',
    HomePageContent: '',
  });

  let [loading, setLoading] = useState(false);
  const [showMigrateModal, setShowMigrateModal] = useState(false);

  // 通用设置表单引用
  const formAPISettingGeneral = useRef();
  // 个性化设置表单引用
  const formAPIPersonalization = useRef();

  const [loadingInput, setLoadingInput] = useState({
    Notice: false,
    [LEGAL_USER_AGREEMENT_KEY]: false,
    [LEGAL_PRIVACY_POLICY_KEY]: false,
    SystemName: false,
    Logo: false,
    HomePageContent: false,
    About: false,
    Footer: false,
  });

  // 更新单个配置项
  const updateOption = async (key, value) => {
    const res = await API.put('/api/option/', { key, value });
    const { success, message } = res.data;
    if (success) {
      setOtherInputs((prev) => ({ ...prev, [key]: value }));
    } else {
      showError(message);
    }
  };

  const handleInputChange = async (value, e) => {
    const name = e.target.id;
    setOtherInputs((prev) => ({ ...prev, [name]: value }));
  };

  // 通用设置 - 公告
  const submitNotice = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, Notice: true }));
      await updateOption('Notice', otherInputs.Notice);
      showSuccess(t('公告已更新'));
    } catch (error) {
      console.error(t('公告更新失败'), error);
      showError(t('公告更新失败'));
    } finally {
      setLoadingInput((prev) => ({ ...prev, Notice: false }));
    }
  };

  // 通用设置 - 用户协议
  const submitUserAgreement = async () => {
    try {
      setLoadingInput((prev) => ({
        ...prev,
        [LEGAL_USER_AGREEMENT_KEY]: true,
      }));
      await updateOption(
        LEGAL_USER_AGREEMENT_KEY,
        otherInputs[LEGAL_USER_AGREEMENT_KEY],
      );
      showSuccess(t('用户协议已更新'));
    } catch (error) {
      console.error(t('用户协议更新失败'), error);
      showError(t('用户协议更新失败'));
    } finally {
      setLoadingInput((prev) => ({
        ...prev,
        [LEGAL_USER_AGREEMENT_KEY]: false,
      }));
    }
  };

  // 通用设置 - 隐私政策
  const submitPrivacyPolicy = async () => {
    try {
      setLoadingInput((prev) => ({
        ...prev,
        [LEGAL_PRIVACY_POLICY_KEY]: true,
      }));
      await updateOption(
        LEGAL_PRIVACY_POLICY_KEY,
        otherInputs[LEGAL_PRIVACY_POLICY_KEY],
      );
      showSuccess(t('隐私政策已更新'));
    } catch (error) {
      console.error(t('隐私政策更新失败'), error);
      showError(t('隐私政策更新失败'));
    } finally {
      setLoadingInput((prev) => ({
        ...prev,
        [LEGAL_PRIVACY_POLICY_KEY]: false,
      }));
    }
  };

  // 个性化设置 - 系统名称
  const submitSystemName = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, SystemName: true }));
      await updateOption('SystemName', otherInputs.SystemName);
      showSuccess(t('系统名称已更新'));
    } catch (error) {
      console.error(t('系统名称更新失败'), error);
      showError(t('系统名称更新失败'));
    } finally {
      setLoadingInput((prev) => ({ ...prev, SystemName: false }));
    }
  };

  // 个性化设置 - Logo
  const submitLogo = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, Logo: true }));
      await updateOption('Logo', otherInputs.Logo);
      showSuccess('Logo 已更新');
    } catch (error) {
      console.error('Logo 更新失败', error);
      showError('Logo 更新失败');
    } finally {
      setLoadingInput((prev) => ({ ...prev, Logo: false }));
    }
  };

  // 个性化设置 - 首页内容
  const submitHomePageContent = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, HomePageContent: true }));
      await updateOption('HomePageContent', otherInputs.HomePageContent);
      showSuccess('首页内容已更新');
    } catch (error) {
      console.error('首页内容更新失败', error);
      showError('首页内容更新失败');
    } finally {
      setLoadingInput((prev) => ({ ...prev, HomePageContent: false }));
    }
  };

  // 个性化设置 - 关于
  const submitAbout = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, About: true }));
      await updateOption('About', otherInputs.About);
      showSuccess('关于内容已更新');
    } catch (error) {
      console.error('关于内容更新失败', error);
      showError('关于内容更新失败');
    } finally {
      setLoadingInput((prev) => ({ ...prev, About: false }));
    }
  };

  // 个性化设置 - 页脚
  const submitFooter = async () => {
    try {
      setLoadingInput((prev) => ({ ...prev, Footer: true }));
      await updateOption('Footer', otherInputs.Footer);
      showSuccess('页脚内容已更新');
    } catch (error) {
      console.error('页脚内容更新失败', error);
      showError('页脚内容更新失败');
    } finally {
      setLoadingInput((prev) => ({ ...prev, Footer: false }));
    }
  };

  // 获取仪表盘相关配置
  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newDashboardInputs = {};
      let newOtherInputs = {};
      data.forEach((item) => {
        if (item.key in dashboardInputs) {
          newDashboardInputs[item.key] = item.value;
        }
        if (item.key.endsWith('Enabled') && item.key === 'DataExportEnabled') {
          newDashboardInputs[item.key] = toBoolean(item.value);
        }
        if (item.key in otherInputs) {
          newOtherInputs[item.key] = item.value;
        }
      });
      setDashboardInputs(newDashboardInputs);
      setOtherInputs(newOtherInputs);

      // 同步表单值
      if (formAPISettingGeneral.current) {
        formAPISettingGeneral.current.setValues(newOtherInputs);
      }
      if (formAPIPersonalization.current) {
        formAPIPersonalization.current.setValues(newOtherInputs);
      }
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

  // 用于迁移检测的旧键，下个版本会删除
  const hasLegacyData = useMemo(() => {
    const legacyKeys = [
      'ApiInfo',
      'Announcements',
      'FAQ',
      'UptimeKumaUrl',
      'UptimeKumaSlug',
    ];
    return legacyKeys.some((k) => dashboardInputs[k]);
  }, [dashboardInputs]);

  useEffect(() => {
    if (hasLegacyData) {
      setShowMigrateModal(true);
    }
  }, [hasLegacyData]);

  const handleMigrate = async () => {
    try {
      setLoading(true);
      await API.post('/api/option/migrate_console_setting');
      showSuccess('旧配置迁移完成');
      await onRefresh();
      setShowMigrateModal(false);
    } catch (err) {
      console.error(err);
      showError('迁移失败: ' + (err.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Spin spinning={loading} size='large'>
        {/* 用于迁移检测的旧键模态框，下个版本会删除 */}
        <Modal
          title='配置迁移确认'
          visible={showMigrateModal}
          onOk={handleMigrate}
          onCancel={() => setShowMigrateModal(false)}
          confirmLoading={loading}
          okText='确认迁移'
          cancelText='取消'
        >
          <p>检测到旧版本的配置数据，是否要迁移到新的配置格式？</p>
          <p style={{ color: '#f57c00', marginTop: '10px' }}>
            <strong>注意：</strong>
            迁移过程中会自动处理数据格式转换，迁移完成后旧配置将被清除，请在迁移前在数据库中备份好旧配置。
          </p>
        </Modal>

        {/* 顶栏模块管理 */}
        <div style={{ marginTop: '10px' }}>
          <SettingsHeaderNavModules options={dashboardInputs} refresh={onRefresh} />
        </div>

        {/* 左侧边栏模块管理（管理员） */}
        <div style={{ marginTop: '10px' }}>
          <SettingsSidebarModulesAdmin options={dashboardInputs} refresh={onRefresh} />
        </div>

        {/* 数据看板设置（大卡片，包含子项） */}
        <Card
          style={{ marginTop: '10px' }}
          title={
            <span style={{ fontSize: 16, fontWeight: 600 }}>
              {t('数据看板设置')}
            </span>
          }
        >
          {/* 数据看板基础配置 */}
          <SettingsDataDashboard
            options={dashboardInputs}
            refresh={onRefresh}
          />

          {/* 系统公告管理 */}
          <div style={{ marginTop: 16 }}>
            <SettingsAnnouncements
              options={dashboardInputs}
              refresh={onRefresh}
            />
          </div>

          {/* API信息管理 */}
          <div style={{ marginTop: 16 }}>
            <SettingsAPIInfo options={dashboardInputs} refresh={onRefresh} />
          </div>

          {/* 常见问答管理 */}
          <div style={{ marginTop: 16 }}>
            <SettingsFAQ options={dashboardInputs} refresh={onRefresh} />
          </div>

          {/* Uptime Kuma 监控分类管理 */}
          <div style={{ marginTop: 16 }}>
            <SettingsUptimeKuma options={dashboardInputs} refresh={onRefresh} />
          </div>
        </Card>

        {/* 通用设置 */}
        <Form
          values={otherInputs}
          getFormApi={(formAPI) => (formAPISettingGeneral.current = formAPI)}
        >
          <Card style={{ marginTop: '10px' }}>
            <Form.Section text={t('通用设置')}>
              <Form.TextArea
                label={t('公告')}
                placeholder={t(
                  '在此输入新的公告内容，支持 Markdown & HTML 代码',
                )}
                field={'Notice'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitNotice} loading={loadingInput['Notice']}>
                {t('设置公告')}
              </Button>
              <Form.TextArea
                label={t('用户协议')}
                placeholder={t(
                  '在此输入用户协议内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_USER_AGREEMENT_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写用户协议内容后，用户注册时将被要求勾选已阅读用户协议',
                )}
              />
              <Button
                onClick={submitUserAgreement}
                loading={loadingInput[LEGAL_USER_AGREEMENT_KEY]}
              >
                {t('设置用户协议')}
              </Button>
              <Form.TextArea
                label={t('隐私政策')}
                placeholder={t(
                  '在此输入隐私政策内容，支持 Markdown & HTML 代码',
                )}
                field={LEGAL_PRIVACY_POLICY_KEY}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
                helpText={t(
                  '填写隐私政策内容后，用户注册时将被要求勾选已阅读隐私政策',
                )}
              />
              <Button
                onClick={submitPrivacyPolicy}
                loading={loadingInput[LEGAL_PRIVACY_POLICY_KEY]}
              >
                {t('设置隐私政策')}
              </Button>
            </Form.Section>
          </Card>
        </Form>

        {/* 个性化设置 */}
        <Form
          values={otherInputs}
          getFormApi={(formAPI) => (formAPIPersonalization.current = formAPI)}
        >
          <Card style={{ marginTop: '10px' }}>
            <Form.Section text={t('个性化设置')}>
              <Form.Input
                label={t('系统名称')}
                placeholder={t('在此输入系统名称')}
                field={'SystemName'}
                onChange={handleInputChange}
              />
              <Button
                onClick={submitSystemName}
                loading={loadingInput['SystemName']}
              >
                {t('设置系统名称')}
              </Button>
              <Form.Input
                label={t('Logo 图片地址')}
                placeholder={t('在此输入 Logo 图片地址')}
                field={'Logo'}
                onChange={handleInputChange}
              />
              <Button onClick={submitLogo} loading={loadingInput['Logo']}>
                {t('设置 Logo')}
              </Button>
              <Form.TextArea
                label={t('首页内容')}
                placeholder={t(
                  '在此输入首页内容，支持 Markdown & HTML 代码，设置后首页的状态信息将不再显示。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为首页',
                )}
                field={'HomePageContent'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button
                onClick={submitHomePageContent}
                loading={loadingInput['HomePageContent']}
              >
                {t('设置首页内容')}
              </Button>
              <Form.TextArea
                label={t('关于')}
                placeholder={t(
                  '在此输入新的关于内容，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为关于页面',
                )}
                field={'About'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitAbout} loading={loadingInput['About']}>
                {t('设置关于')}
              </Button>
              <Banner
                fullMode={false}
                type='info'
                description={t(
                  '移除 One API 的版权标识必须首先获得授权，项目维护需要花费大量精力，如果本项目对你有意义，请主动支持本项目',
                )}
                closeIcon={null}
                style={{ marginTop: 15 }}
              />
              <Form.Input
                label={t('页脚')}
                placeholder={t(
                  '在此输入新的页脚，留空则使用默认页脚，支持 HTML 代码',
                )}
                field={'Footer'}
                onChange={handleInputChange}
              />
              <Button onClick={submitFooter} loading={loadingInput['Footer']}>
                {t('设置页脚')}
              </Button>
            </Form.Section>
          </Card>
        </Form>
      </Spin>
    </>
  );
};

export default DashboardSetting;
