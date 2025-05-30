import React from 'react';

const CustomInputRender = (props) => {
  const { detailProps } = props;
  const { clearContextNode, uploadNode, inputNode, sendNode, onClick } = detailProps;

  const styledSendNode = React.cloneElement(sendNode, {
    className: `!rounded-full !bg-purple-500 hover:!bg-purple-600 flex-shrink-0 ${sendNode.props.className || ''}`
  });

  return (
    <div className="p-2 sm:p-4">
      <div
        className="flex items-end gap-2 sm:gap-3 p-2 bg-gray-50 rounded-xl sm:rounded-2xl shadow-sm hover:shadow-md transition-shadow"
        style={{ border: '1px solid var(--semi-color-border)' }}
        onClick={onClick}
      >
        <div className="flex-1">
          {inputNode}
        </div>
        {styledSendNode}
      </div>
    </div>
  );
};

export default CustomInputRender; 