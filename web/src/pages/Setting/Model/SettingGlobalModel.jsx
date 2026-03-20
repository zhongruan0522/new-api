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

import React, { useEffect, useState, useRef } from 'react';
import {
  Button,
  Col,
  Form,
  Row,
  Spin,
  Banner,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const defaultGlobalSettingInputs = {
  'global.pass_through_request_enabled': false,
  'global.third_party_multimodal_model_id': '',
  'global.third_party_multimodal_call_api_type': 0,
  'global.third_party_multimodal_system_prompt': '',
  'global.third_party_multimodal_first_user_prompt': '',
  'global.third_party_multimodal_user_agent': '',
  'global.third_party_multimodal_x_title': '',
  'global.third_party_multimodal_http_referer': '',
  'general_setting.ping_interval_enabled': false,
  'general_setting.ping_interval_seconds': 60,
};

const thirdPartyMultimodalOptionKeys = [
  'global.third_party_multimodal_model_id',
  'global.third_party_multimodal_call_api_type',
  'global.third_party_multimodal_system_prompt',
  'global.third_party_multimodal_first_user_prompt',
  'global.third_party_multimodal_user_agent',
  'global.third_party_multimodal_x_title',
  'global.third_party_multimodal_http_referer',
];

export default function SettingGlobalModel(props) {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const [thirdPartyMultimodalSaving, setThirdPartyMultimodalSaving] =
    useState(false);
  const [inputs, setInputs] = useState(defaultGlobalSettingInputs);
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(defaultGlobalSettingInputs);
  const [enabledModels, setEnabledModels] = useState([]);

  const getAllEnabledModels = async () => {
    try {
      const res = await API.get('/api/channel/models_enabled');
      const { success, message, data } = res.data;
      if (success) {
        setEnabledModels(Array.isArray(data) ? data : []);
      } else {
        showError(message);
      }
    } catch (error) {
      console.error(t('获取启用模型失败:'), error);
      showError(t('获取启用模型失败'));
    }
  };

  const normalizeValueBeforeSave = (key, value) => {
    if (key === 'global.thinking_model_blacklist') {
      const text = typeof value === 'string' ? value.trim() : '';
      return text === '' ? '[]' : value;
    }
    return value;
  };

  const saveThirdPartyMultimodalSettings = async () => {
    const changed = compareObjects(inputs, inputsRow).filter((item) =>
      thirdPartyMultimodalOptionKeys.includes(item.key),
    );
    if (!changed.length) return showWarning(t('你似乎并没有修改什么'));

    setThirdPartyMultimodalSaving(true);
    try {
      const requestQueue = changed.map((item) =>
        API.put('/api/option/', {
          key: item.key,
          value: String(inputs[item.key] ?? ''),
        }),
      );
      const res = await Promise.all(requestQueue);
      if (changed.length === 1) {
        if (res.includes(undefined)) return;
      } else if (changed.length > 1) {
        if (res.includes(undefined))
          return showError(t('部分保存失败，请重试'));
      }

      const updated = { ...inputsRow };
      changed.forEach((item) => {
        updated[item.key] = inputs[item.key];
      });
      setInputsRow(updated);
      showSuccess(t('保存成功'));
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setThirdPartyMultimodalSaving(false);
    }
  };

  const resetThirdPartyMultimodalSettings = () => {
    const next = { ...inputs };
    thirdPartyMultimodalOptionKeys.forEach((key) => {
      next[key] = inputsRow[key] ?? defaultGlobalSettingInputs[key];
    });

    setInputs(next);
    if (refForm.current) {
      thirdPartyMultimodalOptionKeys.forEach((key) =>
        refForm.current.setValue(key, next[key]),
      );
    }
  };

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      const normalizedValue = normalizeValueBeforeSave(
        item.key,
        inputs[item.key],
      );
      let value = String(normalizedValue);

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
    getAllEnabledModels();
  }, []);

  useEffect(() => {
    const currentInputs = {};
    for (const key of Object.keys(defaultGlobalSettingInputs)) {
      if (props.options[key] !== undefined) {
        let value = props.options[key];
        if (key === 'global.thinking_model_blacklist') {
          try {
            value =
              value && String(value).trim() !== ''
                ? JSON.stringify(JSON.parse(value), null, 2)
                : defaultGlobalSettingInputs[key];
          } catch (error) {
            value = defaultGlobalSettingInputs[key];
          }
        }
        if (key === 'global.third_party_multimodal_call_api_type') {
          const parsed = parseInt(value, 10);
          value = Number.isFinite(parsed)
            ? parsed
            : defaultGlobalSettingInputs[key];
        }
        currentInputs[key] = value;
      } else {
        currentInputs[key] = defaultGlobalSettingInputs[key];
      }
    }

    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    if (refForm.current) {
      refForm.current.setValues(currentInputs);
    }
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('全局设置')}>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  label={t('启用请求透传')}
                  field={'global.pass_through_request_enabled'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'global.pass_through_request_enabled': value,
                    })
                  }
                  extraText={t(
                    '开启后，所有请求将直接透传给上游，不会进行任何处理（重定向和渠道适配也将失效）,请谨慎开启',
                  )}
                />
              </Col>
            </Row>

            <Form.Section
              text={
                <span style={{ fontSize: 14, fontWeight: 600 }}>
                  {t('第三方模型方式（多模态转文本）')}
                </span>
              }
            >
              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <Banner
                    type='info'
                    description={t(
                      '说明：当渠道选择“第三方模型方式”时，系统会先调用此处配置的多模态模型，将图片/视频转成文本，再以“图片1：.../视频1：...”形式回填到原始请求的 user 消息中。',
                    )}
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }}>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Select
                    label={t('多模态模型ID')}
                    field={'global.third_party_multimodal_model_id'}
                    placeholder={t('请选择一个已启用的模型ID')}
                    showClear
                    search
                    optionList={enabledModels.map((m) => ({
                      label: m,
                      value: m,
                    }))}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_model_id': value,
                      })
                    }
                    extraText={t('从站内已启用的模型ID中选择')}
                  />
                </Col>

                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Select
                    label={t('调用方式')}
                    field={'global.third_party_multimodal_call_api_type'}
                    placeholder={t('请选择调用方式')}
                    optionList={[
                      { label: 'OpenAI-Chat', value: 0 },
                      { label: 'Claude', value: 1 },
                      { label: 'Gemini', value: 9 },
                    ]}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_call_api_type': value,
                      })
                    }
                    extraText={t(
                      '用于选择第三方多模态模型的请求规范（OpenAI/Claude/Gemini）',
                    )}
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <Form.TextArea
                    label={t('系统提示词')}
                    field={'global.third_party_multimodal_system_prompt'}
                    rows={3}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_system_prompt': value,
                      })
                    }
                    extraText={t('发送给第三方多模态模型的 system 提示词')}
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <Form.TextArea
                    label={t('第一条User提示词')}
                    field={'global.third_party_multimodal_first_user_prompt'}
                    rows={3}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_first_user_prompt': value,
                      })
                    }
                    extraText={t(
                      '发送给第三方多模态模型的第一条 user 提示词（建议明确要求输出格式/语言/长度）',
                    )}
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <Banner
                    type='warning'
                    description={t(
                      '说明：这里配置的 User-Agent / X-Title / HTTP-Referer 仅用于“第三方模型方式（多模态转文本）”内部调用上游多模态模型时的请求头；不会写入普通渠道的上游转发请求。',
                    )}
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }} gutter={16}>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    label='User-Agent'
                    field={'global.third_party_multimodal_user_agent'}
                    placeholder={t('可选，不填则不设置')}
                    showClear
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_user_agent': value,
                      })
                    }
                  />
                </Col>

                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    label='HTTP-TITLE / X-Title'
                    field={'global.third_party_multimodal_x_title'}
                    placeholder={t('可选，不填则不设置')}
                    showClear
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_x_title': value,
                      })
                    }
                  />
                </Col>

                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    label='HTTP-Referer / Referer'
                    field={'global.third_party_multimodal_http_referer'}
                    placeholder={t('可选，不填则不设置')}
                    showClear
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'global.third_party_multimodal_http_referer': value,
                      })
                    }
                  />
                </Col>
              </Row>

              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <div className='flex gap-2'>
                    <Button
                      type='primary'
                      size='small'
                      onClick={saveThirdPartyMultimodalSettings}
                      loading={thirdPartyMultimodalSaving}
                    >
                      {t('保存')}
                    </Button>
                    <Button
                      type='secondary'
                      size='small'
                      onClick={resetThirdPartyMultimodalSettings}
                      disabled={thirdPartyMultimodalSaving}
                    >
                      {t('重置')}
                    </Button>
                  </div>
                </Col>
              </Row>
            </Form.Section>

            <Form.Section
              text={
                <span style={{ fontSize: 14, fontWeight: 600 }}>
                  {t('连接保活设置')}
                </span>
              }
            >
              <Row style={{ marginTop: 10 }}>
                <Col span={24}>
                  <Banner
                    type='warning'
                    description={t(
                      '警告：启用保活后，如果已经写入保活数据后渠道出错，系统无法重试，如果必须开启，推荐设置尽可能大的Ping间隔',
                    )}
                  />
                </Col>
              </Row>
              <Row>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    label={t('启用Ping间隔')}
                    field={'general_setting.ping_interval_enabled'}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'general_setting.ping_interval_enabled': value,
                      })
                    }
                    extraText={t('开启后，将定期发送ping数据保持连接活跃')}
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.InputNumber
                    label={t('Ping间隔（秒）')}
                    field={'general_setting.ping_interval_seconds'}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        'general_setting.ping_interval_seconds': value,
                      })
                    }
                    min={1}
                    disabled={!inputs['general_setting.ping_interval_enabled']}
                  />
                </Col>
              </Row>
            </Form.Section>

            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
