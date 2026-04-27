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

import React, { useEffect, useState, useMemo, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { Typography } from '@douyinfe/semi-ui';
import { getFooterHTML, getLogo, getSystemName } from '../../helpers';
import { StatusContext } from '../../context/Status';

const FooterBar = () => {
  const { t } = useTranslation();
  const [footer, setFooter] = useState(getFooterHTML());
  const systemName = getSystemName();
  const logo = getLogo();
  const [statusState] = useContext(StatusContext);

  const loadFooter = () => {
    let footer_html = localStorage.getItem('footer_html');
    if (footer_html) {
      setFooter(footer_html);
    }
  };

  const currentYear = new Date().getFullYear();

  const customFooter = useMemo(
    () => (
      <footer className='relative w-full flex items-center justify-center py-3 px-6'>
        <div className='flex flex-wrap items-center gap-2'>
          <Typography.Text className='text-sm !text-semi-color-text-1'>
            © {currentYear} {systemName}. {t('版权所有')}
          </Typography.Text>
        </div>
      </footer>
    ),
    [logo, systemName, t, currentYear],
  );

  useEffect(() => {
    loadFooter();
  }, []);

  return (
    <div className='w-full'>
      {footer ? (
        <div
          className='custom-footer'
          dangerouslySetInnerHTML={{ __html: footer }}
        ></div>
      ) : (
        customFooter
      )}
    </div>
  );
};

export default FooterBar;
