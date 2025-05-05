import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function RequestRateLimit(props) {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
  	ModelRequestRateLimitEnabled: false,
  	ModelRequestRateLimitCount: -1,
  	ModelRequestRateLimitSuccessCount: 1000,
  	ModelRequestRateLimitDurationMinutes: 1,
  	ModelRequestRateLimitGroup: '{}',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
 
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
  		if (item.key === 'ModelRequestRateLimitGroup') {
  			try {
  				JSON.parse(value);
  			} catch (e) {
  				showError(t('用户组速率限制配置不是有效的 JSON 格式！'));
  				return Promise.reject('Invalid JSON format');
  			}
  		}
  		return API.put('/api/option/', {
  			key: item.key,
  			value,
  		});
  	});
 
  	const validRequests = requestQueue.filter(req => req !== null && req !== undefined && typeof req.then === 'function');
 
  	if (validRequests.length === 0 && requestQueue.length > 0) {
  		return;
  	}
 
  	setLoading(true);
  	Promise.all(validRequests)
  		.then((res) => {
  			if (validRequests.length === 1) {
  				if (res.includes(undefined)) return;
  			} else if (validRequests.length > 1) {
  				if (res.includes(undefined))
  					return showError(t('部分保存失败，请重试'));
  			}
  			showSuccess(t('保存成功'));
  			props.refresh();
  			setInputsRow(structuredClone(inputs));
  		})
  		.catch((error) => {
  			if (error !== 'Invalid JSON format') {
  				showError(t('保存失败，请重试'));
  			}
  		})
  		.finally(() => {
  			setLoading(false);
  		});
  }
 
  useEffect(() => {
  	const currentInputs = {};
  	for (let key in props.options) {
  		if (Object.prototype.hasOwnProperty.call(inputs, key)) { // 使用 hasOwnProperty 检查
  			currentInputs[key] = props.options[key];
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
  				<Form.Section text={t('模型请求速率限制')}>
  					<Row gutter={16}>
  						<Col xs={24} sm={12} md={8} lg={8} xl={8}>
  							<Form.Switch
  								field={'ModelRequestRateLimitEnabled'}
  								label={t('启用用户模型请求速率限制（可能会影响高并发性能）')}
  								size='default'
  								checkedText='｜'
  								uncheckedText='〇'
  								onChange={(value) => {
  									setInputs({
  										...inputs,
  										ModelRequestRateLimitEnabled: value,
  									});
  								}}
  							/>
  						</Col>
  					</Row>
  					<Row>
  						<Col xs={24} sm={12} md={8} lg={8} xl={8}>
  							<Form.InputNumber
  								label={t('限制周期')}
  								step={1}
  								min={0}
  								suffix={t('分钟')}
  								extraText={t('频率限制的周期（分钟）')}
  								field={'ModelRequestRateLimitDurationMinutes'}
  								onChange={(value) =>
  									setInputs({
  										...inputs,
  										ModelRequestRateLimitDurationMinutes: String(value),
  									})
  								}
  							/>
  						</Col>
  					</Row>
  					<Row>
  						<Col xs={24} sm={12} md={8} lg={8} xl={8}>
  							<Form.InputNumber
  								label={t('用户每周期最多请求次数')}
  								step={1}
  								min={0}
  								suffix={t('次')}
  								extraText={t('包括失败请求的次数，0代表不限制')}
  								field={'ModelRequestRateLimitCount'}
  								onChange={(value) =>
  									setInputs({
  										...inputs,
  										ModelRequestRateLimitCount: String(value),
  									})
  								}
  							/>
  						</Col>
  						<Col xs={24} sm={12} md={8} lg={8} xl={8}>
  							<Form.InputNumber
  								label={t('用户每周期最多请求完成次数')}
  								step={1}
  								min={1}
  								suffix={t('次')}
  								extraText={t('只包括请求成功的次数')}
  								field={'ModelRequestRateLimitSuccessCount'}
  								onChange={(value) =>
  									setInputs({
  										...inputs,
  										ModelRequestRateLimitSuccessCount: String(value),
  									})
  								}
  							/>
  						</Col>
  					</Row>
  					<Row style={{ marginTop: 15 }}>
  						<Col span={24}>
  							<Form.TextArea
  								label={t('用户组速率限制 (JSON)')}
  								field={'ModelRequestRateLimitGroup'}
  								placeholder={t(
  									'请输入 JSON 格式的用户组限制，例如：\n{\n  "default": [200, 100],\n  "vip": [1000, 500]\n}',
  								)}
  								extraText={
  									<div>
  										<p>{t('说明:')}</p>
  										<ul>
  											<li>{t('使用 JSON 对象格式，键为用户组名 (字符串)，值为包含两个整数的数组 [总次数限制, 成功次数限制]。')}</li>
  											<li>{t('总次数限制: 周期内允许的总请求次数 (含失败)，0 代表不限制。')}</li>
  											<li>{t('成功次数限制: 周期内允许的成功请求次数 (HTTP < 400)，必须大于 0。')}</li>
  											<li>{t('此配置将优先于上方的全局限制设置。')}</li>
  											<li>{t('未在此处配置的用户组将使用全局限制。')}</li>
  											<li>{t('限制周期统一使用上方配置的“限制周期”值。')}</li>
  											<li>{t('输入无效的 JSON 将无法保存。')}</li>
  										</ul>
  									</div>
  								}
  								autosize={{ minRows: 5, maxRows: 15 }}
  								style={{ fontFamily: 'monospace' }}
  								onChange={(value) => {
  									setInputs({
  										...inputs,
  										ModelRequestRateLimitGroup: value,
  									});
  								}}
  							/>
  						</Col>
  					</Row>
  					<Row style={{ marginTop: 15 }}>
  						<Button size='default' onClick={onSubmit}>
  							{t('保存模型速率限制')}
  						</Button>
  					</Row>
  				</Form.Section>
  			</Form>
  		</Spin>
  	</>
  );
 }
