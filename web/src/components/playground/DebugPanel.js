import React from 'react';
import {
  Card,
  Typography,
  Tabs,
  TabPane,
  Button,
} from '@douyinfe/semi-ui';
import {
  Code,
  FileText,
  Zap,
  Clock,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

const DebugPanel = ({
  debugData,
  activeDebugTab,
  onActiveDebugTabChange,
  styleState,
  onCloseDebugPanel,
}) => {
  const { t } = useTranslation();

  return (
    <Card
      className="!rounded-2xl h-full flex flex-col"
      bodyStyle={{
        padding: styleState.isMobile ? '16px' : '24px',
        height: '100%',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      <div className="flex items-center justify-between mb-6 flex-shrink-0">
        <div className="flex items-center">
          <div className="w-10 h-10 rounded-full bg-gradient-to-r from-green-500 to-blue-500 flex items-center justify-center mr-3">
            <Code size={20} className="text-white" />
          </div>
          <Typography.Title heading={5} className="mb-0">
            {t('调试信息')}
          </Typography.Title>
        </div>

        {/* 移动端关闭按钮 */}
        {styleState.isMobile && onCloseDebugPanel && (
          <Button
            icon={<X size={16} />}
            onClick={onCloseDebugPanel}
            theme="borderless"
            type="tertiary"
            size="small"
            className="!rounded-lg"
          />
        )}
      </div>

      <div className="flex-1 overflow-hidden debug-panel">
        <Tabs
          type="line"
          className="h-full"
          style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
          activeKey={activeDebugTab}
          onChange={onActiveDebugTabChange}
        >
          <TabPane tab={
            <div className="flex items-center gap-2">
              <FileText size={16} />
              {t('请求体')}
            </div>
          } itemKey="request">
            <div className="h-full overflow-y-auto bg-gray-50 rounded-lg p-4 model-settings-scroll">
              {debugData.request ? (
                <pre className="debug-code text-gray-700 whitespace-pre-wrap break-words">
                  {JSON.stringify(debugData.request, null, 2)}
                </pre>
              ) : (
                <Typography.Text type="secondary" className="text-sm">
                  {t('暂无请求数据')}
                </Typography.Text>
              )}
            </div>
          </TabPane>

          <TabPane tab={
            <div className="flex items-center gap-2">
              <Zap size={16} />
              {t('响应内容')}
            </div>
          } itemKey="response">
            <div className="h-full overflow-y-auto bg-gray-50 rounded-lg p-4 model-settings-scroll">
              {debugData.response ? (
                <pre className="debug-code text-gray-700 whitespace-pre-wrap break-words">
                  {debugData.response}
                </pre>
              ) : (
                <Typography.Text type="secondary" className="text-sm">
                  {t('暂无响应数据')}
                </Typography.Text>
              )}
            </div>
          </TabPane>
        </Tabs>
      </div>

      {debugData.timestamp && (
        <div className="flex items-center gap-2 mt-4 pt-4 flex-shrink-0">
          <Clock size={14} className="text-gray-500" />
          <Typography.Text className="text-xs text-gray-500">
            {t('最后更新')}: {new Date(debugData.timestamp).toLocaleString()}
          </Typography.Text>
        </div>
      )}
    </Card>
  );
};

export default DebugPanel; 