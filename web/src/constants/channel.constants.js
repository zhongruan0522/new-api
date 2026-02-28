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

export const CHANNEL_OPTIONS = [
  { value: 1, color: 'green', label: 'OpenAI' },
  { value: 4, color: 'grey', label: 'Ollama' },
  {
    value: 14,
    color: 'indigo',
    label: 'Anthropic Claude',
  },
  {
    value: 33,
    color: 'indigo',
    label: 'AWS Claude',
  },
  { value: 41, color: 'blue', label: 'Vertex AI' },
  {
    value: 3,
    color: 'teal',
    label: 'Azure OpenAI',
  },
  {
    value: 34,
    color: 'purple',
    label: 'Cohere',
  },
  { value: 39, color: 'grey', label: 'Cloudflare' },
  { value: 43, color: 'blue', label: 'DeepSeek' },
  {
    value: 17,
    color: 'orange',
    label: '阿里通义千问',
  },
  {
    value: 18,
    color: 'blue',
    label: '讯飞星火认知',
  },
  {
    value: 26,
    color: 'purple',
    label: '智谱 GLM-4V',
  },
  {
    value: 24,
    color: 'orange',
    label: 'Google Gemini',
  },
  {
    value: 11,
    color: 'orange',
    label: 'Google PaLM2',
  },
  { value: 25, color: 'green', label: 'Moonshot' },
  { value: 20, color: 'green', label: 'OpenRouter' },
  { value: 23, color: 'teal', label: '腾讯混元' },
  { value: 31, color: 'green', label: '零一万物' },
  { value: 35, color: 'green', label: 'MiniMax' },
  { value: 37, color: 'teal', label: 'Dify' },
  { value: 38, color: 'blue', label: 'Jina' },
  { value: 40, color: 'purple', label: 'SiliconCloud' },
  { value: 42, color: 'blue', label: 'Mistral AI' },
  { value: 8, color: 'pink', label: '自定义渠道' },
  {
    value: 45,
    color: 'blue',
    label: '字节火山方舟、豆包通用',
  },
  {
    value: 48,
    color: 'blue',
    label: 'xAI',
  },
];

export const MODEL_TABLE_PAGE_SIZE = 10;
