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

import React, { useEffect, useState, useCallback } from 'react';
import {
  Button,
  Table,
  Input,
  Space,
  Form,
  Card,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlus, IconDelete, IconSave } from '@douyinfe/semi-icons';
import {
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const PAGE_SIZE = 5;

// Normalize JSON for semantic comparison: parse → re-stringify compact
const normalizeJSON = (str) => {
  if (!str) return '';
  try {
    return JSON.stringify(JSON.parse(str));
  } catch {
    return str;
  }
};

// ============================================================
// 1. SimpleKeyValueTable — for GroupRatio & UserUsableGroups
// Data shape: { "key": "value", ... }
// ============================================================
function SimpleKeyValueTable({
  title,
  description,
  data,
  onChange,
  keyLabel,
  valueLabel,
  valuePlaceholder,
  valueMode = 'number', // 'number' | 'string'
}) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  // Track local edits to avoid cursor jumping
  const [editCache, setEditCache] = useState({});

  // Maintain insertion order as an array of keys
  const orderedKeys = Object.keys(data);
  const totalPages = Math.max(1, Math.ceil(orderedKeys.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);

  const handleAdd = () => {
    const trimmedKey = newKey.trim();
    if (!trimmedKey) {
      showError(t('请输入键名'));
      return;
    }
    if (data.hasOwnProperty(trimmedKey)) {
      showError(t('该键已存在'));
      return;
    }
    const val =
      valueMode === 'number'
        ? newValue === ''
          ? 0
          : Number(newValue)
        : newValue;
    if (valueMode === 'number' && (isNaN(val) || newValue === '')) {
      showError(t('请输入有效的数字'));
      return;
    }
    onChange({ ...data, [trimmedKey]: val });
    setNewKey('');
    setNewValue('');
  };

  const handleDelete = (key) => {
    const updated = { ...data };
    delete updated[key];
    onChange(updated);
    const remaining = Object.keys(updated).length;
    const newTotalPages = Math.max(1, Math.ceil(remaining / PAGE_SIZE));
    if (page > newTotalPages) setPage(newTotalPages);
  };

  const handleValueBlur = (key, rawValue) => {
    if (valueMode === 'number') {
      if (rawValue.trim() === '') {
        // Reject blank — revert to current data value
        setEditCache((prev) => {
          const next = { ...prev };
          delete next[key];
          return next;
        });
        return;
      }
      const val = Number(rawValue);
      if (isNaN(val)) {
        setEditCache((prev) => {
          const next = { ...prev };
          delete next[key];
          return next;
        });
        return;
      }
      onChange({ ...data, [key]: val });
    } else {
      onChange({ ...data, [key]: rawValue });
    }
    setEditCache((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  };

  const handleMoveUp = (index) => {
    if (index <= 0) return;
    const keys = [...orderedKeys];
    [keys[index - 1], keys[index]] = [keys[index], keys[index - 1]];
    const updated = {};
    keys.forEach((k) => (updated[k] = data[k]));
    onChange(updated);
  };

  const handleMoveDown = (index) => {
    if (index >= orderedKeys.length - 1) return;
    const keys = [...orderedKeys];
    [keys[index], keys[index + 1]] = [keys[index + 1], keys[index]];
    const updated = {};
    keys.forEach((k) => (updated[k] = data[k]));
    onChange(updated);
  };

  const pagedEntries = orderedKeys
    .slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE)
    .map((key) => [key, data[key]]);

  const dataSource = pagedEntries.map(([key, value], idx) => ({
    key,
    value: String(value),
    realIndex: (safePage - 1) * PAGE_SIZE + idx,
  }));

  const columns = [
    {
      title: keyLabel,
      dataIndex: 'key',
      key: 'key',
      render: (text) => <Typography.Text strong>{text}</Typography.Text>,
    },
    {
      title: valueLabel,
      dataIndex: 'value',
      key: 'value',
      render: (text, record) => (
        <Input
          value={editCache[record.key] ?? text}
          placeholder={valuePlaceholder || valueLabel}
          onChange={(v) =>
            setEditCache((prev) => ({ ...prev, [record.key]: v }))
          }
          onBlur={() =>
            handleValueBlur(record.key, editCache[record.key] ?? text)
          }
        />
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            disabled={record.realIndex === 0}
            onClick={() => handleMoveUp(record.realIndex)}
          >
            ↑
          </Button>
          <Button
            size='small'
            disabled={record.realIndex === orderedKeys.length - 1}
            onClick={() => handleMoveDown(record.realIndex)}
          >
            ↓
          </Button>
          <Button
            icon={<IconDelete />}
            type='danger'
            size='small'
            onClick={() => handleDelete(record.key)}
          />
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={title}
      style={{ marginBottom: 16 }}
      headerExtraContent={
        <Typography.Text type='tertiary' size='small'>
          {description}
        </Typography.Text>
      }
    >
      <Table
        columns={columns}
        dataSource={dataSource}
        rowKey='key'
        pagination={{
          currentPage: safePage,
          pageSize: PAGE_SIZE,
          total: orderedKeys.length,
          onPageChange: (p) => setPage(p),
          showTotal: (total) => t('共 {{total}} 条', { total }),
          showSizeChanger: false,
        }}
        size='small'
      />
      <Space style={{ marginTop: 8 }}>
        <Input
          placeholder={keyLabel}
          value={newKey}
          onChange={setNewKey}
          style={{ width: 150 }}
        />
        <Input
          placeholder={valuePlaceholder || valueLabel}
          value={newValue}
          onChange={setNewValue}
          style={{ width: 150 }}
          onPressEnter={handleAdd}
        />
        <Button icon={<IconPlus />} onClick={handleAdd}>
          {t('添加')}
        </Button>
      </Space>
    </Card>
  );
}

// ============================================================
// 2. NestedKeyValueTable — for GroupGroupRatio
// Data shape: { "outerKey": { "innerKey": number, ... }, ... }
// ============================================================
function NestedKeyValueTable({ title, description, data, onChange }) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [newOuterKey, setNewOuterKey] = useState('');
  const [innerAddKey, setInnerAddKey] = useState('');
  const [innerAddGroup, setInnerAddGroup] = useState('');
  const [innerAddValue, setInnerAddValue] = useState('');
  const [editCache, setEditCache] = useState({});

  // Flatten nested structure to rows, including empty groups
  const flatRows = [];
  for (const [outerKey, innerObj] of Object.entries(data)) {
    if (typeof innerObj === 'object' && innerObj !== null) {
      const innerEntries = Object.entries(innerObj);
      if (innerEntries.length === 0) {
        // Render a placeholder row for empty groups so they're visible
        flatRows.push({
          outerKey,
          innerKey: '',
          value: '',
          rowId: `${outerKey}::__empty__`,
          isEmpty: true,
        });
      } else {
        for (const [innerKey, val] of Object.entries(innerObj)) {
          flatRows.push({
            outerKey,
            innerKey,
            value: val,
            rowId: `${outerKey}::${innerKey}`,
            isEmpty: false,
          });
        }
      }
    }
  }

  const totalPages = Math.max(1, Math.ceil(flatRows.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const pagedRows = flatRows.slice(
    (safePage - 1) * PAGE_SIZE,
    safePage * PAGE_SIZE,
  );

  const handleAddGroup = () => {
    const trimmed = newOuterKey.trim();
    if (!trimmed) {
      showError(t('请输入用户分组名称'));
      return;
    }
    if (data.hasOwnProperty(trimmed)) {
      showError(t('该分组已存在'));
      return;
    }
    onChange({ ...data, [trimmed]: {} });
    setNewOuterKey('');
  };

  const handleDeleteGroup = (outerKey) => {
    const updated = { ...data };
    delete updated[outerKey];
    onChange(updated);
    adjustPage(updated);
  };

  const handleAddInner = () => {
    const group = innerAddKey.trim();
    const tokenGroup = innerAddGroup.trim();
    if (!group || !tokenGroup) {
      showError(t('请输入用户分组和令牌分组'));
      return;
    }
    if (!data.hasOwnProperty(group)) {
      showError(t('该用户分组不存在，请先添加'));
      return;
    }
    const val = Number(innerAddValue);
    if (isNaN(val) || innerAddValue.trim() === '') {
      showError(t('请输入有效的数字'));
      return;
    }
    const updated = { ...data };
    updated[group] = { ...(updated[group] || {}), [tokenGroup]: val };
    onChange(updated);
    setInnerAddGroup('');
    setInnerAddValue('');
  };

  const handleDeleteInner = (outerKey, innerKey, isEmpty) => {
    if (isEmpty) {
      // Deleting the placeholder row means delete the whole empty group
      handleDeleteGroup(outerKey);
      return;
    }
    const updated = { ...data };
    if (updated[outerKey]) {
      updated[outerKey] = { ...updated[outerKey] };
      delete updated[outerKey][innerKey];
      if (Object.keys(updated[outerKey]).length === 0) {
        delete updated[outerKey];
      }
    }
    onChange(updated);
    adjustPage(updated);
  };

  const handleEditBlur = (outerKey, innerKey, rowId) => {
    const rawValue = editCache[rowId];
    if (rawValue === undefined) return;
    if (rawValue.trim() === '') {
      // Reject blank values
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[rowId];
        return next;
      });
      return;
    }
    const val = Number(rawValue);
    if (isNaN(val)) {
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[rowId];
        return next;
      });
      return;
    }
    const updated = { ...data };
    updated[outerKey] = { ...(updated[outerKey] || {}), [innerKey]: val };
    onChange(updated);
    setEditCache((prev) => {
      const next = { ...prev };
      delete next[rowId];
      return next;
    });
  };

  const handleMoveUp = (index) => {
    if (index <= 0) return;
    // Move inner entry within its group
    const row = flatRows[index];
    const prevRow = flatRows[index - 1];
    if (row.outerKey !== prevRow.outerKey) return;
    const outerKey = row.outerKey;
    const innerKeys = Object.keys(data[outerKey] || {});
    const innerIdx1 = innerKeys.indexOf(row.innerKey);
    const innerIdx2 = innerKeys.indexOf(prevRow.innerKey);
    if (innerIdx1 < 0 || innerIdx2 < 0) return;
    const newKeys = [...innerKeys];
    [newKeys[innerIdx2], newKeys[innerIdx1]] = [newKeys[innerIdx1], newKeys[innerIdx2]];
    const newInner = {};
    newKeys.forEach((k) => (newInner[k] = data[outerKey][k]));
    onChange({ ...data, [outerKey]: newInner });
  };

  const handleMoveDown = (index) => {
    if (index >= flatRows.length - 1) return;
    const row = flatRows[index];
    const nextRow = flatRows[index + 1];
    if (row.outerKey !== nextRow.outerKey) return;
    const outerKey = row.outerKey;
    const innerKeys = Object.keys(data[outerKey] || {});
    const innerIdx1 = innerKeys.indexOf(row.innerKey);
    const innerIdx2 = innerKeys.indexOf(nextRow.innerKey);
    if (innerIdx1 < 0 || innerIdx2 < 0) return;
    const newKeys = [...innerKeys];
    [newKeys[innerIdx1], newKeys[innerIdx2]] = [newKeys[innerIdx2], newKeys[innerIdx1]];
    const newInner = {};
    newKeys.forEach((k) => (newInner[k] = data[outerKey][k]));
    onChange({ ...data, [outerKey]: newInner });
  };

  const adjustPage = (updatedData) => {
    const remaining = Object.keys(updatedData).reduce(
      (sum, k) => {
        const innerLen = Object.keys(updatedData[k] || {}).length;
        return sum + (innerLen === 0 ? 1 : innerLen);
      },
      0,
    );
    const newTotalPages = Math.max(1, Math.ceil(remaining / PAGE_SIZE));
    if (page > newTotalPages) setPage(newTotalPages);
  };

  const columns = [
    {
      title: t('用户分组'),
      dataIndex: 'outerKey',
      key: 'outerKey',
      render: (text, record, index) => {
        if (index > 0 && pagedRows[index - 1]?.outerKey === text) return null;
        const count = pagedRows.filter((r) => r.outerKey === text).length;
        return {
          children: (
            <Space>
              <Typography.Text strong>{text}</Typography.Text>
              <Button
                icon={<IconDelete />}
                type='danger'
                size='small'
                onClick={() => handleDeleteGroup(text)}
              />
            </Space>
          ),
          props: { rowSpan: count },
        };
      },
    },
    {
      title: t('令牌分组'),
      dataIndex: 'innerKey',
      key: 'innerKey',
      render: (text, record) =>
        record.isEmpty ? (
          <Typography.Text type='quaternary'>{t('（空）')}</Typography.Text>
        ) : (
          <Typography.Text>{text}</Typography.Text>
        ),
    },
    {
      title: t('倍率'),
      dataIndex: 'value',
      key: 'value',
      render: (text, record) =>
        record.isEmpty ? (
          <Typography.Text type='quaternary'>—</Typography.Text>
        ) : (
          <Input
            value={editCache[record.rowId] ?? String(text)}
            style={{ width: 100 }}
            onChange={(v) =>
              setEditCache((prev) => ({ ...prev, [record.rowId]: v }))
            }
            onBlur={() =>
              handleEditBlur(record.outerKey, record.innerKey, record.rowId)
            }
          />
        ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 180,
      render: (_, record, index) => (
        <Space>
          <Button
            size='small'
            disabled={record.isEmpty || index === 0 || pagedRows[index - 1]?.outerKey !== record.outerKey}
            onClick={() => handleMoveUp((safePage - 1) * PAGE_SIZE + index)}
          >
            ↑
          </Button>
          <Button
            size='small'
            disabled={record.isEmpty || index === pagedRows.length - 1 || pagedRows[index + 1]?.outerKey !== record.outerKey}
            onClick={() => handleMoveDown((safePage - 1) * PAGE_SIZE + index)}
          >
            ↓
          </Button>
          <Button
            icon={<IconDelete />}
            type='danger'
            size='small'
            onClick={() =>
              handleDeleteInner(record.outerKey, record.innerKey, record.isEmpty)
            }
          />
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={title}
      style={{ marginBottom: 16 }}
      headerExtraContent={
        <Typography.Text type='tertiary' size='small'>
          {description}
        </Typography.Text>
      }
    >
      <Table
        columns={columns}
        dataSource={pagedRows.map((row) => ({ ...row, key: row.rowId }))}
        rowKey='key'
        pagination={{
          currentPage: safePage,
          pageSize: PAGE_SIZE,
          total: flatRows.length,
          onPageChange: (p) => setPage(p),
          showTotal: (total) => t('共 {{total}} 条', { total }),
          showSizeChanger: false,
        }}
        size='small'
      />
      <Space style={{ marginTop: 8 }}>
        <Input
          placeholder={t('用户分组名称')}
          value={newOuterKey}
          onChange={setNewOuterKey}
          style={{ width: 150 }}
          onPressEnter={handleAddGroup}
        />
        <Button icon={<IconPlus />} onClick={handleAddGroup}>
          {t('添加分组')}
        </Button>
      </Space>
      <Space style={{ marginTop: 8, display: 'flex' }}>
        <Input
          placeholder={t('用户分组')}
          value={innerAddKey}
          onChange={setInnerAddKey}
          style={{ width: 120 }}
        />
        <Input
          placeholder={t('令牌分组名称')}
          value={innerAddGroup}
          onChange={setInnerAddGroup}
          style={{ width: 120 }}
        />
        <Input
          placeholder={t('倍率')}
          value={innerAddValue}
          onChange={setInnerAddValue}
          style={{ width: 100 }}
          onPressEnter={handleAddInner}
        />
        <Button icon={<IconPlus />} onClick={handleAddInner}>
          {t('添加倍率')}
        </Button>
      </Space>
    </Card>
  );
}

// ============================================================
// 3. SpecialUsableGroupTable — for group_special_usable_group
// Data shape: { "vip": { "+:premium": "描述", "-:default": "描述" }, ... }
// ============================================================
function SpecialUsableGroupTable({ title, description, data, onChange }) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [newOuterKey, setNewOuterKey] = useState('');
  const [addOuterKey, setAddOuterKey] = useState('');
  const [addGroup, setAddGroup] = useState('');
  const [addDesc, setAddDesc] = useState('');
  const [addOp, setAddOp] = useState('add');
  const [editCache, setEditCache] = useState({});

  // Flatten, including empty groups
  const flatRows = [];
  for (const [outerKey, innerObj] of Object.entries(data)) {
    if (typeof innerObj === 'object' && innerObj !== null) {
      const innerEntries = Object.entries(innerObj);
      if (innerEntries.length === 0) {
        flatRows.push({
          outerKey,
          rawKey: '',
          operation: '',
          group: '',
          value: '',
          rowId: `${outerKey}::__empty__`,
          isEmpty: true,
        });
      } else {
        for (const [rawKey, val] of Object.entries(innerObj)) {
          let operation = 'add';
          let group = rawKey;
          if (rawKey.startsWith('+:')) {
            operation = 'add';
            group = rawKey.slice(2);
          } else if (rawKey.startsWith('-:')) {
            operation = 'remove';
            group = rawKey.slice(2);
          }
          flatRows.push({
            outerKey,
            rawKey,
            operation,
            group,
            value: val,
            rowId: `${outerKey}::${rawKey}`,
            isEmpty: false,
          });
        }
      }
    }
  }

  const totalPages = Math.max(1, Math.ceil(flatRows.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const pagedRows = flatRows.slice(
    (safePage - 1) * PAGE_SIZE,
    safePage * PAGE_SIZE,
  );

  const handleAddGroup = () => {
    const trimmed = newOuterKey.trim();
    if (!trimmed) {
      showError(t('请输入用户分组名称'));
      return;
    }
    if (data.hasOwnProperty(trimmed)) {
      showError(t('该分组已存在'));
      return;
    }
    onChange({ ...data, [trimmed]: {} });
    setNewOuterKey('');
  };

  const handleDeleteGroup = (outerKey) => {
    const updated = { ...data };
    delete updated[outerKey];
    onChange(updated);
    adjustPage(updated);
  };

  const handleAddEntry = () => {
    const outer = addOuterKey.trim();
    const grp = addGroup.trim();
    if (!outer || !grp) {
      showError(t('请输入用户分组和分组名称'));
      return;
    }
    if (!data.hasOwnProperty(outer)) {
      showError(t('该用户分组不存在，请先添加'));
      return;
    }
    const prefix = addOp === 'remove' ? '-:' : '+:';
    const updated = { ...data };
    updated[outer] = { ...(updated[outer] || {}), [prefix + grp]: addDesc || grp };
    onChange(updated);
    setAddGroup('');
    setAddDesc('');
  };

  const handleDeleteEntry = (outerKey, rawKey, isEmpty) => {
    if (isEmpty) {
      handleDeleteGroup(outerKey);
      return;
    }
    const updated = { ...data };
    if (updated[outerKey]) {
      updated[outerKey] = { ...updated[outerKey] };
      delete updated[outerKey][rawKey];
      if (Object.keys(updated[outerKey]).length === 0) {
        delete updated[outerKey];
      }
    }
    onChange(updated);
    adjustPage(updated);
  };

  const handleFieldBlur = (
    outerKey,
    rawKey,
    field,
    newValue,
    operation,
    currentGroup,
    currentDesc,
  ) => {
    let newGroup = field === 'group' ? newValue.trim() : currentGroup;
    let newDesc = field === 'desc' ? newValue : currentDesc;

    // Validate: reject empty group name
    if (field === 'group' && !newGroup) {
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[`${outerKey}::${rawKey}::group`];
        delete next[`${outerKey}::${rawKey}::desc`];
        return next;
      });
      showError(t('分组名称不能为空'));
      return;
    }

    const prefix = operation === 'remove' ? '-:' : '+:';
    const newRawKey = prefix + newGroup;

    // Check if new key conflicts with an existing different entry
    if (newRawKey !== rawKey && data[outerKey]?.hasOwnProperty(newRawKey)) {
      showError(t('该分组名称已存在'));
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[`${outerKey}::${rawKey}::group`];
        delete next[`${outerKey}::${rawKey}::desc`];
        return next;
      });
      return;
    }

    const updated = { ...data };
    if (updated[outerKey]) {
      updated[outerKey] = { ...updated[outerKey] };
      if (newRawKey !== rawKey) {
        delete updated[outerKey][rawKey];
      }
      updated[outerKey][newRawKey] = newDesc;
    }
    onChange(updated);
    setEditCache((prev) => {
      const next = { ...prev };
      delete next[`${outerKey}::${rawKey}::group`];
      delete next[`${outerKey}::${rawKey}::desc`];
      return next;
    });
  };

  const handleMoveUp = (index) => {
    if (index <= 0) return;
    const row = flatRows[index];
    const prevRow = flatRows[index - 1];
    if (row.outerKey !== prevRow.outerKey || row.isEmpty || prevRow.isEmpty) return;
    const outerKey = row.outerKey;
    const innerKeys = Object.keys(data[outerKey] || {});
    const idx1 = innerKeys.indexOf(row.rawKey);
    const idx2 = innerKeys.indexOf(prevRow.rawKey);
    if (idx1 < 0 || idx2 < 0) return;
    const newKeys = [...innerKeys];
    [newKeys[idx2], newKeys[idx1]] = [newKeys[idx1], newKeys[idx2]];
    const newInner = {};
    newKeys.forEach((k) => (newInner[k] = data[outerKey][k]));
    onChange({ ...data, [outerKey]: newInner });
  };

  const handleMoveDown = (index) => {
    if (index >= flatRows.length - 1) return;
    const row = flatRows[index];
    const nextRow = flatRows[index + 1];
    if (row.outerKey !== nextRow.outerKey || row.isEmpty || nextRow.isEmpty) return;
    const outerKey = row.outerKey;
    const innerKeys = Object.keys(data[outerKey] || {});
    const idx1 = innerKeys.indexOf(row.rawKey);
    const idx2 = innerKeys.indexOf(nextRow.rawKey);
    if (idx1 < 0 || idx2 < 0) return;
    const newKeys = [...innerKeys];
    [newKeys[idx1], newKeys[idx2]] = [newKeys[idx2], newKeys[idx1]];
    const newInner = {};
    newKeys.forEach((k) => (newInner[k] = data[outerKey][k]));
    onChange({ ...data, [outerKey]: newInner });
  };

  const adjustPage = (updatedData) => {
    const remaining = Object.keys(updatedData).reduce((sum, k) => {
      const innerLen = Object.keys(updatedData[k] || {}).length;
      return sum + (innerLen === 0 ? 1 : innerLen);
    }, 0);
    const newTotalPages = Math.max(1, Math.ceil(remaining / PAGE_SIZE));
    if (page > newTotalPages) setPage(newTotalPages);
  };

  const operationLabel = {
    add: t('添加'),
    remove: t('移除'),
  };

  const columns = [
    {
      title: t('用户分组'),
      dataIndex: 'outerKey',
      key: 'outerKey',
      render: (text, record, index) => {
        if (index > 0 && pagedRows[index - 1]?.outerKey === text) return null;
        const count = pagedRows.filter((r) => r.outerKey === text).length;
        return {
          children: (
            <Space>
              <Typography.Text strong>{text}</Typography.Text>
              <Button
                icon={<IconDelete />}
                type='danger'
                size='small'
                onClick={() => handleDeleteGroup(text)}
              />
            </Space>
          ),
          props: { rowSpan: count },
        };
      },
    },
    {
      title: t('操作类型'),
      dataIndex: 'operation',
      key: 'operation',
      width: 80,
      render: (text, record) =>
        record.isEmpty ? (
          <Typography.Text type='quaternary'>—</Typography.Text>
        ) : (
          <Typography.Text
            type={record.operation === 'remove' ? 'danger' : 'success'}
          >
            {operationLabel[record.operation] || record.operation}
          </Typography.Text>
        ),
    },
    {
      title: t('分组名称'),
      dataIndex: 'group',
      key: 'group',
      render: (text, record) =>
        record.isEmpty ? (
          <Typography.Text type='quaternary'>{t('（空）')}</Typography.Text>
        ) : (
          <Input
            value={editCache[record.rowId + '::group'] ?? text}
            style={{ width: 120 }}
            onChange={(v) =>
              setEditCache((prev) => ({
                ...prev,
                [record.rowId + '::group']: v,
              }))
            }
            onBlur={() =>
              handleFieldBlur(
                record.outerKey,
                record.rawKey,
                'group',
                editCache[record.rowId + '::group'] ?? text,
                record.operation,
                text,
                record.value,
              )
            }
          />
        ),
    },
    {
      title: t('描述'),
      dataIndex: 'value',
      key: 'value',
      render: (text, record) =>
        record.isEmpty ? (
          <Typography.Text type='quaternary'>—</Typography.Text>
        ) : (
          <Input
            value={editCache[record.rowId + '::desc'] ?? text}
            style={{ width: 150 }}
            onChange={(v) =>
              setEditCache((prev) => ({
                ...prev,
                [record.rowId + '::desc']: v,
              }))
            }
            onBlur={() =>
              handleFieldBlur(
                record.outerKey,
                record.rawKey,
                'desc',
                editCache[record.rowId + '::desc'] ?? text,
                record.operation,
                record.group,
                text,
              )
            }
          />
        ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 180,
      render: (_, record, index) => (
        <Space>
          <Button
            size='small'
            disabled={record.isEmpty || index === 0 || pagedRows[index - 1]?.outerKey !== record.outerKey || pagedRows[index - 1]?.isEmpty}
            onClick={() => handleMoveUp((safePage - 1) * PAGE_SIZE + index)}
          >
            ↑
          </Button>
          <Button
            size='small'
            disabled={record.isEmpty || index === pagedRows.length - 1 || pagedRows[index + 1]?.outerKey !== record.outerKey || pagedRows[index + 1]?.isEmpty}
            onClick={() => handleMoveDown((safePage - 1) * PAGE_SIZE + index)}
          >
            ↓
          </Button>
          <Button
            icon={<IconDelete />}
            type='danger'
            size='small'
            onClick={() =>
              handleDeleteEntry(record.outerKey, record.rawKey, record.isEmpty)
            }
          />
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={title}
      style={{ marginBottom: 16 }}
      headerExtraContent={
        <Typography.Text type='tertiary' size='small'>
          {description}
        </Typography.Text>
      }
    >
      <Table
        columns={columns}
        dataSource={pagedRows.map((row) => ({ ...row, key: row.rowId }))}
        rowKey='key'
        pagination={{
          currentPage: safePage,
          pageSize: PAGE_SIZE,
          total: flatRows.length,
          onPageChange: (p) => setPage(p),
          showTotal: (total) => t('共 {{total}} 条', { total }),
          showSizeChanger: false,
        }}
        size='small'
      />
      <Space style={{ marginTop: 8 }}>
        <Input
          placeholder={t('用户分组名称')}
          value={newOuterKey}
          onChange={setNewOuterKey}
          style={{ width: 150 }}
          onPressEnter={handleAddGroup}
        />
        <Button icon={<IconPlus />} onClick={handleAddGroup}>
          {t('添加分组')}
        </Button>
      </Space>
      {Object.keys(data).length > 0 && (
        <Space style={{ marginTop: 8, display: 'flex' }}>
          <Input
            placeholder={t('用户分组')}
            value={addOuterKey}
            onChange={setAddOuterKey}
            style={{ width: 100 }}
          />
          <Input
            placeholder={t('分组名称')}
            value={addGroup}
            onChange={setAddGroup}
            style={{ width: 100 }}
          />
          <Input
            placeholder={t('描述')}
            value={addDesc}
            onChange={setAddDesc}
            style={{ width: 100 }}
          />
          <Button
            size='small'
            type={addOp === 'add' ? 'primary' : 'warning'}
            onClick={() => setAddOp(addOp === 'add' ? 'remove' : 'add')}
          >
            {addOp === 'add' ? t('添加') : t('移除')}
          </Button>
          <Button icon={<IconPlus />} onClick={handleAddEntry}>
            {t('添加条目')}
          </Button>
        </Space>
      )}
    </Card>
  );
}

// ============================================================
// 4. StringListTable — for AutoGroups
// Data shape: ["g1", "g2", "g3"]
// ============================================================
function StringListTable({ title, description, data, onChange }) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const [newItem, setNewItem] = useState('');
  const [editCache, setEditCache] = useState({});

  const list = Array.isArray(data) ? data : [];
  const totalPages = Math.max(1, Math.ceil(list.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);

  const handleAdd = () => {
    const trimmed = newItem.trim();
    if (!trimmed) {
      showError(t('请输入分组名称'));
      return;
    }
    if (list.includes(trimmed)) {
      showError(t('该分组已存在'));
      return;
    }
    onChange([...list, trimmed]);
    setNewItem('');
  };

  const handleDelete = (index) => {
    const updated = [...list];
    updated.splice(index, 1);
    onChange(updated);
    const newTotalPages = Math.max(1, Math.ceil(updated.length / PAGE_SIZE));
    if (page > newTotalPages) setPage(newTotalPages);
  };

  const handleMoveUp = (index) => {
    if (index <= 0) return;
    const updated = [...list];
    [updated[index - 1], updated[index]] = [updated[index], updated[index - 1]];
    onChange(updated);
  };

  const handleMoveDown = (index) => {
    if (index >= list.length - 1) return;
    const updated = [...list];
    [updated[index], updated[index + 1]] = [updated[index + 1], updated[index]];
    onChange(updated);
  };

  const handleEditBlur = (index, rawValue) => {
    const trimmed = rawValue.trim();
    if (!trimmed) {
      // Reject blank — revert
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[index];
        return next;
      });
      showError(t('分组名称不能为空'));
      return;
    }
    if (trimmed !== list[index] && list.includes(trimmed)) {
      showError(t('该分组已存在'));
      setEditCache((prev) => {
        const next = { ...prev };
        delete next[index];
        return next;
      });
      return;
    }
    const updated = [...list];
    updated[index] = trimmed;
    onChange(updated);
    setEditCache((prev) => {
      const next = { ...prev };
      delete next[index];
      return next;
    });
  };

  const pagedItems = list.slice(
    (safePage - 1) * PAGE_SIZE,
    safePage * PAGE_SIZE,
  );

  const dataSource = pagedItems.map((name, idx) => ({
    key: (safePage - 1) * PAGE_SIZE + idx,
    name,
    realIndex: (safePage - 1) * PAGE_SIZE + idx,
  }));

  const columns = [
    {
      title: t('序号'),
      key: 'index',
      width: 60,
      render: (_, record) => record.realIndex + 1,
    },
    {
      title: t('分组名称'),
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Input
          value={editCache[record.realIndex] ?? text}
          style={{ width: 150 }}
          onChange={(v) =>
            setEditCache((prev) => ({ ...prev, [record.realIndex]: v }))
          }
          onBlur={() =>
            handleEditBlur(record.realIndex, editCache[record.realIndex] ?? text)
          }
        />
      ),
    },
    {
      title: t('操作'),
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space>
          <Button
            size='small'
            disabled={record.realIndex === 0}
            onClick={() => handleMoveUp(record.realIndex)}
          >
            ↑
          </Button>
          <Button
            size='small'
            disabled={record.realIndex === list.length - 1}
            onClick={() => handleMoveDown(record.realIndex)}
          >
            ↓
          </Button>
          <Button
            icon={<IconDelete />}
            type='danger'
            size='small'
            onClick={() => handleDelete(record.realIndex)}
          />
        </Space>
      ),
    },
  ];

  return (
    <Card
      title={title}
      style={{ marginBottom: 16 }}
      headerExtraContent={
        <Typography.Text type='tertiary' size='small'>
          {description}
        </Typography.Text>
      }
    >
      <Table
        columns={columns}
        dataSource={dataSource}
        rowKey='key'
        pagination={{
          currentPage: safePage,
          pageSize: PAGE_SIZE,
          total: list.length,
          onPageChange: (p) => setPage(p),
          showTotal: (total) => t('共 {{total}} 条', { total }),
          showSizeChanger: false,
        }}
        size='small'
      />
      <Space style={{ marginTop: 8 }}>
        <Input
          placeholder={t('分组名称')}
          value={newItem}
          onChange={setNewItem}
          style={{ width: 200 }}
          onPressEnter={handleAdd}
        />
        <Button icon={<IconPlus />} onClick={handleAdd}>
          {t('添加')}
        </Button>
      </Space>
    </Card>
  );
}

// ============================================================
// Main component
// ============================================================
export default function GroupRatioSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    GroupRatio: '',
    UserUsableGroups: '',
    GroupGroupRatio: '',
    'group_ratio_setting.group_special_usable_group': '',
    AutoGroups: '',
    DefaultUseAutoGroup: false,
  });
  const [inputsRow, setInputsRow] = useState(inputs);

  const parseJSON = (str, fallback) => {
    if (!str) return fallback;
    try {
      return JSON.parse(str);
    } catch {
      return fallback;
    }
  };

  const groupRatioData = parseJSON(inputs.GroupRatio, {});
  const userUsableGroupsData = parseJSON(inputs.UserUsableGroups, {});
  const groupGroupRatioData = parseJSON(inputs.GroupGroupRatio, {});
  const specialUsableGroupData = parseJSON(
    inputs['group_ratio_setting.group_special_usable_group'],
    {},
  );
  const autoGroupsData = parseJSON(inputs.AutoGroups, []);

  const updateField = useCallback((field, value) => {
    const serialized =
      typeof value === 'string' ? value : JSON.stringify(value);
    setInputs((prev) => ({ ...prev, [field]: serialized }));
  }, []);

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.options]);

  async function onSubmit() {
    // Use semantic JSON comparison: normalize both sides before comparing
    const changedFields = [];
    for (const key of Object.keys(inputs)) {
      const currentVal =
        typeof inputs[key] === 'boolean'
          ? String(inputs[key])
          : inputs[key];
      const originalVal =
        typeof inputsRow[key] === 'boolean'
          ? String(inputsRow[key])
          : inputsRow[key];

      // For boolean, simple string compare
      if (typeof inputs[key] === 'boolean') {
        if (currentVal !== originalVal) {
          changedFields.push({ key, value: currentVal });
        }
        continue;
      }

      // For JSON fields, normalize both sides for comparison
      const normCurrent = normalizeJSON(currentVal);
      const normOriginal = normalizeJSON(originalVal);
      if (normCurrent !== normOriginal) {
        // Send compact JSON to backend
        changedFields.push({ key, value: normCurrent });
      }
    }

    if (!changedFields.length) {
      return showWarning(t('你似乎并没有修改什么'));
    }

    setLoading(true);
    try {
      const requestQueue = changedFields.map((item) =>
        API.put('/api/option/', { key: item.key, value: item.value }),
      );

      const res = await Promise.all(requestQueue);

      if (res.includes(undefined)) {
        return showError(
          requestQueue.length > 1
            ? t('部分保存失败，请重试')
            : t('保存失败'),
        );
      }

      for (let i = 0; i < res.length; i++) {
        if (!res[i].data.success) {
          return showError(res[i].data.message);
        }
      }

      showSuccess(t('保存成功'));
      props.refresh();
    } catch (error) {
      console.error('Unexpected error:', error);
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ opacity: loading ? 0.6 : 1, pointerEvents: loading ? 'none' : 'auto' }}>
      <SimpleKeyValueTable
        title={t('分组倍率')}
        description={t('分组倍率设置，键为分组名称，值为倍率')}
        data={groupRatioData}
        keyLabel={t('分组名称')}
        valueLabel={t('倍率')}
        valuePlaceholder={t('倍率，如 0.5')}
        valueMode='number'
        onChange={(data) => updateField('GroupRatio', data)}
      />

      <SimpleKeyValueTable
        title={t('用户可选分组')}
        description={t('用户新建令牌时可选的分组，键为分组名称，值为分组描述')}
        data={userUsableGroupsData}
        keyLabel={t('分组名称')}
        valueLabel={t('分组描述')}
        valuePlaceholder={t('描述')}
        valueMode='string'
        onChange={(data) => updateField('UserUsableGroups', data)}
      />

      <NestedKeyValueTable
        title={t('分组特殊倍率')}
        description={t('键为用户分组，值为令牌分组→倍率的映射')}
        data={groupGroupRatioData}
        onChange={(data) => updateField('GroupGroupRatio', data)}
      />

      <SpecialUsableGroupTable
        title={t('分组特殊可用分组')}
        description={t(
          '键为用户分组，值为操作映射。+: 添加分组，-: 移除分组',
        )}
        data={specialUsableGroupData}
        onChange={(data) =>
          updateField('group_ratio_setting.group_special_usable_group', data)
        }
      />

      <StringListTable
        title={t('自动分组auto，从第一个开始选择')}
        description={t('按顺序排列的自动分组列表')}
        data={Array.isArray(autoGroupsData) ? autoGroupsData : []}
        onChange={(data) => updateField('AutoGroups', data)}
      />

      <Card style={{ marginBottom: 16 }}>
        <Form.Switch
          label={t(
            '创建令牌默认选择auto分组，初始令牌也将设为auto（否则留空，为用户默认分组）',
          )}
          checked={inputs.DefaultUseAutoGroup}
          onChange={(value) =>
            setInputs({ ...inputs, DefaultUseAutoGroup: value })
          }
        />
      </Card>

      <Button
        type='primary'
        icon={<IconSave />}
        onClick={onSubmit}
        loading={loading}
        size='large'
      >
        {t('保存分组倍率设置')}
      </Button>
    </div>
  );
}
