import React from 'react';
import TokensTable from '../../components/TokensTable';
import { useTranslation } from 'react-i18next';
const Token = () => {
  const { t } = useTranslation();
  return (
    <>
      <TokensTable />
    </>
  );
};

export default Token;
