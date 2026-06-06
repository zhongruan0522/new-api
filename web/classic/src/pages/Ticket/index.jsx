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
