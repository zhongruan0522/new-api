import React, { useState } from 'react';
import { Button, Tooltip, Toast } from '@douyinfe/semi-ui';
import { Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { copy } from '../../helpers/utils';

// VS Code 深色主题样式
const codeThemeStyles = {
  container: {
    backgroundColor: '#1e1e1e',
    color: '#d4d4d4',
    fontFamily: 'Consolas, "Courier New", Monaco, "SF Mono", monospace',
    fontSize: '13px',
    lineHeight: '1.4',
    borderRadius: '8px',
    border: '1px solid #3c3c3c',
    position: 'relative',
    overflow: 'hidden',
    boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)',
  },
  content: {
    height: '100%',
    overflowY: 'auto',
    overflowX: 'auto',
    padding: '16px',
    margin: 0,
    whiteSpace: 'pre',
    wordBreak: 'normal',
    background: '#1e1e1e',
  },
  copyButton: {
    position: 'absolute',
    top: '12px',
    right: '12px',
    zIndex: 10,
    backgroundColor: 'rgba(45, 45, 45, 0.9)',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    color: '#d4d4d4',
    borderRadius: '6px',
    transition: 'all 0.2s ease',
  },
  copyButtonHover: {
    backgroundColor: 'rgba(60, 60, 60, 0.95)',
    borderColor: 'rgba(255, 255, 255, 0.2)',
    transform: 'scale(1.05)',
  },
  noContent: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100%',
    color: '#666',
    fontSize: '14px',
    fontStyle: 'italic',
    backgroundColor: 'var(--semi-color-fill-0)',
    borderRadius: '8px',
  }
};

// 自定义 JSON 高亮器（使用 VS Code 深色主题配色）
const highlightJson = (str) => {
  return str.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+-]?\d+)?)/g,
    (match) => {
      let color = '#b5cea8'; // 数字颜色 (绿色)
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          color = '#9cdcfe'; // 键名颜色 (蓝色)
        } else {
          color = '#ce9178'; // 字符串值颜色 (橙色)
        }
      } else if (/true|false/.test(match)) {
        color = '#569cd6'; // 布尔值颜色 (蓝色)
      } else if (/null/.test(match)) {
        color = '#569cd6'; // null 值颜色 (蓝色)
      }
      return `<span style="color: ${color}">${match}</span>`;
    }
  );
};

const CodeViewer = ({ content, title, language = 'json' }) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const [isHoveringCopy, setIsHoveringCopy] = useState(false);

  const handleCopy = async () => {
    try {
      let textToCopy = content;

      // 如果是对象，转换为格式化的 JSON 字符串
      if (typeof content === 'object' && content !== null) {
        textToCopy = JSON.stringify(content, null, 2);
      }

      const success = await copy(textToCopy);
      if (success) {
        setCopied(true);
        Toast.success(t('已复制到剪贴板'));
        setTimeout(() => setCopied(false), 2000);
      } else {
        Toast.error(t('复制失败'));
      }
    } catch (err) {
      Toast.error(t('复制失败'));
      console.error('Copy failed:', err);
    }
  };

  // 格式化内容
  const getFormattedContent = () => {
    if (!content) return '';

    if (typeof content === 'object') {
      try {
        return JSON.stringify(content, null, 2);
      } catch (e) {
        return String(content);
      }
    } else if (typeof content === 'string') {
      // 尝试解析并重新格式化 JSON
      try {
        const parsed = JSON.parse(content);
        return JSON.stringify(parsed, null, 2);
      } catch (e) {
        return content;
      }
    }

    return String(content);
  };

  // 获取高亮的 HTML
  const getHighlightedContent = () => {
    const formattedContent = getFormattedContent();

    // 尝试检测是否为 JSON 格式
    const isJsonLike = () => {
      if (language === 'json') return true;

      // 自动检测：如果内容看起来像 JSON，就用 JSON 高亮
      const trimmed = formattedContent.trim();
      const looksLikeJson = (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
        (trimmed.startsWith('[') && trimmed.endsWith(']'));

      // 调试日志
      if (process.env.NODE_ENV === 'development') {
        console.log('CodeViewer Debug:', {
          language,
          contentType: typeof content,
          trimmedStart: trimmed.substring(0, 10),
          looksLikeJson,
          willHighlight: looksLikeJson
        });
      }

      return looksLikeJson;
    };

    if (isJsonLike()) {
      return highlightJson(formattedContent);
    }

    // 对于非 JSON 内容，使用简单的文本高亮
    return formattedContent;
  };

  if (!content) {
    return (
      <div style={codeThemeStyles.noContent}>
        <span>
          {title === 'preview' ? t('正在构造请求体预览...') :
            title === 'request' ? t('暂无请求数据') :
              t('暂无响应数据')}
        </span>
      </div>
    );
  }

  return (
    <div style={codeThemeStyles.container} className="h-full">
      {/* 复制按钮 */}
      <div
        style={{
          ...codeThemeStyles.copyButton,
          ...(isHoveringCopy ? codeThemeStyles.copyButtonHover : {})
        }}
        onMouseEnter={() => setIsHoveringCopy(true)}
        onMouseLeave={() => setIsHoveringCopy(false)}
      >
        <Tooltip content={copied ? t('已复制') : t('复制代码')}>
          <Button
            icon={<Copy size={14} />}
            onClick={handleCopy}
            size="small"
            theme="borderless"
            style={{
              backgroundColor: 'transparent',
              border: 'none',
              color: copied ? '#4ade80' : '#d4d4d4',
              padding: '6px',
            }}
          />
        </Tooltip>
      </div>

      {/* 代码内容 */}
      <div
        style={codeThemeStyles.content}
        className="model-settings-scroll"
        dangerouslySetInnerHTML={{ __html: getHighlightedContent() }}
      />
    </div>
  );
};

export default CodeViewer; 