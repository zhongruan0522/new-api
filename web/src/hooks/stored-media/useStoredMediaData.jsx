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

import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  copy,
  getTodayStartTimestamp,
  isAdmin,
  showError,
  showSuccess,
  timestamp2string,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';

export const useStoredMediaData = () => {
  const { t } = useTranslation();
  const isAdminUser = isAdmin();

  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [total, setTotal] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);

  const [selectedKeys, setSelectedKeys] = useState([]);

  const [formApi, setFormApi] = useState(null);
  const now = useMemo(() => new Date(), []);
  const formInitValues = {
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000 + 3600),
    ],
  };

  const [viewModalVisible, setViewModalVisible] = useState(false);
  const [viewModalData, setViewModalData] = useState(null);
  const [viewLoadingKey, setViewLoadingKey] = useState('');

  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    let start_timestamp = timestamp2string(getTodayStartTimestamp());
    let end_timestamp = timestamp2string(now.getTime() / 1000 + 3600);

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      start_timestamp = formValues.dateRange[0];
      end_timestamp = formValues.dateRange[1];
    }

    return { start_timestamp, end_timestamp };
  };

  const enrichItems = (rawItems = []) => {
    return (rawItems || []).map((it) => ({
      ...it,
      key: `${it.media_type || 'unknown'}:${it.id}`,
      base_preview: it.base_preview || '',
      base_truncated: !!it.base_truncated,
    }));
  };

  const syncPageData = (payload) => {
    const newItems = enrichItems(payload.items || []);
    setItems(newItems);
    setTotal(payload.total || 0);
    setActivePage(payload.page || 1);
    setPageSize(payload.page_size || pageSize);
  };

  const buildListUrl = (page, size) => {
    const { start_timestamp, end_timestamp } = getFormValues();
    const localStart = Math.floor(Date.parse(start_timestamp) / 1000);
    const localEnd = Math.floor(Date.parse(end_timestamp) / 1000);

    const base = isAdminUser ? '/api/stored_media/' : '/api/stored_media/self/';
    const url = `${base}?p=${page}&page_size=${size}&start_timestamp=${localStart}&end_timestamp=${localEnd}`;
    return encodeURI(url);
  };

  const loadItems = async (page = 1, size = pageSize) => {
    setLoading(true);
    try {
      const url = buildListUrl(page, size);
      const res = await API.get(url);
      const { success, message, data } = res.data || {};
      if (success) {
        syncPageData(data);
      } else {
        showError(message || t('加载失败'));
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  };

  const refresh = async () => {
    setActivePage(1);
    await loadItems(1, pageSize);
    setSelectedKeys([]);
  };

  const handlePageChange = (page) => {
    setActivePage(page);
    loadItems(page, pageSize).then();
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    await loadItems(1, size);
    setSelectedKeys([]);
  };

  const rowSelection = {
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedKeys(selectedRows);
    },
  };

  const copyText = async (text) => {
    if (!text) return;
    if (await copy(text)) {
      showSuccess(t('复制成功'));
    } else {
      Modal.error({
        title: t('无法复制到剪贴板，请手动复制'),
        content: text,
        size: 'large',
      });
    }
  };

  const openViewModal = async (record) => {
    if (!record?.id || !record?.media_type) return;
    const key = record.key || `${record.media_type}:${record.id}`;
    setViewLoadingKey(key);

    try {
      const res = await API.get(`/api/stored_media/${record.media_type}/${record.id}`);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载失败'));
        return;
      }

      setItems((prev) =>
        (prev || []).map((it) => {
          if (it.key !== key) return it;
          return {
            ...it,
            base_preview: data?.base_preview || it.base_preview || '',
            base_truncated: !!data?.base_truncated,
            url: data?.url || it.url,
          };
        }),
      );

      setViewModalData(data || null);
      setViewModalVisible(true);
    } catch (e) {
      showError(e);
    } finally {
      setViewLoadingKey('');
    }
  };

  const closeViewModal = () => {
    setViewModalVisible(false);
    setTimeout(() => setViewModalData(null), 300);
  };

  const deleteOne = async (record) => {
    if (!record?.id || !record?.media_type) return;
    setLoading(true);
    try {
      const res = await API.delete(
        `/api/stored_media/${record.media_type}/${record.id}`,
      );
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('删除成功'));
        await refresh();
      } else {
        showError(message || t('删除失败'));
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  };

  const batchDelete = async () => {
    if (!selectedKeys || selectedKeys.length === 0) {
      showError(t('请先选择要删除的文件'));
      return;
    }

    setLoading(true);
    try {
      const items = selectedKeys
        .map((r) => ({
          id: r.id,
          media_type: r.media_type,
        }))
        .filter((x) => x.id && x.media_type);

      const res = await API.post('/api/stored_media/batch', { items });
      const { success, message, data } = res.data || {};
      if (success) {
        const count = data || 0;
        showSuccess(t('批量删除成功') + ` (${count})`);
        await refresh();
      } else {
        showError(message || t('批量删除失败'));
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  };

  // Initialize
  useEffect(() => {
    const localPageSize =
      parseInt(localStorage.getItem('page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadItems(1, localPageSize).then();
  }, []);

  return {
    t,
    isAdminUser,

    items,
    loading,
    activePage,
    total,
    pageSize,

    selectedKeys,
    setSelectedKeys,
    rowSelection,

    formInitValues,
    formApi,
    setFormApi,

    refresh,
    handlePageChange,
    handlePageSizeChange,

    copyText,
    openViewModal,
    closeViewModal,
    viewModalVisible,
    viewModalData,
    viewLoadingKey,

    deleteOne,
    batchDelete,
  };
};

