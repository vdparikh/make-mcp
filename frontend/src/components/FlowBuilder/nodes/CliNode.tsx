import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface CliNodeData {
  label: string;
  description?: string;
  config?: {
    command?: string;
    timeout?: number;
  };
}

function CliNode({ data, selected }: NodeProps<CliNodeData>) {
  const command = data.config?.command || 'Not configured';
  
  return (
    <div className={`flow-node cli-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-target" />
      <div className="node-header" style={{ background: '#f59e0b' }}>
        <i className="bi bi-terminal"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        <div className="command-preview">
          <code>{command.length > 35 ? command.substring(0, 35) + '...' : command}</code>
        </div>
        <div className="node-io">
          <div className="node-inputs">
            <div className="io-item">
              <span className="io-dot input"></span>
              <span>args</span>
            </div>
          </div>
          <div className="node-outputs">
            <div className="io-item">
              <span>stdout</span>
              <span className="io-dot output"></span>
            </div>
            <div className="io-item">
              <span>stderr</span>
              <span className="io-dot output error"></span>
            </div>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className="handle-source" />
    </div>
  );
}

export default memo(CliNode);
