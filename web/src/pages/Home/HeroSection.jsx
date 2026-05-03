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

import React from 'react';
import { Button, Input, ScrollList, ScrollItem } from '@douyinfe/semi-ui';
import { IconPlay, IconFile, IconCopy } from '@douyinfe/semi-icons';
import { Link } from 'react-router-dom';

/**
 * Hero 区域 — 首页顶部主标题、BASE URL 输入框、操作按钮
 */
const HeroSection = ({
  t,
  isMobile,
  serverAddress,
  endpointItems,
  endpointIndex,
  setEndpointIndex,
  handleCopyBaseURL,
  docsLink,
}) => {
  return (
    <div className='flex flex-col items-center justify-center mb-6 md:mb-8'>
      {/* 主标题 */}
      <h1 className='text-4xl md:text-5xl lg:text-6xl xl:text-7xl font-bold text-semi-color-text-0 leading-tight tracking-wide md:tracking-wider'>
        {t('统一的')}
        <br />
        <span className='shine-text'>{t('大模型接口网关')}</span>
      </h1>

      {/* 副标题 */}
      <p className='text-base md:text-lg lg:text-xl text-semi-color-text-1 mt-4 md:mt-6 max-w-xl'>
        {t('更好的价格，更好的稳定性，只需要将模型基址替换为：')}
      </p>

      {/* BASE URL 与端点选择 */}
      <div className='flex flex-col md:flex-row items-center justify-center gap-4 w-full mt-4 md:mt-6 max-w-md'>
        <Input
          readonly
          value={serverAddress}
          className='flex-1 !rounded-full'
          size={isMobile ? 'default' : 'large'}
          suffix={
            <div className='flex items-center gap-2'>
              <ScrollList
                bodyHeight={32}
                style={{ border: 'unset', boxShadow: 'unset' }}
              >
                <ScrollItem
                  mode='wheel'
                  cycled={true}
                  list={endpointItems}
                  selectedIndex={endpointIndex}
                  onSelect={({ index }) => setEndpointIndex(index)}
                />
              </ScrollList>
              <Button
                type='primary'
                onClick={handleCopyBaseURL}
                icon={<IconCopy />}
                className='!rounded-full'
              />
            </div>
          }
        />
      </div>

      {/* 操作按钮 */}
      <div className='flex flex-row gap-4 justify-center items-center mt-6'>
        <Link to='/console'>
          <Button
            theme='solid'
            type='primary'
            size={isMobile ? 'default' : 'large'}
            className='!rounded-3xl px-8 py-2'
            icon={<IconPlay />}
          >
            {t('获取密钥')}
          </Button>
        </Link>
        {docsLink && (
          <Button
            size={isMobile ? 'default' : 'large'}
            className='flex items-center !rounded-3xl px-6 py-2'
            icon={<IconFile />}
            onClick={() => window.open(docsLink, '_blank')}
          >
            {t('文档')}
          </Button>
        )}
      </div>
    </div>
  );
};

export default HeroSection;
