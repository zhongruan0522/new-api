import React from 'react';
import { Navigate } from 'react-router-dom';

const AuthRedirect = ({ children }) => {
  const user = localStorage.getItem('user');

  if (user) {
    return <Navigate to="/console" replace />;
  }

  return children;
};

export default AuthRedirect; 