import React from 'react';
import CardPro from '../../common/ui/CardPro';
import UsersTable from './UsersTable.jsx';
import UsersActions from './UsersActions.jsx';
import UsersFilters from './UsersFilters.jsx';
import UsersDescription from './UsersDescription.jsx';
import AddUserModal from './modals/AddUserModal.jsx';
import EditUserModal from './modals/EditUserModal.jsx';
import { useUsersData } from '../../../hooks/users/useUsersData';

const UsersPage = () => {
  const usersData = useUsersData();

  const {
    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    closeAddUser,
    closeEditUser,
    refresh,

    // Form state
    formInitValues,
    setFormApi,
    searchUsers,
    loadUsers,
    activePage,
    pageSize,
    groupOptions,
    loading,
    searching,

    // Description state
    compactMode,
    setCompactMode,

    // Translation
    t,
  } = usersData;

  return (
    <>
      <AddUserModal
        refresh={refresh}
        visible={showAddUser}
        handleClose={closeAddUser}
      />
      
      <EditUserModal
        refresh={refresh}
        visible={showEditUser}
        handleClose={closeEditUser}
        editingUser={editingUser}
      />

      <CardPro
        type="type1"
        descriptionArea={
          <UsersDescription
            compactMode={compactMode}
            setCompactMode={setCompactMode}
            t={t}
          />
        }
        actionsArea={
          <div className="flex flex-col md:flex-row justify-between items-center gap-2 w-full">
            <UsersActions
              setShowAddUser={setShowAddUser}
              t={t}
            />

            <UsersFilters
              formInitValues={formInitValues}
              setFormApi={setFormApi}
              searchUsers={searchUsers}
              loadUsers={loadUsers}
              activePage={activePage}
              pageSize={pageSize}
              groupOptions={groupOptions}
              loading={loading}
              searching={searching}
              t={t}
            />
          </div>
        }
      >
        <UsersTable {...usersData} />
      </CardPro>
    </>
  );
};

export default UsersPage; 