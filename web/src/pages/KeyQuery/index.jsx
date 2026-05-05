import React from 'react';
import KeyQueryComponent from '../../components/key-query/KeyQuery';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { useTranslation } from 'react-i18next';

const KeyQuery = () => {
  const { t } = useTranslation();

  return (
    <div className='mt-[60px]' style={{ maxWidth: '80%', margin: '0 auto' }}>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Key消耗查询')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('通过API Key查询令牌余额和调用详情')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <KeyQueryComponent />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </div>
  );
};

export default KeyQuery;
