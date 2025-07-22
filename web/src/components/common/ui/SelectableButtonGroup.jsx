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

import React, { useState, useRef } from 'react';
import { Divider, Button, Tag, Row, Col, Collapsible } from '@douyinfe/semi-ui';
import { IconChevronDown, IconChevronUp } from '@douyinfe/semi-icons';

/**
 * 通用可选择按钮组组件
 *
 * @param {string} title 标题
 * @param {Array<{value:any,label:string,icon?:React.ReactNode,tagCount?:number}>} items 按钮项
 * @param {*} activeValue 当前激活的值
 * @param {(value:any)=>void} onChange 选择改变回调
 * @param {function} t i18n
 * @param {object} style 额外样式
 * @param {boolean} collapsible 是否支持折叠，默认true
 * @param {number} collapseHeight 折叠时的高度，默认200
 */
const SelectableButtonGroup = ({
  title,
  items = [],
  activeValue,
  onChange,
  t = (v) => v,
  style = {},
  collapsible = true,
  collapseHeight = 200
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const perRow = 3;
  const maxVisibleRows = Math.max(1, Math.floor(collapseHeight / 32)); // Approx row height 32
  const needCollapse = collapsible && items.length > perRow * maxVisibleRows;

  const contentRef = useRef(null);

  const maskStyle = isOpen
    ? {}
    : {
      WebkitMaskImage:
        'linear-gradient(to bottom, black 0%, rgba(0, 0, 0, 1) 60%, rgba(0, 0, 0, 0.2) 80%, transparent 100%)',
    };

  const toggle = () => {
    setIsOpen(!isOpen);
  };

  const linkStyle = {
    position: 'absolute',
    left: 0,
    right: 0,
    textAlign: 'center',
    bottom: -10,
    fontWeight: 400,
    cursor: 'pointer',
    fontSize: '12px',
    color: 'var(--semi-color-text-2)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 4,
  };

  const contentElement = (
    <Row gutter={[8, 8]} style={{ lineHeight: '32px', ...style }} ref={contentRef}>
      {items.map((item) => {
        const isActive = activeValue === item.value;
        return (
          <Col span={8} key={item.value}>
            <Button
              onClick={() => onChange(item.value)}
              theme={isActive ? 'solid' : 'outline'}
              type={isActive ? 'primary' : 'tertiary'}
              icon={item.icon}
              style={{ width: '100%' }}
            >
              <span style={{ marginRight: item.tagCount !== undefined ? 4 : 0 }}>{item.label}</span>
              {item.tagCount !== undefined && (
                <Tag
                  color='white'
                  shape="circle"
                  size="small"
                >
                  {item.tagCount}
                </Tag>
              )}
            </Button>
          </Col>
        );
      })}
    </Row>
  );

  return (
    <div className="mb-8">
      {title && (
        <Divider margin="12px" align="left">
          {title}
        </Divider>
      )}
      {needCollapse ? (
        <div style={{ position: 'relative' }}>
          <Collapsible isOpen={isOpen} collapseHeight={collapseHeight} style={{ ...maskStyle }}>
            {contentElement}
          </Collapsible>
          {isOpen ? null : (
            <div onClick={toggle} style={{ ...linkStyle }}>
              <IconChevronDown size="small" />
              <span>{t('展开更多')}</span>
            </div>
          )}
          {isOpen && (
            <div onClick={toggle} style={{
              ...linkStyle,
              position: 'static',
              marginTop: 8,
              bottom: 'auto'
            }}>
              <IconChevronUp size="small" />
              <span>{t('收起')}</span>
            </div>
          )}
        </div>
      ) : (
        contentElement
      )}
    </div>
  );
};

export default SelectableButtonGroup; 