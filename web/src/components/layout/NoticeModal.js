import React, { useEffect, useState } from 'react';
import { Button, Modal, Empty } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../../helpers';
import { marked } from 'marked';
import { IllustrationNoContent, IllustrationNoContentDark } from '@douyinfe/semi-illustrations';

const NoticeModal = ({ visible, onClose, isMobile }) => {
  const { t } = useTranslation();
  const [noticeContent, setNoticeContent] = useState('');
  const [loading, setLoading] = useState(false);

  const handleCloseTodayNotice = () => {
    const today = new Date().toDateString();
    localStorage.setItem('notice_close_date', today);
    onClose();
  };

  const displayNotice = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/notice');
      const { success, message, data } = res.data;
      if (success) {
        if (data !== '') {
          const htmlNotice = marked.parse(data);
          setNoticeContent(htmlNotice);
        } else {
          setNoticeContent('');
        }
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      displayNotice();
    }
  }, [visible]);

  const renderContent = () => {
    if (loading) {
      return <div className="py-12"><Empty description={t('加载中...')} /></div>;
    }

    if (!noticeContent) {
      return (
        <div className="py-12">
          <Empty
            image={<IllustrationNoContent style={{ width: 150, height: 150 }} />}
            darkModeImage={<IllustrationNoContentDark style={{ width: 150, height: 150 }} />}
            description={t('暂无公告')}
          />
        </div>
      );
    }

    return (
      <div
        dangerouslySetInnerHTML={{ __html: noticeContent }}
        className="max-h-[60vh] overflow-y-auto pr-2"
        style={{
          scrollbarWidth: 'thin',
          scrollbarColor: 'var(--semi-color-tertiary) transparent'
        }}
      />
    );
  };

  return (
    <Modal
      title={t('系统公告')}
      visible={visible}
      onCancel={onClose}
      footer={(
        <div className="flex justify-end">
          <Button type='secondary' className='!rounded-full' onClick={handleCloseTodayNotice}>{t('今日关闭')}</Button>
          <Button type="primary" className='!rounded-full' onClick={onClose}>{t('关闭公告')}</Button>
        </div>
      )}
      size={isMobile ? 'full-width' : 'large'}
    >
      {renderContent()}
    </Modal>
  );
};

export default NoticeModal; 