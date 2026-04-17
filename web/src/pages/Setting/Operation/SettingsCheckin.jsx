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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import { Button, Col, Form, Row, Spin, Typography } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import {
  getCurrencyConfig,
  getQuotaPerUnit,
} from '../../../helpers/render';

function tokenToDisplay(tokenValue) {
  const quotaPerUnit = getQuotaPerUnit();
  const { type, rate } = getCurrencyConfig();
  if (type === 'TOKENS') return tokenValue;
  const usd = tokenValue / quotaPerUnit;
  return type === 'USD' ? usd : usd * (rate || 1);
}

function displayToToken(displayValue) {
  const quotaPerUnit = getQuotaPerUnit();
  const { type, rate } = getCurrencyConfig();
  if (type === 'TOKENS') return displayValue;
  const usd = type === 'USD' ? displayValue : displayValue / (rate || 1);
  return Math.round(usd * quotaPerUnit);
}

// 需要额度转换的字段
const QUOTA_FIELDS = ['checkin_setting.min_quota', 'checkin_setting.max_quota'];

export default function SettingsCheckin(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  // inputs 存储显示值（货币转换后的值），提交时再转换回内部 Token 值
  const [inputs, setInputs] = useState({
    'checkin_setting.enabled': false,
    'checkin_setting.min_quota': '',
    'checkin_setting.max_quota': '',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const currencySuffix = useMemo(() => {
    const { symbol, type } = getCurrencyConfig();
    return type === 'TOKENS' ? 'Token' : symbol;
  }, []);

  // 将 inputs 中的显示值转换为内部 Token 值用于提交
  function toRawValues(displayState) {
    const raw = {};
    for (const key in displayState) {
      const val = displayState[key];
      if (QUOTA_FIELDS.includes(key) && val !== '' && val !== undefined) {
        raw[key] = displayToToken(val);
      } else {
        raw[key] = val;
      }
    }
    return raw;
  }

  function onSubmit() {
    const rawInputs = toRawValues(inputs);
    const rawInputsRow = toRawValues(inputsRow);
    const updateArray = compareObjects(rawInputs, rawInputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      const value = String(rawInputs[item.key]);
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        let val = props.options[key];
        // 将内部 Token 值转为显示值存入 state
        if (QUOTA_FIELDS.includes(key) && val !== '' && val !== undefined) {
          val = tokenToDisplay(parseFloat(val));
        }
        currentInputs[key] = val;
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current?.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('签到设置')}>
            <Typography.Text
              type='tertiary'
              style={{ marginBottom: 16, display: 'block' }}
            >
              {t('签到功能允许用户每日签到获取随机额度奖励')}
            </Typography.Text>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'checkin_setting.enabled'}
                  label={t('启用签到功能')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs((prev) => ({
                      ...prev,
                      'checkin_setting.enabled': value,
                    }))
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='checkin_setting.min_quota'
                  label={t('签到最小额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  placeholder={t('签到奖励的最小额度')}
                  onChange={(value) =>
                    setInputs((prev) => ({
                      ...prev,
                      'checkin_setting.min_quota': value,
                    }))
                  }
                  disabled={!inputs['checkin_setting.enabled']}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field='checkin_setting.max_quota'
                  label={t('签到最大额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  placeholder={t('签到奖励的最大额度')}
                  onChange={(value) =>
                    setInputs((prev) => ({
                      ...prev,
                      'checkin_setting.max_quota': value,
                    }))
                  }
                  disabled={!inputs['checkin_setting.enabled']}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存签到设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
