// contexts/User/index.jsx

import React, { useState, useEffect, useCallback } from 'react';
import { useLocation } from 'react-router-dom';
import { isMobile as getIsMobile } from '../../helpers/index.js';

export const StyleContext = React.createContext({
  dispatch: () => null,
});

export const StyleProvider = ({ children }) => {
  const location = useLocation();
  const initialIsMobile = getIsMobile();

  const initialPathname = location.pathname;
  let initialShowSiderValue = false;
  let initialInnerPaddingValue = false;

  if (initialPathname.includes('/console')) {
    initialShowSiderValue = !initialIsMobile;
    initialInnerPaddingValue = true;
  }

  const [state, setState] = useState({
    isMobile: initialIsMobile,
    showSider: initialShowSiderValue,
    siderCollapsed: false,
    shouldInnerPadding: initialInnerPaddingValue,
    manualSiderControl: false,
  });

  const dispatch = useCallback((action) => {
    if ('type' in action) {
      switch (action.type) {
        case 'TOGGLE_SIDER':
          setState((prev) => ({
            ...prev,
            showSider: !prev.showSider,
            manualSiderControl: true
          }));
          break;
        case 'SET_SIDER':
          setState((prev) => ({
            ...prev,
            showSider: action.payload,
            manualSiderControl: action.manual || false
          }));
          break;
        case 'SET_MOBILE':
          setState((prev) => ({ ...prev, isMobile: action.payload }));
          break;
        case 'SET_SIDER_COLLAPSED':
          setState((prev) => ({ ...prev, siderCollapsed: action.payload }));
          break;
        case 'SET_INNER_PADDING':
          setState((prev) => ({ ...prev, shouldInnerPadding: action.payload }));
          break;
        default:
          setState((prev) => ({ ...prev, ...action }));
      }
    } else {
      setState((prev) => ({ ...prev, ...action }));
    }
  }, []);

  useEffect(() => {
    const updateMobileStatus = () => {
      const currentIsMobile = getIsMobile();
      if (!currentIsMobile &&
        (location.pathname === '/console' || location.pathname.startsWith('/console/'))) {
        dispatch({ type: 'SET_SIDER', payload: true, manual: false });
      }
      dispatch({ type: 'SET_MOBILE', payload: currentIsMobile });
    };
    window.addEventListener('resize', updateMobileStatus);
    return () => window.removeEventListener('resize', updateMobileStatus);
  }, [dispatch, location.pathname]);

  useEffect(() => {
    if (state.isMobile && state.showSider && !state.manualSiderControl) {
      dispatch({ type: 'SET_SIDER', payload: false });
    }
  }, [state.isMobile, state.showSider, state.manualSiderControl, dispatch]);

  useEffect(() => {
    const currentPathname = location.pathname;
    const currentlyMobile = getIsMobile();

    if (currentPathname === '/console' || currentPathname.startsWith('/console/')) {
      dispatch({
        type: 'SET_SIDER',
        payload: !currentlyMobile,
        manual: false
      });
      dispatch({ type: 'SET_INNER_PADDING', payload: true });
    } else {
      dispatch({
        type: 'SET_SIDER',
        payload: false,
        manual: false
      });
      dispatch({ type: 'SET_INNER_PADDING', payload: false });
    }
  }, [location.pathname, dispatch]);

  useEffect(() => {
    const isCollapsed =
      localStorage.getItem('default_collapse_sidebar') === 'true';
    dispatch({ type: 'SET_SIDER_COLLAPSED', payload: isCollapsed });
  }, [dispatch]);

  return (
    <StyleContext.Provider value={[state, dispatch]}>
      {children}
    </StyleContext.Provider>
  );
};
