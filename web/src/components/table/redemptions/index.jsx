import React from 'react';
import CardPro from '../../common/ui/CardPro';
import RedemptionsTable from './RedemptionsTable.jsx';
import RedemptionsActions from './RedemptionsActions.jsx';
import RedemptionsFilters from './RedemptionsFilters.jsx';
import RedemptionsDescription from './RedemptionsDescription.jsx';
import EditRedemptionModal from './modals/EditRedemptionModal';
import { useRedemptionsData } from '../../../hooks/redemptions/useRedemptionsData';

const RedemptionsPage = () => {
  const redemptionsData = useRedemptionsData();

  const {
    // Edit state
    showEdit,
    editingRedemption,
    closeEdit,
    refresh,

    // Actions state
    selectedKeys,
    setEditingRedemption,
    setShowEdit,
    batchCopyRedemptions,
    batchDeleteRedemptions,

    // Filters state
    formInitValues,
    setFormApi,
    searchRedemptions,
    loading,
    searching,

    // UI state
    compactMode,
    setCompactMode,

    // Translation
    t,
  } = redemptionsData;

  return (
    <>
      <EditRedemptionModal
        refresh={refresh}
        editingRedemption={editingRedemption}
        visiable={showEdit}
        handleClose={closeEdit}
      />

      <CardPro
        type="type1"
        descriptionArea={
          <RedemptionsDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={
          <div className="flex flex-col md:flex-row justify-between items-center gap-2 w-full">
            <RedemptionsActions
              selectedKeys={selectedKeys}
              setEditingRedemption={setEditingRedemption}
              setShowEdit={setShowEdit}
              batchCopyRedemptions={batchCopyRedemptions}
              batchDeleteRedemptions={batchDeleteRedemptions}
              t={t}
            />

            <div className="w-full md:w-full lg:w-auto order-1 md:order-2">
              <RedemptionsFilters
                formInitValues={formInitValues}
                setFormApi={setFormApi}
                searchRedemptions={searchRedemptions}
                loading={loading}
                searching={searching}
                t={t}
              />
            </div>
          </div>
        }
      >
        <RedemptionsTable {...redemptionsData} />
      </CardPro>
    </>
  );
};

export default RedemptionsPage; 