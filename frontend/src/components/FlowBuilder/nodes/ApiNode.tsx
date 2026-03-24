import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface ApiNodeData {
  label: string;
  description?: string;
  config?: {
    url?: string;
    method?: string;
  };
}

function ApiNode({ data, selected }: NodeProps<ApiNodeData>) {
  const method = data.config?.method || 'GET';
  const url = data.config?.url || 'Not configured';
  
  return (
    <div className={`flow-node api-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-target" />
      <div className="node-header" style={{ background: '#6366f1' }}>
        <i className="bi bi-globe"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        <div className="node-config-preview">
          <span className={`method-badge method-${method.toLowerCase()}`}>{method}</span>
          <span className="url-preview">{url.length > 30 ? url.substring(0, 30) + '...' : url}</span>
        </div>
        <div className="node-io">
          <div className="node-inputs">
            <div className="io-item">
              <span className="io-dot input"></span>
              <span>request</span>
            </div>
          </div>
          <div className="node-outputs">
            <div className="io-item">
              <span>response</span>
              <span className="io-dot output"></span>
            </div>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className="handle-source" />
    </div>
  );
}

export default memo(ApiNode);
