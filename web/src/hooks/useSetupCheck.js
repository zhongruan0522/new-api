import { useContext, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { StatusContext } from '../context/Status';

/**
 * 自定义Hook：检查系统setup状态并进行重定向
 * @param {Object} options - 配置选项
 * @param {boolean} options.autoRedirect - 是否自动重定向，默认true
 * @param {string} options.setupPath - setup页面路径，默认'/setup'
 * @returns {Object} 返回setup状态信息
 */
export function useSetupCheck(options = {}) {
  const { autoRedirect = true, setupPath = '/setup' } = options;
  const [statusState] = useContext(StatusContext);
  const location = useLocation();

  const isSetupComplete = statusState?.status?.setup !== false;
  const needsSetup = !isSetupComplete && location.pathname !== setupPath;

  useEffect(() => {
    if (autoRedirect && needsSetup) {
      window.location.href = setupPath;
    }
  }, [autoRedirect, needsSetup, setupPath]);

  return {
    isSetupComplete,
    needsSetup,
    statusState,
    currentPath: location.pathname
  };
} 