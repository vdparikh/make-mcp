import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface TransformNodeData {
  label: string;
  description?: string;
  config?: {
    expression?: string;
  };
}

function TransformNode({ data, selected }: NodeProps<TransformNodeData>) {
  return (
    <div className={`flow-node transform-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-target" />
      <div className="node-header" style={{ background: '#06b6d4' }}>
        <i className="bi bi-shuffle"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        <p className="node-description">{data.description || 'Transform data'}</p>
        {data.config?.expression && (
          <div className="expression-preview">
            <code>{data.config.expression}</code>
          </div>
        )}
        <div className="node-io">
          <div className="node-inputs">
            <div className="io-item">
              <span className="io-dot input"></span>
              <span>data</span>
            </div>
          </div>
          <div className="node-outputs">
            <div className="io-item">
              <span>result</span>
              <span className="io-dot output"></span>
            </div>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className="handle-source" />
    </div>
  );
}

export default memo(TransformNode);
