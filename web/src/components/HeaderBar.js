import React, { useContext, useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { UserContext } from '../context/User';
import { useSetTheme, useTheme } from '../context/Theme';
import { useTranslation } from 'react-i18next';
import { API, getLogo, getSystemName, showSuccess } from '../helpers';
import fireworks from 'react-fireworks';
import { CN, GB } from 'country-flag-icons/react/3x2';

import {
  IconClose,
  IconMenu,
  IconLanguage,
  IconChevronDown,
  IconSun,
  IconMoon,
  IconExit,
  IconUserSetting,
  IconCreditCard,
  IconKey,
} from '@douyinfe/semi-icons';
import {
  Avatar,
  Button,
  Dropdown,
  Tag,
  Typography,
  Skeleton,
} from '@douyinfe/semi-ui';
import { stringToColor } from '../helpers/render';
import { StatusContext } from '../context/Status/index.js';
import { StyleContext } from '../context/Style/index.js';

const HeaderBar = () => {
  const { t, i18n } = useTranslation();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [styleState, styleDispatch] = useContext(StyleContext);
  const [isLoading, setIsLoading] = useState(true);
  let navigate = useNavigate();
  const [currentLang, setCurrentLang] = useState(i18n.language);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const systemName = getSystemName();
  const logo = getLogo();
  const currentDate = new Date();
  const isNewYear = currentDate.getMonth() === 0 && currentDate.getDate() === 1;

  const isSelfUseMode = statusState?.status?.self_use_mode_enabled || false;
  const docsLink = statusState?.status?.docs_link || '';
  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;

  const theme = useTheme();
  const setTheme = useSetTheme();

  const mainNavLinks = [
    {
      text: t('首页'),
      itemKey: 'home',
      to: '/',
    },
    {
      text: t('控制台'),
      itemKey: 'detail',
      to: '/detail',
    },
    {
      text: t('定价'),
      itemKey: 'pricing',
      to: '/pricing',
    },
    ...(docsLink
      ? [
        {
          text: t('文档'),
          itemKey: 'docs',
          isExternal: true,
          externalLink: docsLink,
        },
      ]
      : []),
    {
      text: t('关于'),
      itemKey: 'about',
      to: '/about',
    },
  ];

  async function logout() {
    await API.get('/api/user/logout');
    showSuccess(t('注销成功!'));
    userDispatch({ type: 'logout' });
    localStorage.removeItem('user');
    navigate('/login');
    setMobileMenuOpen(false);
  }

  const handleNewYearClick = () => {
    fireworks.init('root', {});
    fireworks.start();
    setTimeout(() => {
      fireworks.stop();
    }, 3000);
  };

  useEffect(() => {
    if (theme === 'dark') {
      document.body.setAttribute('theme-mode', 'dark');
      document.documentElement.classList.add('dark');
    } else {
      document.body.removeAttribute('theme-mode');
      document.documentElement.classList.remove('dark');
    }

    const iframe = document.querySelector('iframe');
    if (iframe) {
      iframe.contentWindow.postMessage({ themeMode: theme }, '*');
    }

  }, [theme, isNewYear]);

  useEffect(() => {
    const handleLanguageChanged = (lng) => {
      setCurrentLang(lng);
      const iframe = document.querySelector('iframe');
      if (iframe) {
        iframe.contentWindow.postMessage({ lang: lng }, '*');
      }
    };

    i18n.on('languageChanged', handleLanguageChanged);
    return () => {
      i18n.off('languageChanged', handleLanguageChanged);
    };
  }, [i18n]);

  useEffect(() => {
    // 模拟加载用户状态的过程
    const timer = setTimeout(() => {
      setIsLoading(false);
    }, 500);
    return () => clearTimeout(timer);
  }, []);

  const handleLanguageChange = (lang) => {
    i18n.changeLanguage(lang);
    setMobileMenuOpen(false);
  };

  const handleNavLinkClick = (itemKey) => {
    if (itemKey === 'home') {
      styleDispatch({ type: 'SET_INNER_PADDING', payload: false });
      styleDispatch({ type: 'SET_SIDER', payload: false });
    } else {
      styleDispatch({ type: 'SET_INNER_PADDING', payload: true });
      if (!styleState.isMobile) {
        styleDispatch({ type: 'SET_SIDER', payload: true });
      }
    }
    setMobileMenuOpen(false);
  };

  const renderNavLinks = (isMobileView = false) =>
    mainNavLinks.map((link) => {
      const commonLinkClasses = isMobileView
        ? 'flex items-center gap-1 p-3 w-full text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md transition-colors font-semibold'
        : 'flex items-center gap-1 p-2 text-sm text-gray-700 dark:text-gray-300 hover:text-blue-600 dark:hover:text-blue-400 transition-colors rounded-md font-semibold';

      const linkContent = (
        <span>{link.text}</span>
      );

      if (link.isExternal) {
        return (
          <a
            key={link.itemKey}
            href={link.externalLink}
            target='_blank'
            rel='noopener noreferrer'
            className={commonLinkClasses}
            onClick={() => handleNavLinkClick(link.itemKey)}
          >
            {linkContent}
          </a>
        );
      }
      return (
        <Link
          key={link.itemKey}
          to={link.to}
          className={commonLinkClasses}
          onClick={() => handleNavLinkClick(link.itemKey)}
        >
          {linkContent}
        </Link>
      );
    });

  const renderUserArea = () => {
    if (isLoading) {
      return (
        <div className="flex items-center p-1 rounded-full bg-semi-color-fill-0 dark:bg-semi-color-fill-1">
          <Skeleton.Avatar size="extra-small" className="shadow-sm" />
          <div className="ml-1.5 mr-1">
            <Skeleton.Title style={{ width: styleState.isMobile ? 15 : 50, height: 12 }} />
          </div>
        </div>
      );
    }

    if (userState.user) {
      return (
        <Dropdown
          position="bottomRight"
          render={
            <Dropdown.Menu className="!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600">
              <Dropdown.Item
                onClick={() => {
                  navigate('/personal');
                  setMobileMenuOpen(false);
                }}
                className="!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white"
              >
                <div className="flex items-center gap-2">
                  <IconUserSetting size="small" className="text-gray-500 dark:text-gray-400" />
                  <span>{t('个人设置')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={() => {
                  navigate('/token');
                  setMobileMenuOpen(false);
                }}
                className="!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white"
              >
                <div className="flex items-center gap-2">
                  <IconKey size="small" className="text-gray-500 dark:text-gray-400" />
                  <span>{t('API令牌')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item
                onClick={() => {
                  navigate('/topup');
                  setMobileMenuOpen(false);
                }}
                className="!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-blue-500 dark:hover:!text-white"
              >
                <div className="flex items-center gap-2">
                  <IconCreditCard size="small" className="text-gray-500 dark:text-gray-400" />
                  <span>{t('钱包')}</span>
                </div>
              </Dropdown.Item>
              <Dropdown.Item onClick={logout} className="!px-3 !py-1.5 !text-sm !text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-red-500 dark:hover:!text-white">
                <div className="flex items-center gap-2">
                  <IconExit size="small" className="text-gray-500 dark:text-gray-400" />
                  <span>{t('退出')}</span>
                </div>
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <Button
            theme="borderless"
            type="tertiary"
            className="flex items-center gap-1.5 !p-1 !rounded-full hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2"
          >
            <Avatar
              size="extra-small"
              color={stringToColor(userState.user.username)}
              className="mr-1"
            >
              {userState.user.username[0].toUpperCase()}
            </Avatar>
            <span className="hidden md:inline">
              <Typography.Text className="!text-xs !font-medium !text-semi-color-text-1 dark:!text-gray-300 mr-1">
                {userState.user.username}
              </Typography.Text>
            </span>
            <IconChevronDown className="text-xs text-semi-color-text-2 dark:text-gray-400" />
          </Button>
        </Dropdown>
      );
    } else {
      const showRegisterButton = !isSelfUseMode;

      const commonSizingAndLayoutClass = "flex items-center justify-center !py-[10px] !px-1.5";

      const loginButtonSpecificStyling = "!bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-700 transition-colors";
      let loginButtonClasses = `${commonSizingAndLayoutClass} ${loginButtonSpecificStyling}`;

      let registerButtonClasses = `${commonSizingAndLayoutClass}`;

      const loginButtonTextSpanClass = "!text-xs !text-semi-color-text-1 dark:!text-gray-300 !p-1.5";
      const registerButtonTextSpanClass = "!text-xs !text-white !p-1.5";

      if (showRegisterButton) {
        if (styleState.isMobile) {
          loginButtonClasses += " !rounded-full";
        } else {
          loginButtonClasses += " !rounded-l-full !rounded-r-none";
        }
        registerButtonClasses += " !rounded-r-full !rounded-l-none";
      } else {
        loginButtonClasses += " !rounded-full";
      }

      return (
        <div className="flex items-center">
          <Link to="/login" onClick={() => handleNavLinkClick('login')} className="flex">
            <Button
              theme="borderless"
              type="tertiary"
              className={loginButtonClasses}
            >
              <span className={loginButtonTextSpanClass}>
                {t('登录')}
              </span>
            </Button>
          </Link>
          {showRegisterButton && (
            <div className="hidden md:block">
              <Link to="/register" onClick={() => handleNavLinkClick('register')} className="flex -ml-px">
                <Button
                  theme="solid"
                  type="primary"
                  className={registerButtonClasses}
                >
                  <span className={registerButtonTextSpanClass}>
                    {t('注册')}
                  </span>
                </Button>
              </Link>
            </div>
          )}
        </div>
      );
    }
  };

  return (
    <header className="bg-semi-color-bg-0 text-semi-color-text-0 sticky top-0 z-50 transition-colors duration-300">
      <div className="w-full px-4">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center">
            <div className="md:hidden">
              <Button
                icon={mobileMenuOpen ? <IconClose className="text-lg" /> : <IconMenu className="text-lg" />}
                aria-label={mobileMenuOpen ? t('关闭菜单') : t('打开菜单')}
                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                theme="borderless"
                type="tertiary"
                className="!p-2 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700"
              />
            </div>
            <Link to="/" onClick={() => handleNavLinkClick('home')} className="flex items-center gap-2 group ml-2">
              {isLoading ? (
                <Skeleton.Image className="h-7 md:h-8 !rounded-full" style={{ width: 32, height: 32 }} />
              ) : (
                <img src={logo} alt="logo" className="h-7 md:h-8 transition-transform duration-300 ease-in-out group-hover:scale-105" />
              )}
              <div className="hidden md:flex items-center gap-2">
                <div className="flex items-center gap-2">
                  {isLoading ? (
                    <Skeleton.Title style={{ width: 120, height: 24 }} />
                  ) : (
                    <Typography.Title heading={4} className="!text-lg !font-semibold !mb-0 
                                                          bg-gradient-to-r from-blue-500 to-purple-500 dark:from-blue-400 dark:to-purple-400
                                                          bg-clip-text text-transparent">
                      {systemName}
                    </Typography.Title>
                  )}
                  {(isSelfUseMode || isDemoSiteMode) && !isLoading && (
                    <Tag
                      color={isSelfUseMode ? 'purple' : 'blue'}
                      className="text-xs px-1.5 py-0.5 rounded whitespace-nowrap shadow-sm"
                      size="small"
                    >
                      {isSelfUseMode ? t('自用模式') : t('演示站点')}
                    </Tag>
                  )}
                </div>
              </div>
            </Link>
            {(isSelfUseMode || isDemoSiteMode) && !isLoading && (
              <div className="md:hidden">
                <Tag
                  color={isSelfUseMode ? 'purple' : 'blue'}
                  className="ml-2 text-xs px-1 py-0.5 rounded whitespace-nowrap shadow-sm"
                  size="small"
                >
                  {isSelfUseMode ? t('自用模式') : t('演示站点')}
                </Tag>
              </div>
            )}

            <nav className="hidden md:flex items-center gap-1 lg:gap-2 ml-6">
              {renderNavLinks()}
            </nav>
          </div>

          <div className="flex items-center gap-2 md:gap-3">
            {isNewYear && (
              <Dropdown
                position="bottomRight"
                render={
                  <Dropdown.Menu className="!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600">
                    <Dropdown.Item onClick={handleNewYearClick} className="!text-semi-color-text-0 hover:!bg-semi-color-fill-1 dark:!text-gray-200 dark:hover:!bg-gray-600">
                      Happy New Year!!! 🎉
                    </Dropdown.Item>
                  </Dropdown.Menu>
                }
              >
                <Button
                  theme="borderless"
                  type="tertiary"
                  icon={<span className="text-xl">🎉</span>}
                  aria-label="New Year"
                  className="!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 rounded-full"
                />
              </Dropdown>
            )}

            <Button
              icon={theme === 'dark' ? <IconSun size="large" className="text-yellow-500" /> : <IconMoon size="large" className="text-gray-300" />}
              aria-label={t('切换主题')}
              onClick={() => setTheme(theme === 'dark' ? false : true)}
              theme="borderless"
              type="tertiary"
              className="!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full bg-semi-color-fill-0 dark:bg-semi-color-fill-1 hover:bg-semi-color-fill-1 dark:hover:bg-semi-color-fill-2"
            />

            <Dropdown
              position="bottomRight"
              render={
                <Dropdown.Menu className="!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600">
                  <Dropdown.Item
                    onClick={() => handleLanguageChange('zh')}
                    className={`!flex !items-center !gap-2 !px-3 !py-1.5 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${currentLang === 'zh' ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold' : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'}`}
                  >
                    <CN title="中文" className="!w-5 !h-auto" />
                    <span>中文</span>
                  </Dropdown.Item>
                  <Dropdown.Item
                    onClick={() => handleLanguageChange('en')}
                    className={`!flex !items-center !gap-2 !px-3 !py-1.5 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${currentLang === 'en' ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold' : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'}`}
                  >
                    <GB title="English" className="!w-5 !h-auto" />
                    <span>English</span>
                  </Dropdown.Item>
                </Dropdown.Menu>
              }
            >
              <Button
                icon={<IconLanguage className="text-lg" />}
                aria-label={t('切换语言')}
                theme="borderless"
                type="tertiary"
                className="!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full bg-semi-color-fill-0 dark:bg-semi-color-fill-1 hover:bg-semi-color-fill-1 dark:hover:bg-semi-color-fill-2"
              />
            </Dropdown>

            {renderUserArea()}
          </div>
        </div>
      </div>

      <div className="md:hidden">
        <div
          className={`
            absolute top-16 left-0 right-0 bg-semi-color-bg-0 
            shadow-lg p-3
            transform transition-all duration-300 ease-in-out
            ${mobileMenuOpen ? 'translate-y-0 opacity-100 visible' : '-translate-y-4 opacity-0 invisible'}
          `}
        >
          <nav className="flex flex-col gap-1">
            {renderNavLinks(true)}
          </nav>
        </div>
      </div>
    </header>
  );
};

export default HeaderBar;
