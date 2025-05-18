import React, { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { getFooterHTML } from '../helpers';

const FooterBar = () => {
  const { t } = useTranslation();
  const [footer, setFooter] = useState(getFooterHTML());

  const loadFooter = () => {
    let footer_html = localStorage.getItem('footer_html');
    if (footer_html) {
      setFooter(footer_html);
    }
  };

  const defaultFooter = useMemo(() => (
    <div className='custom-footer'>
      <a
        href='https://github.com/Calcium-Ion/new-api'
        target='_blank'
        rel='noreferrer'
      >
        New API {import.meta.env.VITE_REACT_APP_VERSION}{' '}
      </a>
      {t('由')}{' '}
      <a href='https://github.com/Calcium-Ion' target='_blank' rel='noreferrer'>
        Calcium-Ion
      </a>{' '}
      {t('开发，基于')}{' '}
      <a
        href='https://github.com/songquanpeng/one-api'
        target='_blank'
        rel='noreferrer'
      >
        One API
      </a>
    </div>
  ), [t]);

  useEffect(() => {
    loadFooter();
  }, []);

  return (
    <div
      style={{
        textAlign: 'center',
        paddingBottom: '5px',
      }}
    >
      {footer ? (
        <div
          className='custom-footer'
          dangerouslySetInnerHTML={{ __html: footer }}
        ></div>
      ) : (
        defaultFooter
      )}
    </div>
  );
};

export default FooterBar;
