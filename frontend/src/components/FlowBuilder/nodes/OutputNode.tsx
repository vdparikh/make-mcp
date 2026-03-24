import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface OutputNodeData {
  label: string;
  description?: string;
  config?: {
    format?: string;
  };
}

function OutputNode({ data, selected }: NodeProps<OutputNodeData>) {
  const format = data.config?.format || 'json';
  
  return (
    <div className={`flow-node output-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-target" />
      <div className="node-header" style={{ background: '#ef4444' }}>
        <i className="bi bi-box-arrow-right"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        <p className="node-description">Return result to AI agent</p>
        <div className="output-format">
          <span className="format-label">Format:</span>
          <span className="format-value">{format.toUpperCase()}</span>
        </div>
      </div>
    </div>
  );
}

export default memo(OutputNode);
