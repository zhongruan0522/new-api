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
import { Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import {
  getCurrencyConfig,
  getQuotaPerUnit,
} from '../../../helpers/render';

// 将内部 Token 额度转换为当前货币显示值
function tokenToDisplay(tokenValue) {
  const quotaPerUnit = getQuotaPerUnit();
  const { type, rate } = getCurrencyConfig();
  if (type === 'TOKENS') return tokenValue;
  const usd = tokenValue / quotaPerUnit;
  return type === 'USD' ? usd : usd * (rate || 1);
}

// 将货币显示值转换回内部 Token 额度
function displayToToken(displayValue) {
  const quotaPerUnit = getQuotaPerUnit();
  const { type, rate } = getCurrencyConfig();
  if (type === 'TOKENS') return displayValue;
  const usd = type === 'USD' ? displayValue : displayValue / (rate || 1);
  return Math.round(usd * quotaPerUnit);
}

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    QuotaForInviter: '',
    QuotaForInvitee: '',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  const currencySuffix = useMemo(() => {
    const { symbol, type } = getCurrencyConfig();
    return type === 'TOKENS' ? 'Token' : symbol;
  }, []);

  // 将内部 Token 值转为货币显示值用于表单
  const displayValues = useMemo(() => {
    return {
      QuotaForNewUser: inputs.QuotaForNewUser !== '' ? tokenToDisplay(parseFloat(inputs.QuotaForNewUser)) : '',
      PreConsumedQuota: inputs.PreConsumedQuota !== '' ? tokenToDisplay(parseFloat(inputs.PreConsumedQuota)) : '',
      QuotaForInviter: inputs.QuotaForInviter !== '' ? tokenToDisplay(parseFloat(inputs.QuotaForInviter)) : '',
      QuotaForInvitee: inputs.QuotaForInvitee !== '' ? tokenToDisplay(parseFloat(inputs.QuotaForInvitee)) : '',
    };
  }, [inputs.QuotaForNewUser, inputs.PreConsumedQuota, inputs.QuotaForInviter, inputs.QuotaForInvitee]);

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else {
        value = inputs[item.key];
      }
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
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);
  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('额度设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('新用户初始额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  value={displayValues.QuotaForNewUser}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForNewUser: String(displayToToken(value)),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('请求预扣费额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  extraText={t('请求结束后多退少补')}
                  value={displayValues.PreConsumedQuota}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      PreConsumedQuota: String(displayToToken(value)),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('邀请新用户奖励额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  extraText={''}
                  value={displayValues.QuotaForInviter}
                  placeholder={t('例如：2000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInviter: String(displayToToken(value)),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('新用户使用邀请码奖励额度')}
                  step={0.01}
                  min={0}
                  suffix={currencySuffix}
                  extraText={''}
                  value={displayValues.QuotaForInvitee}
                  placeholder={t('例如：1000')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInvitee: String(displayToToken(value)),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col>
                <Form.Switch
                  label={t('对免费模型启用预消耗')}
                  field={'quota_setting.enable_free_model_pre_consume'}
                  extraText={t(
                    '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                  )}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'quota_setting.enable_free_model_pre_consume': value,
                    })
                  }
                />
              </Col>
            </Row>

            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存额度设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
