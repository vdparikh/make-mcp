import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface TriggerNodeData {
  label: string;
  description?: string;
}

function TriggerNode({ data, selected }: NodeProps<TriggerNodeData>) {
  return (
    <div className={`flow-node trigger-node ${selected ? 'selected' : ''}`}>
      <div className="node-header" style={{ background: '#10b981' }}>
        <i className="bi bi-play-circle"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        <p>{data.description || 'Tool entry point'}</p>
        <div className="node-outputs">
          <div className="output-item">
            <span className="output-label">input</span>
            <span className="output-type">object</span>
          </div>
        </div>
      </div>
      <Handle type="source" position={Position.Right} className="handle-source" />
    </div>
  );
}

export default memo(TriggerNode);
