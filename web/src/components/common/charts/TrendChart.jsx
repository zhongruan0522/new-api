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
import { VChart } from '@visactor/react-vchart';

const TrendChart = ({
  data,
  color,
  width = 100,
  height = 40,
  config = { mode: 'desktop-browser' }
}) => {
  const getTrendSpec = (data, color) => ({
    type: 'line',
    data: [{ id: 'trend', values: data.map((val, idx) => ({ x: idx, y: val })) }],
    xField: 'x',
    yField: 'y',
    height: height,
    width: width,
    axes: [
      {
        orient: 'bottom',
        visible: false
      },
      {
        orient: 'left',
        visible: false
      }
    ],
    padding: 0,
    autoFit: false,
    legends: { visible: false },
    tooltip: { visible: false },
    crosshair: { visible: false },
    line: {
      style: {
        stroke: color,
        lineWidth: 2
      }
    },
    point: {
      visible: false
    },
    background: {
      fill: 'transparent'
    }
  });

  return (
    <VChart
      spec={getTrendSpec(data, color)}
      option={config}
    />
  );
};

export default TrendChart; 