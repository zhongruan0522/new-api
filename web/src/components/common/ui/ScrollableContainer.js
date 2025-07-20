/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useRef, useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';

/**
 * ScrollableContainer 可滚动容器组件
 * 
 * 提供自动检测滚动状态和显示渐变指示器的功能
 * 当内容超出容器高度且未滚动到底部时，会显示底部渐变指示器
 */
const ScrollableContainer = ({
  children,
  maxHeight = '24rem',
  className = '',
  contentClassName = 'p-2',
  fadeIndicatorClassName = '',
  checkInterval = 100,
  scrollThreshold = 5,
  onScroll,
  onScrollStateChange,
  ...props
}) => {
  const scrollRef = useRef(null);
  const [showScrollHint, setShowScrollHint] = useState(false);

  // 检查是否可滚动且未滚动到底部
  const checkScrollable = useCallback(() => {
    if (scrollRef.current) {
      const element = scrollRef.current;
      const isScrollable = element.scrollHeight > element.clientHeight;
      const isAtBottom = element.scrollTop + element.clientHeight >= element.scrollHeight - scrollThreshold;
      const shouldShowHint = isScrollable && !isAtBottom;

      setShowScrollHint(shouldShowHint);

      // 通知父组件滚动状态变化
      if (onScrollStateChange) {
        onScrollStateChange({
          isScrollable,
          isAtBottom,
          showScrollHint: shouldShowHint,
          scrollTop: element.scrollTop,
          scrollHeight: element.scrollHeight,
          clientHeight: element.clientHeight
        });
      }
    }
  }, [scrollThreshold, onScrollStateChange]);

  // 处理滚动事件
  const handleScroll = useCallback((e) => {
    checkScrollable();
    if (onScroll) {
      onScroll(e);
    }
  }, [checkScrollable, onScroll]);

  // 初始检查和内容变化时检查
  useEffect(() => {
    const timer = setTimeout(() => {
      checkScrollable();
    }, checkInterval);
    return () => clearTimeout(timer);
  }, [children, checkScrollable, checkInterval]);

  // 暴露检查方法给父组件
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.checkScrollable = checkScrollable;
    }
  }, [checkScrollable]);

  return (
    <div
      className={`card-content-container ${className}`}
      {...props}
    >
      <div
        ref={scrollRef}
        className={`overflow-y-auto card-content-scroll ${contentClassName}`}
        style={{ maxHeight }}
        onScroll={handleScroll}
      >
        {children}
      </div>
      <div
        className={`card-content-fade-indicator ${fadeIndicatorClassName}`}
        style={{ opacity: showScrollHint ? 1 : 0 }}
      />
    </div>
  );
};

ScrollableContainer.propTypes = {
  // 子组件内容
  children: PropTypes.node.isRequired,

  // 样式相关
  maxHeight: PropTypes.string,
  className: PropTypes.string,
  contentClassName: PropTypes.string,
  fadeIndicatorClassName: PropTypes.string,

  // 行为配置
  checkInterval: PropTypes.number,
  scrollThreshold: PropTypes.number,

  // 事件回调
  onScroll: PropTypes.func,
  onScrollStateChange: PropTypes.func,
};

export default ScrollableContainer; 