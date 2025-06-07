import React, { useContext, useEffect, useState } from 'react';
import { Button, Typography, Tag } from '@douyinfe/semi-ui';
import { API, showError, isMobile } from '../../helpers';
import { StatusContext } from '../../context/Status';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { IconGithubLogo } from '@douyinfe/semi-icons';
import exampleImage from '/example.png';
import { Link } from 'react-router-dom';
import NoticeModal from '../../components/layout/NoticeModal';
import { Moonshot, OpenAI, XAI, Zhipu, Volcengine, Cohere, Claude, Gemini, Suno, Minimax, Wenxin, Spark, Qingyan, DeepSeek, Qwen, Midjourney, Grok, AzureAI, Hunyuan, Xinference } from '@lobehub/icons';

const { Text } = Typography;

const Home = () => {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');
  const [noticeVisible, setNoticeVisible] = useState(false);

  const isDemoSiteMode = statusState?.status?.demo_site_enabled || false;

  useEffect(() => {
    const checkNoticeAndShow = async () => {
      const lastCloseDate = localStorage.getItem('notice_close_date');
      const today = new Date().toDateString();
      if (lastCloseDate !== today) {
        try {
          const res = await API.get('/api/notice');
          const { success, data } = res.data;
          if (success && data && data.trim() !== '') {
            setNoticeVisible(true);
          }
        } catch (error) {
          console.error('获取公告失败:', error);
        }
      }
    };

    checkNoticeAndShow();
  }, []);

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);

      // 如果内容是 URL，则发送主题模式
      if (data.startsWith('https://')) {
        const iframe = document.querySelector('iframe');
        if (iframe) {
          const theme = localStorage.getItem('theme-mode') || 'light';
          iframe.onload = () => {
            iframe.contentWindow.postMessage({ themeMode: theme }, '*');
            iframe.contentWindow.postMessage({ lang: i18n.language }, '*');
          };
        }
      }
    } else {
      showError(message);
      setHomePageContent('加载首页内容失败...');
    }
    setHomePageContentLoaded(true);
  };

  useEffect(() => {
    displayHomePageContent().then();
  }, []);

  return (
    <div className="w-full overflow-x-hidden">
      <NoticeModal
        visible={noticeVisible}
        onClose={() => setNoticeVisible(false)}
        isMobile={isMobile()}
      />
      {homePageContentLoaded && homePageContent === '' ? (
        <div className="w-full overflow-x-hidden">
          {/* Banner 部分 */}
          <div className="w-full border-b border-semi-color-border min-h-[500px] md:h-[650px] lg:h-[750px] relative overflow-x-hidden">
            <div className="flex flex-col md:flex-row items-center justify-center h-full px-4 py-8 md:py-0">
              {/* 左侧内容区 */}
              <div className="flex-shrink-0 w-full md:w-[480px] md:mr-[60px] lg:mr-[120px] mb-8 md:mb-0">
                <div className="flex items-center gap-2 justify-center md:justify-start">
                  <h1 className="text-3xl md:text-4xl lg:text-5xl font-semibold text-semi-color-text-0 w-auto leading-normal md:leading-[67px]">
                    {statusState?.status?.system_name || 'New API'}
                  </h1>
                  {statusState?.status?.version && (
                    <Tag color='light-blue' size='large' shape='circle' className="ml-1">
                      {statusState.status.version}
                    </Tag>
                  )}
                </div>
                <p className="text-base md:text-lg text-semi-color-text-0 mt-4 md:mt-8 w-full md:w-[480px] leading-7 md:leading-8 text-center md:text-left">
                  {t('新一代大模型网关与AI资产管理系统，一键接入主流大模型，轻松管理您的AI资产')}
                </p>

                {/* 操作按钮 */}
                <div className="mt-6 md:mt-10 flex flex-wrap gap-4 justify-center md:justify-start">
                  <Link to="/console">
                    <Button theme="solid" type="primary" size="large" className="!rounded-3xl">
                      {t('开始使用')}
                    </Button>
                  </Link>
                  {isDemoSiteMode && (
                    <Button
                      size="large"
                      className="flex items-center !rounded-3xl"
                      icon={<IconGithubLogo />}
                      onClick={() => window.open('https://github.com/QuantumNous/new-api', '_blank')}
                    >
                      GitHub
                    </Button>
                  )}
                </div>

                {/* 框架兼容性图标 */}
                <div className="mt-8 md:mt-16">
                  <div className="flex items-center mb-3 justify-center md:justify-start">
                    <Text type="tertiary" className="text-lg md:text-xl font-light">
                      {t('支持众多的大模型供应商')}
                    </Text>
                  </div>
                  <div className="flex flex-wrap items-center relative mt-6 md:mt-8 gap-6 md:gap-8 justify-center md:justify-start">
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Moonshot size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <OpenAI size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <XAI size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Zhipu.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Volcengine.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Cohere.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Claude.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Gemini.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Suno size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Minimax.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Wenxin.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Spark.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Qingyan.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <DeepSeek.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Qwen.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Midjourney size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Grok size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <AzureAI.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Hunyuan.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Xinference.Color size={40} />
                    </div>
                    <div className="relative w-8 md:w-10 h-8 md:h-10 flex items-center justify-center">
                      <Typography.Text className="!text-2xl font-bold">30+</Typography.Text>
                    </div>
                  </div>
                </div>
              </div>

              {/* 右侧图片区域 - 在小屏幕上隐藏或调整位置 */}
              <div className="flex-shrink-0 relative md:mr-[-200px] lg:mr-[-400px] hidden md:block lg:min-w-[1100px]">
                <div className="absolute w-[320px] md:w-[500px] lg:w-[640px] h-[320px] md:h-[500px] lg:h-[640px] left-[-25px] md:left-[-40px] lg:left-[-50px] top-[-10px] md:top-[-15px] lg:top-[-20px] opacity-60"
                  style={{ filter: 'blur(120px)' }}>
                  <div className="absolute w-[320px] md:w-[400px] lg:w-[474px] h-[320px] md:h-[400px] lg:h-[474px] top-[80px] md:top-[100px] lg:top-[132px] bg-semi-color-primary rounded-full opacity-30"></div>
                  <div className="absolute w-[320px] md:w-[400px] lg:w-[474px] h-[320px] md:h-[400px] lg:h-[474px] left-[80px] md:left-[120px] lg:left-[166px] bg-semi-color-tertiary rounded-full opacity-30"></div>
                </div>

                <img
                  src={exampleImage}
                  alt="application demo"
                  className="relative h-[400px] md:h-[600px] lg:h-[721px] ml-[-15px] md:ml-[-20px] lg:ml-[-30px] mt-[-15px] md:mt-[-20px] lg:mt-[-30px]"
                />
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="overflow-x-hidden w-full">
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              className="w-full h-screen border-none"
            />
          ) : (
            <div
              className="text-base md:text-lg p-4 md:p-6 overflow-x-hidden"
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            ></div>
          )}
        </div>
      )}
    </div>
  );
};

export default Home;
