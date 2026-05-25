import React from 'react';
import TicketsPage from '../../components/table/tickets';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { useTranslation } from 'react-i18next';

const Ticket = () => {
  const { t } = useTranslation();

  return (
    <div className='mt-[60px]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('工单管理')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('查看和管理您的工单')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <TicketsPage />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </div>
  );
};

export default Ticket;
