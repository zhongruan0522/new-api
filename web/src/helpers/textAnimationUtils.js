import { visit } from 'unist-util-visit';

/**
 * rehype 插件：将段落等文本节点拆分为逐词 <span>，并添加淡入动画 class。
 * 仅在流式渲染阶段使用，避免已渲染文字重复动画。
 */
export function rehypeSplitWordsIntoSpans(options = {}) {
  const { previousContentLength = 0 } = options;

  return (tree) => {
    let currentCharCount = 0; // 当前已处理的字符数

    visit(tree, 'element', (node) => {
      if (
        ['p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'li', 'strong'].includes(node.tagName) &&
        node.children
      ) {
        const newChildren = [];
        node.children.forEach((child) => {
          if (child.type === 'text') {
            try {
              // 使用 Intl.Segmenter 精准拆分中英文及标点
              const segmenter = new Intl.Segmenter('zh', { granularity: 'word' });
              const segments = segmenter.segment(child.value);

              Array.from(segments)
                .map((seg) => seg.segment)
                .filter(Boolean)
                .forEach((word) => {
                  const wordStartPos = currentCharCount;
                  const wordEndPos = currentCharCount + word.length;

                  // 判断这个词是否是新增的（在 previousContentLength 之后）
                  const isNewContent = wordStartPos >= previousContentLength;

                  newChildren.push({
                    type: 'element',
                    tagName: 'span',
                    properties: {
                      className: isNewContent ? ['animate-fade-in'] : [],
                    },
                    children: [{ type: 'text', value: word }],
                  });

                  currentCharCount = wordEndPos;
                });
            } catch (_) {
              // Fallback：如果浏览器不支持 Segmenter
              const textStartPos = currentCharCount;
              const isNewContent = textStartPos >= previousContentLength;

              if (isNewContent) {
                // 新内容，添加动画
                newChildren.push({
                  type: 'element',
                  tagName: 'span',
                  properties: {
                    className: ['animate-fade-in'],
                  },
                  children: [{ type: 'text', value: child.value }],
                });
              } else {
                // 旧内容，不添加动画
                newChildren.push(child);
              }

              currentCharCount += child.value.length;
            }
          } else {
            newChildren.push(child);
          }
        });
        node.children = newChildren;
      }
    });
  };
} 