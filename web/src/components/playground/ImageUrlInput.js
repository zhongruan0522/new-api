import React from 'react';
import {
  Input,
  Typography,
  Button,
} from '@douyinfe/semi-ui';
import { IconFile } from '@douyinfe/semi-icons';
import {
  FileText,
  Plus,
  X,
} from 'lucide-react';

const ImageUrlInput = ({ imageUrls, onImageUrlsChange }) => {
  const handleAddImageUrl = () => {
    const newUrls = [...imageUrls, ''];
    onImageUrlsChange(newUrls);
  };

  const handleUpdateImageUrl = (index, value) => {
    const newUrls = [...imageUrls];
    newUrls[index] = value;
    onImageUrlsChange(newUrls);
  };

  const handleRemoveImageUrl = (index) => {
    const newUrls = imageUrls.filter((_, i) => i !== index);
    onImageUrlsChange(newUrls);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <FileText size={16} className="text-gray-500" />
          <Typography.Text strong className="text-sm">
            图片地址
          </Typography.Text>
          <Typography.Text className="text-xs text-gray-400">
            (多模态对话)
          </Typography.Text>
        </div>
        <Button
          icon={<Plus size={14} />}
          size="small"
          theme="solid"
          type="primary"
          onClick={handleAddImageUrl}
          className="!rounded-full !w-4 !h-4 !p-0 !min-w-0"
          disabled={imageUrls.length >= 5}
        />
      </div>

      {imageUrls.length === 0 ? (
        <Typography.Text className="text-xs text-gray-500 mb-2 block">
          点击 + 按钮添加图片URL，支持最多5张图片
        </Typography.Text>
      ) : (
        <Typography.Text className="text-xs text-gray-500 mb-2 block">
          已添加 {imageUrls.length}/5 张图片
        </Typography.Text>
      )}

      <div className="space-y-2 max-h-32 overflow-y-auto">
        {imageUrls.map((url, index) => (
          <div key={index} className="flex items-center gap-2">
            <div className="flex-1">
              <Input
                placeholder={`https://example.com/image${index + 1}.jpg`}
                value={url}
                onChange={(value) => handleUpdateImageUrl(index, value)}
                className="!rounded-lg"
                size="small"
                prefix={<IconFile size='small' />}
              />
            </div>
            <Button
              icon={<X size={12} />}
              size="small"
              theme="borderless"
              type="danger"
              onClick={() => handleRemoveImageUrl(index)}
              className="!rounded-full !w-6 !h-6 !p-0 !min-w-0 !text-red-500 hover:!bg-red-50 flex-shrink-0"
            />
          </div>
        ))}
      </div>
    </div>
  );
};

export default ImageUrlInput; 