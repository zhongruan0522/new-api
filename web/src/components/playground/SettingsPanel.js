import React from 'react';
import {
  Card,
  Select,
  TextArea,
  Typography,
  Button,
  Switch,
  Divider,
  Banner,
} from '@douyinfe/semi-ui';
import {
  Sparkles,
  Users,
  ToggleLeft,
  X,
  AlertTriangle,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { renderGroupOption } from '../../helpers/render.js';
import ParameterControl from './ParameterControl';
import ImageUrlInput from './ImageUrlInput';
import ConfigManager from './ConfigManager';
import CustomRequestEditor from './CustomRequestEditor';

const SettingsPanel = ({
  inputs,
  parameterEnabled,
  models,
  groups,
  styleState,
  showDebugPanel,
  customRequestMode,
  customRequestBody,
  onInputChange,
  onParameterToggle,
  onCloseSettings,
  onConfigImport,
  onConfigReset,
  onCustomRequestModeChange,
  onCustomRequestBodyChange,
  previewPayload,
  messages,
}) => {
  const { t } = useTranslation();

  const currentConfig = {
    inputs,
    parameterEnabled,
    showDebugPanel,
    customRequestMode,
    customRequestBody,
  };

  return (
    <Card
      className={`!rounded-2xl h-full flex flex-col ${styleState.isMobile ? 'rounded-none border-none shadow-none' : ''}`}
      bodyStyle={{
        padding: styleState.isMobile ? '24px' : '24px 24px 16px 24px',
        height: '100%',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      {styleState.isMobile && (
        <div className="flex items-center justify-between mb-4">
          {/* 移动端显示配置管理下拉菜单和关闭按钮 */}
          <ConfigManager
            currentConfig={currentConfig}
            onConfigImport={onConfigImport}
            onConfigReset={onConfigReset}
            styleState={styleState}
            messages={messages}
          />
          <Button
            icon={<X size={16} />}
            onClick={onCloseSettings}
            theme="borderless"
            type="tertiary"
            size="small"
            className="!rounded-lg !text-gray-600 hover:!text-red-600 hover:!bg-red-50"
          />
        </div>
      )}

      <div className="space-y-6 overflow-y-auto flex-1 pr-2 model-settings-scroll">
        {/* 自定义请求体编辑器 */}
        <CustomRequestEditor
          customRequestMode={customRequestMode}
          customRequestBody={customRequestBody}
          onCustomRequestModeChange={onCustomRequestModeChange}
          onCustomRequestBodyChange={onCustomRequestBodyChange}
          defaultPayload={previewPayload}
        />

        {/* 分组选择 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <div className="flex items-center gap-2 mb-2">
            <Users size={16} className="text-gray-500" />
            <Typography.Text strong className="text-sm">
              {t('分组')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className="text-xs text-orange-600">
                (已在自定义模式中忽略)
              </Typography.Text>
            )}
          </div>
          <Select
            placeholder={t('请选择分组')}
            name='group'
            required
            selection
            onChange={(value) => onInputChange('group', value)}
            value={inputs.group}
            autoComplete='new-password'
            optionList={groups}
            renderOptionItem={renderGroupOption}
            style={{ width: '100%' }}
            className="!rounded-lg"
            disabled={customRequestMode}
          />
        </div>

        {/* 模型选择 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <div className="flex items-center gap-2 mb-2">
            <Sparkles size={16} className="text-gray-500" />
            <Typography.Text strong className="text-sm">
              {t('模型')}
            </Typography.Text>
            {customRequestMode && (
              <Typography.Text className="text-xs text-orange-600">
                (已在自定义模式中忽略)
              </Typography.Text>
            )}
          </div>
          <Select
            placeholder={t('请选择模型')}
            name='model'
            required
            selection
            searchPosition='dropdown'
            filter
            onChange={(value) => onInputChange('model', value)}
            value={inputs.model}
            autoComplete='new-password'
            optionList={models}
            className="!rounded-lg"
            disabled={customRequestMode}
          />
        </div>

        {/* 图片URL输入 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <ImageUrlInput
            imageUrls={inputs.imageUrls}
            imageEnabled={inputs.imageEnabled}
            onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
            onImageEnabledChange={(enabled) => onInputChange('imageEnabled', enabled)}
            disabled={customRequestMode}
          />
        </div>

        {/* 参数控制组件 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <ParameterControl
            inputs={inputs}
            parameterEnabled={parameterEnabled}
            onInputChange={onInputChange}
            onParameterToggle={onParameterToggle}
            disabled={customRequestMode}
          />
        </div>

        {/* 流式输出开关 */}
        <div className={customRequestMode ? 'opacity-50' : ''}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ToggleLeft size={16} className="text-gray-500" />
              <Typography.Text strong className="text-sm">
                流式输出
              </Typography.Text>
              {customRequestMode && (
                <Typography.Text className="text-xs text-orange-600">
                  (已在自定义模式中忽略)
                </Typography.Text>
              )}
            </div>
            <Switch
              checked={inputs.stream}
              onChange={(checked) => onInputChange('stream', checked)}
              checkedText="开"
              uncheckedText="关"
              size="small"
              disabled={customRequestMode}
            />
          </div>
        </div>
      </div>

      {/* 桌面端的配置管理放在底部 */}
      {!styleState.isMobile && (
        <div className="flex-shrink-0 pt-3">
          <ConfigManager
            currentConfig={currentConfig}
            onConfigImport={onConfigImport}
            onConfigReset={onConfigReset}
            styleState={styleState}
            messages={messages}
          />
        </div>
      )}
    </Card>
  );
};

export default SettingsPanel; 