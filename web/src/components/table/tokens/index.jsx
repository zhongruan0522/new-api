import React from 'react';
import CardPro from '../../common/ui/CardPro';
import TokensTable from './TokensTable.jsx';
import TokensActions from './TokensActions.jsx';
import TokensFilters from './TokensFilters.jsx';
import TokensDescription from './TokensDescription.jsx';
import EditToken from '../../../pages/Token/EditToken';
import { useTokensData } from '../../../hooks/tokens/useTokensData';

const TokensPage = () => {
  const tokensData = useTokensData();

  const {
    // Edit state
    showEdit,
    editingToken,
    closeEdit,
    refresh,

    // Actions state
    selectedKeys,
    setEditingToken,
    setShowEdit,
    batchDeleteTokens,
    copyText,

    // Filters state
    formInitValues,
    setFormApi,
    searchTokens,
    loading,
    searching,

    // Description state
    compactMode,
    setCompactMode,

    // Translation
    t,
  } = tokensData;

  return (
    <>
      <EditToken
        refresh={refresh}
        editingToken={editingToken}
        visiable={showEdit}
        handleClose={closeEdit}
      />

      <CardPro
        type="type1"
        descriptionArea={
          <TokensDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={
          <div className="flex flex-col md:flex-row justify-between items-center gap-2 w-full">
            <TokensActions
              selectedKeys={selectedKeys}
              setEditingToken={setEditingToken}
              setShowEdit={setShowEdit}
              batchDeleteTokens={batchDeleteTokens}
              copyText={copyText}
              t={t}
            />

            <div className="w-full md:w-full lg:w-auto order-1 md:order-2">
              <TokensFilters
                formInitValues={formInitValues}
                setFormApi={setFormApi}
                searchTokens={searchTokens}
                loading={loading}
                searching={searching}
                t={t}
              />
            </div>
          </div>
        }
      >
        <TokensTable {...tokensData} />
      </CardPro>
    </>
  );
};

export default TokensPage; 