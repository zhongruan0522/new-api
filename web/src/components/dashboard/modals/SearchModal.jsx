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

import React, { useRef, useMemo } from 'react';
import { Modal, Form, ButtonGroup, Button } from '@douyinfe/semi-ui';

const DATE_SHORTCUTS = [
  { key: 'today', label: '当日' },
  { key: 'week', label: '近一周' },
  { key: 'month', label: '近一月' },
  { key: 'currentMonth', label: '当月' },
];

const getShortcutTimestamp = (key) => {
  const now = new Date();
  switch (key) {
    case 'today':
      return new Date(now.getFullYear(), now.getMonth(), now.getDate());
    case 'week':
      return new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7);
    case 'month':
      return new Date(now.getFullYear(), now.getMonth() - 1, now.getDate());
    case 'currentMonth':
      return new Date(now.getFullYear(), now.getMonth(), 1);
    default:
      return null;
  }
};

const formatDateTime = (date) => {
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
};

const SearchModal = ({
  searchModalVisible,
  handleSearchConfirm,
  handleCloseModal,
  isMobile,
  isAdminUser,
  inputs,
  dataExportDefaultTime,
  timeOptions,
  handleInputChange,
  t,
}) => {
  const formRef = useRef();

  const FORM_FIELD_PROPS = {
    className: 'w-full mb-2 !rounded-lg',
  };

  const createFormField = (Component, props) => (
    <Component {...FORM_FIELD_PROPS} {...props} />
  );

  const { start_timestamp, end_timestamp, username } = inputs;

  // 判断当前选中的快捷按钮
  const activeShortcut = useMemo(() => {
    const now = new Date();
    for (const shortcut of DATE_SHORTCUTS) {
      const targetDate = getShortcutTimestamp(shortcut.key);
      if (!targetDate) continue;
      const expected = formatDateTime(targetDate);
      if (start_timestamp === expected) {
        return shortcut.key;
      }
    }
    return null;
  }, [start_timestamp]);

  const handleShortcutClick = (key) => {
    const date = getShortcutTimestamp(key);
    if (date) {
      const formatted = formatDateTime(date);
      handleInputChange(formatted, 'start_timestamp');
      // 同步更新 Form 内部状态，使 DatePicker 立即显示新值
      if (formRef.current) {
        formRef.current.formApi.setValue('start_timestamp', formatted);
      }
    }
  };

  return (
    <Modal
      title={t('搜索条件')}
      visible={searchModalVisible}
      onOk={handleSearchConfirm}
      onCancel={handleCloseModal}
      closeOnEsc={true}
      size={isMobile ? 'full-width' : 'small'}
      centered
    >
      <Form ref={formRef} layout='vertical' className='w-full'>
        <div className='mb-3'>
          <label className='text-sm font-medium text-gray-600 mb-2 block'>
            {t('快捷选择')}
          </label>
          <ButtonGroup size='small'>
            {DATE_SHORTCUTS.map((shortcut) => (
              <Button
                key={shortcut.key}
                theme={activeShortcut === shortcut.key ? 'solid' : 'light'}
                type={activeShortcut === shortcut.key ? 'primary' : 'tertiary'}
                onClick={() => handleShortcutClick(shortcut.key)}
              >
                {t(shortcut.label)}
              </Button>
            ))}
          </ButtonGroup>
        </div>

        {createFormField(Form.DatePicker, {
          field: 'start_timestamp',
          label: t('起始时间'),
          initValue: start_timestamp,
          value: start_timestamp,
          type: 'dateTime',
          name: 'start_timestamp',
          onChange: (value) => handleInputChange(value, 'start_timestamp'),
        })}

        {createFormField(Form.DatePicker, {
          field: 'end_timestamp',
          label: t('结束时间'),
          initValue: end_timestamp,
          value: end_timestamp,
          type: 'dateTime',
          name: 'end_timestamp',
          onChange: (value) => handleInputChange(value, 'end_timestamp'),
        })}

        {createFormField(Form.Select, {
          field: 'data_export_default_time',
          label: t('时间粒度'),
          initValue: dataExportDefaultTime,
          placeholder: t('时间粒度'),
          name: 'data_export_default_time',
          optionList: timeOptions,
          onChange: (value) =>
            handleInputChange(value, 'data_export_default_time'),
        })}

        {isAdminUser &&
          createFormField(Form.Input, {
            field: 'username',
            label: t('用户名称'),
            value: username,
            placeholder: t('可选值'),
            name: 'username',
            onChange: (value) => handleInputChange(value, 'username'),
          })}
      </Form>
    </Modal>
  );
};

export default SearchModal;
