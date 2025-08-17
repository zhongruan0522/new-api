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
import { Button } from '@douyinfe/semi-ui';
import { IconSun, IconMoon } from '@douyinfe/semi-icons';

const ThemeToggle = ({ theme, onThemeToggle, t }) => {
  return (
    <Button
      icon={theme === 'dark' ? <IconSun size="large" className="text-yellow-500" /> : <IconMoon size="large" className="text-gray-300" />}
      aria-label={t('切换主题')}
      onClick={onThemeToggle}
      theme="borderless"
      type="tertiary"
      className="!p-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2"
    />
  );
};

export default ThemeToggle;
