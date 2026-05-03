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
import { Typography } from '@douyinfe/semi-ui';
import {
  Moonshot,
  OpenAI,
  XAI,
  Zhipu,
  Volcengine,
  Cohere,
  Claude,
  Gemini,
  Minimax,
  Spark,
  Qingyan,
  DeepSeek,
  Qwen,
  Grok,
  AzureAI,
  Hunyuan,
} from '@lobehub/icons';

const { Text } = Typography;

/**
 * 供应商图标展示区域 — 展示支持的大模型供应商
 * 参考参考项目 default 主题的 scrolling-icons 模式
 */
const ProviderIcons = ({ t }) => {
  const providers = [
    { Icon: Moonshot, color: false },
    { Icon: OpenAI, color: false },
    { Icon: XAI, color: false },
    { Icon: Zhipu, color: true },
    { Icon: Volcengine, color: true },
    { Icon: Cohere, color: true },
    { Icon: Claude, color: true },
    { Icon: Gemini, color: true },
    { Icon: Minimax, color: true },
    { Icon: Spark, color: true },
    { Icon: Qingyan, color: true },
    { Icon: DeepSeek, color: true },
    { Icon: Qwen, color: true },
    { Icon: Grok, color: false },
    { Icon: AzureAI, color: true },
    { Icon: Hunyuan, color: true },
  ];

  return (
    <div className='mt-8 md:mt-10 w-full'>
      <div className='flex items-center mb-4 md:mb-6 justify-center'>
        <Text
          type='tertiary'
          className='text-lg md:text-xl lg:text-2xl font-light'
        >
          {t('支持众多的大模型供应商')}
        </Text>
      </div>
      <div className='flex flex-wrap items-center justify-center gap-3 sm:gap-4 md:gap-6 lg:gap-8 max-w-5xl mx-auto px-4'>
        {providers.map(({ Icon, color }, idx) => (
          <div
            key={idx}
            className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'
          >
            {color ? <Icon.Color size={40} /> : <Icon size={40} />}
          </div>
        ))}
        <div className='w-8 h-8 sm:w-10 sm:h-10 md:w-12 md:h-12 flex items-center justify-center'>
          <Text className='!text-lg sm:!text-xl md:!text-2xl lg:!text-3xl font-bold'>
            30+
          </Text>
        </div>
      </div>
    </div>
  );
};

export default ProviderIcons;
