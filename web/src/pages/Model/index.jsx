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
import ModelsTable from '../../components/table/models';
import SectionPageLayout from '../../components/layout/SectionPageLayout';
import { useTranslation } from 'react-i18next';

const ModelPage = () => {
  const { t } = useTranslation();

  return (
    <div className='mt-[60px] h-[calc(100vh-60px)]'>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('AI模型配置')}</SectionPageLayout.Title>
        <SectionPageLayout.Description>
          {t('管理AI模型配置，包括模型名称、定价和可用性设置')}
        </SectionPageLayout.Description>
        <SectionPageLayout.Content>
          <ModelsTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>
    </div>
  );
};

export default ModelPage;
