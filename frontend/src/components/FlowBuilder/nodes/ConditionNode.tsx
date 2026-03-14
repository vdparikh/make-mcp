import { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';

interface ConditionNodeData {
  label: string;
  description?: string;
  config?: {
    condition?: string;
    trueLabel?: string;
    falseLabel?: string;
  };
}

function ConditionNode({ data, selected }: NodeProps<ConditionNodeData>) {
  const trueLabel = data.config?.trueLabel || 'Yes';
  const falseLabel = data.config?.falseLabel || 'No';
  
  return (
    <div className={`flow-node condition-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Left} className="handle-target" />
      <div className="node-header" style={{ background: '#f97316' }}>
        <i className="bi bi-signpost-split"></i>
        <span>{data.label}</span>
      </div>
      <div className="node-body">
        {data.config?.condition && (
          <div className="condition-preview">
            <code>if ({data.config.condition})</code>
          </div>
        )}
        <div className="condition-outputs">
          <div className="condition-branch true">
            <span className="branch-label">{trueLabel}</span>
            <Handle
              type="source"
              position={Position.Right}
              id="true"
              className="handle-source handle-true"
              style={{ top: '40%' }}
            />
          </div>
          <div className="condition-branch false">
            <span className="branch-label">{falseLabel}</span>
            <Handle
              type="source"
              position={Position.Right}
              id="false"
              className="handle-source handle-false"
              style={{ top: '70%' }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

export default memo(ConditionNode);
