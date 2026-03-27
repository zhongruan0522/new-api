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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const DB_TYPE_LABEL = {
  mysql: 'MySQL',
  postgres: 'PostgreSQL',
  sqlite: 'SQLite',
};

// 同类型迁移支持的方向
function getSameTypeDirection(mainType) {
  if (mainType === 'mysql') return 'mysql_to_mysql';
  if (mainType === 'postgres') return 'postgres_to_postgres';
  return '';
}

function getSameTypeDirectionLabel(mainType) {
  if (mainType === 'mysql') return 'MySQL -> MySQL';
  if (mainType === 'postgres') return 'PostgreSQL -> PostgreSQL';
  return '';
}

function isSameTypeSupported(mainType) {
  return mainType === 'mysql' || mainType === 'postgres';
}

function validateSameTypeDsn(mainType, dsn) {
  const trimmed = (dsn || '').trim();
  if (!trimmed) return '目标数据库 DSN 不能为空';
  if (mainType === 'postgres') {
    const ok =
      trimmed.startsWith('postgres://') || trimmed.startsWith('postgresql://');
    if (!ok) {
      return '目标为 PostgreSQL 时，DSN 必须以 postgres:// 或 postgresql:// 开头';
    }
  }
  return '';
}

function formatJobStatusText(job) {
  if (!job) return '-';
  if (job.status === 'running') return '运行中';
  if (job.status === 'success') return '成功';
  if (job.status === 'failed') return '失败';
  return job.status || '-';
}

const DatabaseSameTypeMigration = () => {
  const { t } = useTranslation();

  const [infoLoading, setInfoLoading] = useState(false);
  const [info, setInfo] = useState(null);

  const [targetDsn, setTargetDsn] = useState('');
  const [includeLogs, setIncludeLogs] = useState(false);
  const [targetLogDsn, setTargetLogDsn] = useState('');
  const [force, setForce] = useState(false);

  const [confirmVisible, setConfirmVisible] = useState(false);
  const [progressVisible, setProgressVisible] = useState(false);
  const [jobLoading, setJobLoading] = useState(false);
  const [job, setJob] = useState(null);
  const [jobId, setJobId] = useState('');

  const pollTimerRef = useRef(null);

  const stopPolling = () => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  };

  const fetchInfo = async () => {
    setInfoLoading(true);
    try {
      const res = await API.get('/api/db/same_type_migrate/info', {
        disableDuplicate: true,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setInfo(data);
    } catch (e) {
      showError(e);
    } finally {
      setInfoLoading(false);
    }
  };

  const fetchJob = async (id) => {
    if (!id) return;
    setJobLoading(true);
    try {
      const res = await API.get(`/api/db/same_type_migrate/${id}`, {
        disableDuplicate: true,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setJob(data);
      if (data?.status && data.status !== 'running') {
        stopPolling();
        if (data.status === 'success') showSuccess(t('同类型迁移完成'));
        if (data.status === 'failed') showError(data?.error || t('同类型迁移失败'));
      }
    } catch (e) {
      showError(e);
    } finally {
      setJobLoading(false);
    }
  };

  const startPolling = (id) => {
    stopPolling();
    pollTimerRef.current = setInterval(() => fetchJob(id), 2000);
    fetchJob(id);
  };

  useEffect(() => {
    fetchInfo();
    return () => {
      stopPolling();
    };
  }, []);

  const mainType = info?.main_db_type || '';
  const supported = isSameTypeSupported(mainType);

  const handleOpenConfirm = () => {
    const msg = validateSameTypeDsn(mainType, targetDsn);
    if (msg) {
      showError(t(msg));
      return;
    }
    setConfirmVisible(true);
  };

  const handleStart = async () => {
    const msg = validateSameTypeDsn(mainType, targetDsn);
    if (msg) {
      showError(t(msg));
      return;
    }
    setConfirmVisible(false);
    setProgressVisible(true);
    setJob(null);
    setJobId('');

    try {
      const res = await API.post('/api/db/same_type_migrate', {
        target_dsn: targetDsn.trim(),
        target_log_dsn: includeLogs ? targetLogDsn.trim() : '',
        include_logs: includeLogs,
        force,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        setProgressVisible(false);
        return;
      }
      const id = data?.job_id;
      if (!id) {
        showError(t('启动失败：未返回任务ID'));
        setProgressVisible(false);
        return;
      }
      setJobId(id);
      startPolling(id);
    } catch (e) {
      showError(e);
      setProgressVisible(false);
    }
  };

  if (infoLoading) {
    return (
      <Card>
        <Form.Section text={t('同类型数据库迁移')}>
          <Spin />
        </Form.Section>
      </Card>
    );
  }

  if (!supported) {
    return (
      <Card>
        <Form.Section text={t('同类型数据库迁移')}>
          <Banner
            type='info'
            description={t(
              '同类型数据库迁移仅支持 MySQL -> MySQL 和 PostgreSQL -> PostgreSQL。当前数据库类型为 {{type}}，不支持此功能。',
              { type: DB_TYPE_LABEL[mainType] || mainType },
            )}
          />
        </Form.Section>
      </Card>
    );
  }

  return (
    <Card>
      <Form.Section text={t('同类型数据库迁移')}>
        <Banner
          type='warning'
          description={t(
            '该功能用于将当前数据库的业务数据复制到同类型的另一个数据库实例。未勾选"覆盖目标库"时，要求目标库必须为空；勾选后会清空目标库的相关表再迁移。请务必提前备份目标数据库。',
          )}
          style={{ marginBottom: 16 }}
        />

        <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Text>
              {t('当前主数据库')}：{DB_TYPE_LABEL[mainType] || '-'}
            </Text>
          </Col>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Text>
              {t('迁移方向')}：{getSameTypeDirectionLabel(mainType)}
            </Text>
          </Col>
        </Row>

        <div style={{ height: 12 }} />

        <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Text>{t('目标数据库 DSN')}</Text>
            <div style={{ marginTop: 8 }}>
              <Input
                value={targetDsn}
                onChange={setTargetDsn}
                mode='password'
                placeholder={
                  mainType === 'postgres'
                    ? 'postgres://user:pass@host:5432/dbname?sslmode=disable'
                    : t('例如：user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4')
                }
              />
            </div>
            <div style={{ marginTop: 8 }}>
              <Text type='tertiary'>
                {t('提示：MySQL DSN 需要带库名；PostgreSQL 必须以 postgres:// 或 postgresql:// 开头。')}
              </Text>
            </div>
          </Col>

          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Text type='tertiary'>
              {t('当前日志数据库')}：{DB_TYPE_LABEL[info?.log_db_type] || '-'}
              {info?.log_db_is_separated ? ` (${t('独立日志库')})` : ''}
            </Text>
          </Col>
        </Row>

        <div style={{ height: 12 }} />

        <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Checkbox
              checked={includeLogs}
              onChange={(e) => setIncludeLogs(e.target.checked)}
            >
              {t('迁移日志（logs表）')}
            </Checkbox>
            {includeLogs && (
              <div style={{ marginTop: 8 }}>
                <Text>{t('目标日志库 DSN（可选）')}</Text>
                <div style={{ marginTop: 8 }}>
                  <Input
                    value={targetLogDsn}
                    onChange={setTargetLogDsn}
                    mode='password'
                    placeholder={t('留空表示使用目标主库')}
                  />
                </div>
              </div>
            )}
          </Col>

          <Col xs={24} sm={24} md={12} lg={12} xl={12}>
            <Checkbox checked={force} onChange={(e) => setForce(e.target.checked)}>
              {t('覆盖目标库（清空后迁移）')}
            </Checkbox>
            {force && (
              <div style={{ marginTop: 8 }}>
                <Text type='danger'>
                  {t('危险操作：将删除目标库对应表的现有数据。')}
                </Text>
              </div>
            )}
          </Col>
        </Row>

        <div style={{ height: 12 }} />

        <Button type='danger' onClick={handleOpenConfirm}>
          {t('开始同类型迁移')}
        </Button>

        <Modal
          title={t('确认开始同类型迁移')}
          visible={confirmVisible}
          onOk={handleStart}
          onCancel={() => setConfirmVisible(false)}
          okText={t('确认')}
          cancelText={t('取消')}
        >
          <Text>
            {t('迁移方向')}：{getSameTypeDirectionLabel(mainType)}
          </Text>
          <div style={{ height: 8 }} />
          <Text>
            {t('迁移日志')}：{includeLogs ? t('是') : t('否')}
          </Text>
          <div style={{ height: 8 }} />
          <Text type={force ? 'danger' : 'secondary'}>
            {t('覆盖目标库')}：{force ? t('是') : t('否')}
          </Text>
        </Modal>

        <Modal
          title={t('同类型迁移进度')}
          visible={progressVisible}
          onCancel={() => setProgressVisible(false)}
          footer={null}
          width={900}
        >
          <Row gutter={12}>
            <Col span={12}>
              <Text>
                {t('任务ID')}：{jobId || '-'}
              </Text>
              <div style={{ height: 8 }} />
              <Text>
                {t('状态')}：{formatJobStatusText(job)}
              </Text>
              <div style={{ height: 8 }} />
              <Text>
                {t('当前步骤')}：{job?.current_step || '-'}
              </Text>
              <div style={{ height: 8 }} />
              {jobLoading && <Spin size='small' />}
              <div style={{ height: 12 }} />

              <Text>{t('表进度')}</Text>
              <div style={{ marginTop: 8 }}>
                {(job?.tables || []).length === 0 ? (
                  <Text type='tertiary'>-</Text>
                ) : (
                  <div style={{ maxHeight: 260, overflow: 'auto' }}>
                    {(job?.tables || []).map((x) => (
                      <div
                        key={x.name}
                        style={{
                          display: 'flex',
                          justifyContent: 'space-between',
                          gap: 12,
                          padding: '4px 0',
                          borderBottom: '1px solid var(--semi-color-border)',
                        }}
                      >
                        <Text>{x.name}</Text>
                        <Text type='tertiary'>
                          {x.copied}/{x.total}
                        </Text>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </Col>

            <Col span={12}>
              <Text>{t('执行日志')}</Text>
              <div style={{ marginTop: 8 }}>
                <pre
                  style={{
                    maxHeight: 360,
                    overflow: 'auto',
                    padding: 12,
                    background: 'var(--semi-color-fill-0)',
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 6,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                  }}
                >
                  {(job?.logs || []).join('\n') || '-'}
                </pre>
              </div>
            </Col>
          </Row>
        </Modal>
      </Form.Section>
    </Card>
  );
};

export default DatabaseSameTypeMigration;
