import React from 'react';
import {
  Card,
  Select,
  TextArea,
  Typography,
  Button,
  Switch,
  Divider,
} from '@douyinfe/semi-ui';
import {
  Sparkles,
  Users,
  Type,
  ToggleLeft,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { renderGroupOption } from '../../helpers/render.js';
import ParameterControl from './ParameterControl';
import ImageUrlInput from './ImageUrlInput';
import ConfigManager from './ConfigManager';

const SettingsPanel = ({
  inputs,
  parameterEnabled,
  models,
  groups,
  systemPrompt,
  styleState,
  showDebugPanel,
  onInputChange,
  onParameterToggle,
  onSystemPromptChange,
  onCloseSettings,
  onConfigImport,
  onConfigReset,
}) => {
  const { t } = useTranslation();

  const currentConfig = {
    inputs,
    parameterEnabled,
    systemPrompt,
    showDebugPanel,
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
        {/* 分组选择 */}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Users size={16} className="text-gray-500" />
            <Typography.Text strong className="text-sm">
              {t('分组')}
            </Typography.Text>
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
          />
        </div>

        {/* 模型选择 */}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Sparkles size={16} className="text-gray-500" />
            <Typography.Text strong className="text-sm">
              {t('模型')}
            </Typography.Text>
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
          />
        </div>

        {/* 图片URL输入 */}
        <ImageUrlInput
          imageUrls={inputs.imageUrls}
          onImageUrlsChange={(urls) => onInputChange('imageUrls', urls)}
        />

        {/* 参数控制组件 */}
        <ParameterControl
          inputs={inputs}
          parameterEnabled={parameterEnabled}
          onInputChange={onInputChange}
          onParameterToggle={onParameterToggle}
        />

        {/* 流式输出开关 */}
        <div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ToggleLeft size={16} className="text-gray-500" />
              <Typography.Text strong className="text-sm">
                流式输出
              </Typography.Text>
            </div>
            <Switch
              checked={inputs.stream}
              onChange={(checked) => onInputChange('stream', checked)}
              checkedText="开"
              uncheckedText="关"
              size="small"
            />
          </div>
        </div>

        {/* System Prompt */}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Type size={16} className="text-gray-500" />
            <Typography.Text strong className="text-sm">
              System Prompt
            </Typography.Text>
          </div>
          <TextArea
            placeholder='System Prompt'
            name='system'
            required
            autoComplete='new-password'
            autosize
            defaultValue={systemPrompt}
            onChange={onSystemPromptChange}
            className="!rounded-lg"
            maxHeight={200}
          />
        </div>
      </div>

      {/* 桌面端的配置管理放在底部 */}
      {!styleState.isMobile && (
        <div className="flex-shrink-0 mt-4 pt-3">
          <ConfigManager
            currentConfig={currentConfig}
            onConfigImport={onConfigImport}
            onConfigReset={onConfigReset}
            styleState={styleState}
          />
        </div>
      )}
    </Card>
  );
};

export default SettingsPanel; 