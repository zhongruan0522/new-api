import React from 'react';
import { Card, Divider, Typography } from '@douyinfe/semi-ui';
import PropTypes from 'prop-types';

const { Text } = Typography;

/**
 * CardPro 高级卡片组件
 * 
 * 布局分为5个区域：
 * 1. 统计信息区域 (statsArea)
 * 2. 描述信息区域 (descriptionArea) 
 * 3. 类型切换/标签区域 (tabsArea)
 * 4. 操作按钮区域 (actionsArea)
 * 5. 搜索表单区域 (searchArea)
 * 
 * 支持三种布局类型：
 * - type1: 操作型 (如TokensTable) - 描述信息 + 操作按钮 + 搜索表单
 * - type2: 查询型 (如LogsTable) - 统计信息 + 搜索表单
 * - type3: 复杂型 (如ChannelsTable) - 描述信息 + 类型切换 + 操作按钮 + 搜索表单
 */
const CardPro = ({
  type = 'type1',
  className = '',
  children,
  // 各个区域的内容
  statsArea,
  descriptionArea,
  tabsArea,
  actionsArea,
  searchArea,
  // 卡片属性
  shadows = 'always',
  bordered = false,
  // 自定义样式
  style,
  ...props
}) => {
  // 渲染头部内容
  const renderHeader = () => {
    const hasContent = statsArea || descriptionArea || tabsArea || actionsArea || searchArea;
    if (!hasContent) return null;

    return (
      <div className="flex flex-col w-full">
        {/* 统计信息区域 - 用于type2 */}
        {type === 'type2' && statsArea && (
          <>
            {statsArea}
          </>
        )}

        {/* 描述信息区域 - 用于type1和type3 */}
        {(type === 'type1' || type === 'type3') && descriptionArea && (
          <>
            {descriptionArea}
          </>
        )}

        {/* 第一个分隔线 - 在描述信息或统计信息后面 */}
        {((type === 'type1' || type === 'type3') && descriptionArea) ||
          (type === 'type2' && statsArea) ? (
          <Divider margin="12px" />
        ) : null}

        {/* 类型切换/标签区域 - 主要用于type3 */}
        {type === 'type3' && tabsArea && (
          <>
            {tabsArea}
          </>
        )}

        {/* 操作按钮和搜索表单的容器 */}
        <div className="flex flex-col gap-2">
          {/* 操作按钮区域 - 用于type1和type3 */}
          {(type === 'type1' || type === 'type3') && actionsArea && (
            <div className="w-full">
              {actionsArea}
            </div>
          )}

          {/* 搜索表单区域 - 所有类型都可能有 */}
          {searchArea && (
            <div className="w-full">
              {searchArea}
            </div>
          )}
        </div>
      </div>
    );
  };

  const headerContent = renderHeader();

  return (
    <Card
      className={`table-scroll-card !rounded-2xl ${className}`}
      title={headerContent}
      shadows={shadows}
      bordered={bordered}
      style={style}
      {...props}
    >
      {children}
    </Card>
  );
};

CardPro.propTypes = {
  // 布局类型
  type: PropTypes.oneOf(['type1', 'type2', 'type3']),
  // 样式相关
  className: PropTypes.string,
  style: PropTypes.object,
  shadows: PropTypes.oneOfType([PropTypes.string, PropTypes.bool]),
  bordered: PropTypes.bool,
  // 内容区域
  statsArea: PropTypes.node,
  descriptionArea: PropTypes.node,
  tabsArea: PropTypes.node,
  actionsArea: PropTypes.node,
  searchArea: PropTypes.node,
  // 表格内容
  children: PropTypes.node,
};

export default CardPro; 