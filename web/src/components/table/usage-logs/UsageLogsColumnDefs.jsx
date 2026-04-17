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
import {
  Avatar,
  Space,
  Tag,
  Tooltip,
  Popover,
  Typography,
  Button
} from '@douyinfe/semi-ui';
import {
  timestamp2string,
  renderGroup,
  renderQuota,
  stringToColor,
  getLogOther,
  renderModelTag,
  renderClaudeLogContent,
  renderLogContent,
  renderAudioModelPrice,
  renderClaudeModelPrice,
  renderModelPrice,
} from '../../../helpers';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import { Route, Sparkles } from 'lucide-react';

const colors = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];

function formatRatio(ratio) {
  if (ratio === undefined || ratio === null) {
    return '-';
  }
  if (typeof ratio === 'number') {
    return ratio.toFixed(4);
  }
  return String(ratio);
}

function buildChannelAffinityTooltip(affinity, t) {
  if (!affinity) {
    return null;
  }

  const keySource = affinity.key_source || '-';
  const keyPath = affinity.key_path || affinity.key_key || '-';
  const keyHint = affinity.key_hint || '';
  const keyFp = affinity.key_fp ? `#${affinity.key_fp}` : '';
  const keyText = `${keySource}:${keyPath}${keyFp}`;

  const lines = [
    t('渠道亲和性'),
    `${t('规则')}：${affinity.rule_name || '-'}`,
    `${t('分组')}：${affinity.selected_group || '-'}`,
    `${t('Key')}：${keyText}`,
    ...(keyHint ? [`${t('Key 摘要')}：${keyHint}`] : []),
  ];

  return (
    <div style={{ lineHeight: 1.6, display: 'flex', flexDirection: 'column' }}>
      {lines.map((line, i) => (
        <div key={i}>{line}</div>
      ))}
    </div>
  );
}

// Render functions
function renderType(type, t) {
  switch (type) {
    case 1:
      return (
        <Tag color='cyan' shape='circle'>
          {t('充值')}
        </Tag>
      );
    case 2:
      return (
        <Tag color='lime' shape='circle'>
          {t('消费')}
        </Tag>
      );
    case 3:
      return (
        <Tag color='orange' shape='circle'>
          {t('管理')}
        </Tag>
      );
    case 4:
      return (
        <Tag color='purple' shape='circle'>
          {t('系统')}
        </Tag>
      );
    case 5:
      return (
        <Tag color='red' shape='circle'>
          {t('错误')}
        </Tag>
      );
    default:
      return (
        <Tag color='grey' shape='circle'>
          {t('未知')}
        </Tag>
      );
  }
}

function renderIsStream(bool, t) {
  if (bool) {
    return (
      <Tag color='blue' shape='circle'>
        {t('流')}
      </Tag>
    );
  } else {
    return (
      <Tag color='purple' shape='circle'>
        {t('非流')}
      </Tag>
    );
  }
}

function renderUseTime(type, t) {
  // use_time 现在存储的是毫秒，前端转换为秒显示
  const ms = parseInt(type);
  const time = (ms / 1000).toFixed(1);
  if (time < 101) {
    return (
      <Tag color='green' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  } else if (time < 300) {
    return (
      <Tag color='orange' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  } else {
    return (
      <Tag color='red' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  }
}

function renderFirstUseTime(type, t) {
  // frt 已经是毫秒，转换为秒显示，保留2位小数
  let time = parseFloat(type) / 1000.0;
  time = time.toFixed(2);
  if (time < 3) {
    return (
      <Tag color='green' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  } else if (time < 10) {
    return (
      <Tag color='orange' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  } else {
    return (
      <Tag color='red' shape='circle'>
        {' '}
        {time} s{' '}
      </Tag>
    );
  }
}

function renderSpeed(speed, t) {
  if (speed === undefined || speed === null || speed <= 0) {
    return null;
  }
  const speedStr = speed.toFixed(1);
  let color = 'red';
  if (speed >= 100) {
    color = 'green';
  } else if (speed >= 70) {
    color = 'blue';
  }
  return (
    <Tag color={color} shape='circle'>
      {speedStr} t/s
    </Tag>
  );
}

function renderModelName(record, copyText, t) {
  let other = getLogOther(record.other);
  let modelMapped =
    other?.is_model_mapped &&
    other?.upstream_model_name &&
    other?.upstream_model_name !== '';
  if (!modelMapped) {
    return renderModelTag(record.model_name, {
      onClick: (event) => {
        copyText(event, record.model_name).then((r) => {});
      },
    });
  } else {
    return (
      <>
        <Space vertical align={'start'}>
          <Popover
            content={
              <div style={{ padding: 10 }}>
                <Space vertical align={'start'}>
                  <div className='flex items-center'>
                    <Typography.Text strong style={{ marginRight: 8 }}>
                      {t('请求并计费模型')}:
                    </Typography.Text>
                    {renderModelTag(record.model_name, {
                      onClick: (event) => {
                        copyText(event, record.model_name).then((r) => {});
                      },
                    })}
                  </div>
                  <div className='flex items-center'>
                    <Typography.Text strong style={{ marginRight: 8 }}>
                      {t('实际模型')}:
                    </Typography.Text>
                    {renderModelTag(other.upstream_model_name, {
                      onClick: (event) => {
                        copyText(event, other.upstream_model_name).then(
                          (r) => {},
                        );
                      },
                    })}
                  </div>
                </Space>
              </div>
            }
          >
            {renderModelTag(record.model_name, {
              onClick: (event) => {
                copyText(event, record.model_name).then((r) => {});
              },
              suffixIcon: (
                <Route
                  style={{ width: '0.9em', height: '0.9em', opacity: 0.75 }}
                />
              ),
            })}
          </Popover>
        </Space>
      </>
    );
  }
}

export const getLogsColumns = ({
  t,
  COLUMN_KEYS,
  copyText,
  showUserInfoFunc,
  openChannelAffinityUsageCacheModal,
  isAdminUser,
}) => {
  return [
    {
      key: COLUMN_KEYS.TIME,
      title: t('时间'),
      dataIndex: 'timestamp2string',
    },
    {
      key: COLUMN_KEYS.CHANNEL,
      title: t('渠道'),
      dataIndex: 'channel',
      render: (text, record, index) => {
        let isMultiKey = false;
        let multiKeyIndex = -1;
        let content = t('渠道') + `：${record.channel}`;
        let affinity = null;
        let showMarker = false;
        let other = getLogOther(record.other);
        if (other?.admin_info) {
          let adminInfo = other.admin_info;
          if (adminInfo?.is_multi_key) {
            isMultiKey = true;
            multiKeyIndex = adminInfo.multi_key_index;
          }
          if (
            Array.isArray(adminInfo.use_channel) &&
            adminInfo.use_channel.length > 0
          ) {
            content = t('渠道') + `：${adminInfo.use_channel.join('->')}`;
          }
          if (adminInfo.channel_affinity) {
            affinity = adminInfo.channel_affinity;
            showMarker = true;
          }
        }

        return isAdminUser &&
          (record.type === 0 || record.type === 2 || record.type === 5) ? (
          <Space>
            <span style={{ position: 'relative', display: 'inline-block' }}>
              <Tooltip content={record.channel_name || t('未知渠道')}>
                <span>
                  <Tag
                    color={colors[parseInt(text) % colors.length]}
                    shape='circle'
                  >
                    {text}
                  </Tag>
                </span>
              </Tooltip>
              {showMarker && (
                <Tooltip
                  content={
                    <div style={{ lineHeight: 1.6 }}>
                      <div>{content}</div>
                      {affinity ? (
                        <div style={{ marginTop: 6 }}>
                          {buildChannelAffinityTooltip(affinity, t)}
                        </div>
                      ) : null}
                    </div>
                  }
                >
                  <span
                    style={{
                      position: 'absolute',
                      right: -4,
                      top: -4,
                      lineHeight: 1,
                      fontWeight: 600,
                      color: '#f59e0b',
                      cursor: 'pointer',
                      userSelect: 'none',
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      openChannelAffinityUsageCacheModal?.(affinity);
                    }}
                  >
                    <Sparkles
                      size={14}
                      strokeWidth={2}
                      color='currentColor'
                      fill='currentColor'
                    />
                  </span>
                </Tooltip>
              )}
            </span>
            {isMultiKey && (
              <Tag color='white' shape='circle'>
                {multiKeyIndex}
              </Tag>
            )}
          </Space>
        ) : null;
      },
    },
    {
      key: COLUMN_KEYS.USERNAME,
      title: t('用户'),
      dataIndex: 'username',
      render: (text, record, index) => {
        return isAdminUser ? (
          <div>
            <Avatar
              size='extra-small'
              color={stringToColor(text)}
              style={{ marginRight: 4 }}
              onClick={(event) => {
                event.stopPropagation();
                showUserInfoFunc(record.user_id);
              }}
            >
              {typeof text === 'string' && text.slice(0, 1)}
            </Avatar>
            {text}
          </div>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.TOKEN,
      title: t('令牌'),
      dataIndex: 'token_name',
      render: (text, record, index) => {
        return record.type === 0 || record.type === 2 || record.type === 5 ? (
          <div>
            <Tag
              color='grey'
              shape='circle'
              onClick={(event) => {
                copyText(event, text);
              }}
            >
              {' '}
              {t(text)}{' '}
            </Tag>
          </div>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.GROUP,
      title: t('分组'),
      dataIndex: 'group',
      render: (text, record, index) => {
        if (record.type === 0 || record.type === 2 || record.type === 5) {
          if (record.group) {
            return <>{renderGroup(record.group)}</>;
          } else {
            let other = null;
            try {
              other = JSON.parse(record.other);
            } catch (e) {
              console.error(
                `Failed to parse record.other: "${record.other}".`,
                e,
              );
            }
            if (other === null) {
              return <></>;
            }
            if (other.group !== undefined) {
              return <>{renderGroup(other.group)}</>;
            } else {
              return <></>;
            }
          }
        } else {
          return <></>;
        }
      },
    },
    {
      key: COLUMN_KEYS.TYPE,
      title: t('类型'),
      dataIndex: 'type',
      render: (text, record, index) => {
        return <>{renderType(text, t)}</>;
      },
    },
    {
      key: COLUMN_KEYS.MODEL,
      title: t('模型'),
      dataIndex: 'model_name',
      render: (text, record, index) => {
        return record.type === 0 || record.type === 2 || record.type === 5 ? (
          <>{renderModelName(record, copyText, t)}</>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.USE_TIME,
      title: t('用时/首字'),
      dataIndex: 'use_time',
      render: (text, record, index) => {
        if (!(record.type === 2 || record.type === 5)) {
          return <></>;
        }
        let other = getLogOther(record.other);
        let speedTag = renderSpeed(other?.speed, t);
        if (record.is_stream) {
          return (
            <>
              <Space>
                {renderUseTime(text, t)}
                {renderFirstUseTime(other?.frt, t)}
                {speedTag}
                {renderIsStream(record.is_stream, t)}
              </Space>
            </>
          );
        } else {
          return (
            <>
              <Space>
                {renderUseTime(text, t)}
                {renderIsStream(record.is_stream, t)}
              </Space>
            </>
          );
        }
      },
    },
    {
      key: COLUMN_KEYS.PROMPT,
      title: t('输入'),
      dataIndex: 'prompt_tokens',
      render: (text, record, index) => {
        return record.type === 0 || record.type === 2 || record.type === 5 ? (
          <>{<span> {text} </span>}</>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.COMPLETION,
      title: t('输出'),
      dataIndex: 'completion_tokens',
      render: (text, record, index) => {
        return parseInt(text) > 0 &&
          (record.type === 0 || record.type === 2 || record.type === 5) ? (
          <>{<span> {text} </span>}</>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.COST,
      title: t('花费'),
      dataIndex: 'quota',
      render: (text, record, index) => {
        if (!(record.type === 0 || record.type === 2 || record.type === 5)) {
          return <></>;
        }
        return <>{renderQuota(text, 6)}</>;
      },
    },
    {
      key: COLUMN_KEYS.IP,
      title: (
        <div className='flex items-center gap-1'>
          {t('IP')}
          <Tooltip
            content={t(
              '仅“消费”和“错误”日志会记录客户端IP（强制开启）',
            )}
          >
            <IconHelpCircle className='text-gray-400 cursor-help' />
          </Tooltip>
        </div>
      ),
      dataIndex: 'ip',
      render: (text, record, index) => {
        return (record.type === 2 || record.type === 5) && text ? (
          <Tooltip content={text}>
            <span>
              <Tag
                color='orange'
                shape='circle'
                onClick={(event) => {
                  copyText(event, text);
                }}
              >
                {text}
              </Tag>
            </span>
          </Tooltip>
        ) : (
          <></>
        );
      },
    },
    {
      key: COLUMN_KEYS.RETRY,
      title: t('重试'),
      dataIndex: 'retry',
      render: (text, record, index) => {
        if (!(record.type === 2 || record.type === 5)) {
          return <></>;
        }
        let content = t('渠道') + `：${record.channel}`;
        if (record.other !== '') {
          let other = JSON.parse(record.other);
          if (other === null) {
            return <></>;
          }
          if (other.admin_info !== undefined) {
            if (
                other.admin_info.use_channel !== null &&
                other.admin_info.use_channel !== undefined &&
                other.admin_info.use_channel !== ''
            ) {
              let useChannel = other.admin_info.use_channel;
              let useChannelStr = useChannel.join('->');
              content = t('渠道') + `：${useChannelStr}`;
            }
          }
        }
        return isAdminUser ? <div>{content}</div> : <></>;
      },
    },
    {
      key: COLUMN_KEYS.DETAILS,
      title: t('详情'),
      dataIndex: 'content',
      fixed: 'right',
      render: (text, record, index) => {
        let other = getLogOther(record.other);
        if (other == null || record.type !== 2) {
          // For error logs (type 5), compress to single line and simplify status_code -> code
          let displayText = text || '';
          if (record.type === 5 && displayText) {
            displayText = displayText.replace(/status_code=/g, 'code=');
          }
          return (
            <Typography.Paragraph
              ellipsis={{
                rows: record.type === 5 ? 1 : 2,
                showTooltip: {
                  type: 'popover',
                  opts: { style: { width: 240 } },
                },
              }}
              style={{ maxWidth: 240 }}
            >
              {displayText}
            </Typography.Paragraph>
          );
        }

        if (
          other?.violation_fee === true ||
          Boolean(other?.violation_fee_code) ||
          Boolean(other?.violation_fee_marker)
        ) {
          const feeQuota = other?.fee_quota ?? record?.quota;
          const ratioText = formatRatio(other?.group_ratio);
          const summary = [
            t('违规扣费'),
            `${t('分组倍率')}：${ratioText}`,
            `${t('扣费')}：${renderQuota(feeQuota, 6)}`,
            text ? `${t('详情')}：${text}` : null,
          ]
            .filter(Boolean)
            .join('\n');
          return (
            <Typography.Paragraph
              ellipsis={{
                rows: 2,
                showTooltip: {
                  type: 'popover',
                  opts: { style: { width: 240 } },
                },
              }}
              style={{ maxWidth: 240, whiteSpace: 'pre-line' }}
            >
              {summary}
            </Typography.Paragraph>
          );
        }

        // Determine billing type and group ratio
        const groupRatio = other.group_ratio;
        const userGroupRatio = other?.user_group_ratio;
        const effectiveGroupRatio =
          Number.isFinite(userGroupRatio) && userGroupRatio !== -1
            ? userGroupRatio
            : groupRatio;
        const groupRatioText = formatRatio(effectiveGroupRatio) + 'x';
        const hasCache = (other.cache_tokens || 0) > 0;

        let content;
        if (other.model_price != null && other.model_price !== -1) {
          // 按次计费
          content = `${t('按次计费')}，${t('分组')}${groupRatioText}`;
        } else if (hasCache) {
          // 按量计费，支持缓存
          content = `${t('按量计费')}，${t('缓存')}√，${t('分组')}${groupRatioText}`;
        } else {
          // 按量计费，不支持缓存
          content = `${t('按量计费')}，${t('缓存')}×，${t('分组')}${groupRatioText}`;
        }

        return (
            <Typography.Paragraph
                ellipsis={{
                  rows: 3,
                }}
                style={{ maxWidth: 240, whiteSpace: 'pre-line' }}
            >
              {content}
            </Typography.Paragraph>
        );
      },
    },
  ];
};
