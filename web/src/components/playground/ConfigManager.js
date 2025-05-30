import React, { useRef } from 'react';
import {
  Button,
  Typography,
  Toast,
  Modal,
  Dropdown,
} from '@douyinfe/semi-ui';
import {
  Download,
  Upload,
  RotateCcw,
  Settings2,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { exportConfig, importConfig, clearConfig, hasStoredConfig, getConfigTimestamp } from './configStorage';

const ConfigManager = ({
  currentConfig,
  onConfigImport,
  onConfigReset,
  styleState,
}) => {
  const { t } = useTranslation();
  const fileInputRef = useRef(null);

  const handleExport = () => {
    try {
      exportConfig(currentConfig);
      Toast.success({
        content: t('配置已导出到下载文件夹'),
        duration: 3,
      });
    } catch (error) {
      Toast.error({
        content: t('导出配置失败: ') + error.message,
        duration: 3,
      });
    }
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    try {
      const importedConfig = await importConfig(file);

      Modal.confirm({
        title: t('确认导入配置'),
        content: t('导入的配置将覆盖当前设置，是否继续？'),
        okText: t('确定导入'),
        cancelText: t('取消'),
        onOk: () => {
          onConfigImport(importedConfig);
          Toast.success({
            content: t('配置导入成功'),
            duration: 3,
          });
        },
      });
    } catch (error) {
      Toast.error({
        content: t('导入配置失败: ') + error.message,
        duration: 3,
      });
    } finally {
      // 重置文件输入，允许重复选择同一文件
      event.target.value = '';
    }
  };

  const handleReset = () => {
    Modal.confirm({
      title: t('重置配置'),
      content: t('将清除所有保存的配置并恢复默认设置，此操作不可撤销。是否继续？'),
      okText: t('确定重置'),
      cancelText: t('取消'),
      okButtonProps: {
        type: 'danger',
      },
      onOk: () => {
        clearConfig();
        onConfigReset();
        Toast.success({
          content: t('配置已重置为默认值'),
          duration: 3,
        });
      },
    });
  };

  const getConfigStatus = () => {
    if (hasStoredConfig()) {
      const timestamp = getConfigTimestamp();
      if (timestamp) {
        const date = new Date(timestamp);
        return t('上次保存: ') + date.toLocaleString();
      }
      return t('已有保存的配置');
    }
    return t('暂无保存的配置');
  };

  const dropdownItems = [
    {
      node: 'item',
      name: 'export',
      onClick: handleExport,
      children: (
        <div className="flex items-center gap-2">
          <Download size={14} />
          {t('导出配置')}
        </div>
      ),
    },
    {
      node: 'item',
      name: 'import',
      onClick: handleImportClick,
      children: (
        <div className="flex items-center gap-2">
          <Upload size={14} />
          {t('导入配置')}
        </div>
      ),
    },
    {
      node: 'divider',
    },
    {
      node: 'item',
      name: 'reset',
      onClick: handleReset,
      children: (
        <div className="flex items-center gap-2 text-red-600">
          <RotateCcw size={14} />
          {t('重置配置')}
        </div>
      ),
    },
  ];

  if (styleState.isMobile) {
    // 移动端显示简化的下拉菜单
    return (
      <>
        <Dropdown
          trigger="click"
          position="bottomLeft"
          showTick
          menu={dropdownItems}
        >
          <Button
            icon={<Settings2 size={14} />}
            theme="borderless"
            type="tertiary"
            size="small"
            className="!rounded-lg !text-gray-600 hover:!text-blue-600 hover:!bg-blue-50"
          />
        </Dropdown>

        <input
          ref={fileInputRef}
          type="file"
          accept=".json"
          onChange={handleFileChange}
          style={{ display: 'none' }}
        />
      </>
    );
  }

  // 桌面端显示紧凑的按钮组
  return (
    <div className="space-y-3">
      {/* 配置状态信息，使用较小的字体 */}
      <div className="text-center">
        <Typography.Text className="text-xs text-gray-500">
          {getConfigStatus()}
        </Typography.Text>
      </div>

      {/* 紧凑的按钮布局 */}
      <div className="flex gap-2">
        <Button
          icon={<Download size={12} />}
          size="small"
          theme="solid"
          type="primary"
          onClick={handleExport}
          className="!rounded-lg flex-1 !text-xs !h-7"
        >
          {t('导出')}
        </Button>

        <Button
          icon={<Upload size={12} />}
          size="small"
          theme="outline"
          type="primary"
          onClick={handleImportClick}
          className="!rounded-lg flex-1 !text-xs !h-7"
        >
          {t('导入')}
        </Button>

        <Button
          icon={<RotateCcw size={12} />}
          size="small"
          theme="borderless"
          type="danger"
          onClick={handleReset}
          className="!rounded-lg !text-xs !h-7 !px-2"
          style={{ minWidth: 'auto' }}
        />
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept=".json"
        onChange={handleFileChange}
        style={{ display: 'none' }}
      />
    </div>
  );
};

export default ConfigManager; 