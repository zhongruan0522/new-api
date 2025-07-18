import React from 'react';
import CardPro from '../../common/ui/CardPro.js';
import ChannelsTable from './ChannelsTable.jsx';
import ChannelsActions from './ChannelsActions.jsx';
import ChannelsFilters from './ChannelsFilters.jsx';
import ChannelsTabs from './ChannelsTabs.jsx';
import { useChannelsData } from '../../../hooks/channels/useChannelsData.js';
import BatchTagModal from './modals/BatchTagModal.jsx';
import ModelTestModal from './modals/ModelTestModal.jsx';
import ColumnSelectorModal from './modals/ColumnSelectorModal.jsx';
import EditChannelModal from './modals/EditChannelModal.jsx';
import EditTagModal from './modals/EditTagModal.jsx';

const ChannelsPage = () => {
  const channelsData = useChannelsData();

  return (
    <>
      {/* Modals */}
      <ColumnSelectorModal {...channelsData} />
      <EditTagModal
        visible={channelsData.showEditTag}
        tag={channelsData.editingTag}
        handleClose={() => channelsData.setShowEditTag(false)}
        refresh={channelsData.refresh}
      />
      <EditChannelModal
        refresh={channelsData.refresh}
        visible={channelsData.showEdit}
        handleClose={channelsData.closeEdit}
        editingChannel={channelsData.editingChannel}
      />
      <BatchTagModal {...channelsData} />
      <ModelTestModal {...channelsData} />

      {/* Main Content */}
      <CardPro
        type="type3"
        tabsArea={<ChannelsTabs {...channelsData} />}
        actionsArea={<ChannelsActions {...channelsData} />}
        searchArea={<ChannelsFilters {...channelsData} />}
      >
        <ChannelsTable {...channelsData} />
      </CardPro>
    </>
  );
};

export default ChannelsPage; 