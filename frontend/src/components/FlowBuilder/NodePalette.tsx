import { DragEvent } from 'react';

interface NodeType {
  type: string;
  label: string;
  icon: string;
  color: string;
  description: string;
  category: string;
  defaultConfig?: Record<string, unknown>;
}

const nodeTypes: NodeType[] = [
  // Triggers
  {
    type: 'trigger',
    label: 'Tool Input',
    icon: 'bi-play-circle',
    color: '#10b981',
    description: 'Entry point for tool execution',
    category: 'Triggers',
    defaultConfig: { inputSchema: {} },
  },
  
  // Actions
  {
    type: 'api',
    label: 'REST API',
    icon: 'bi-globe',
    color: '#6366f1',
    description: 'Call an external REST API',
    category: 'Actions',
    defaultConfig: { url: '', method: 'GET', headers: {} },
  },
  {
    type: 'api',
    label: 'GraphQL',
    icon: 'bi-diagram-3',
    color: '#e535ab',
    description: 'Execute a GraphQL query',
    category: 'Actions',
    defaultConfig: { url: '', query: '', variables: {} },
  },
  {
    type: 'cli',
    label: 'CLI Command',
    icon: 'bi-terminal',
    color: '#f59e0b',
    description: 'Execute a shell command',
    category: 'Actions',
    defaultConfig: { command: '', timeout: 30000 },
  },
  {
    type: 'api',
    label: 'Webhook',
    icon: 'bi-link-45deg',
    color: '#8b5cf6',
    description: 'Send data to a webhook',
    category: 'Actions',
    defaultConfig: { url: '', method: 'POST' },
  },
  
  // Logic
  {
    type: 'transform',
    label: 'Transform',
    icon: 'bi-shuffle',
    color: '#06b6d4',
    description: 'Transform or map data',
    category: 'Logic',
    defaultConfig: { expression: '' },
  },
  {
    type: 'condition',
    label: 'Condition',
    icon: 'bi-signpost-split',
    color: '#f97316',
    description: 'Branch based on condition',
    category: 'Logic',
    defaultConfig: { condition: '', trueLabel: 'Yes', falseLabel: 'No' },
  },
  {
    type: 'transform',
    label: 'Filter',
    icon: 'bi-funnel',
    color: '#14b8a6',
    description: 'Filter array items',
    category: 'Logic',
    defaultConfig: { filterExpression: '' },
  },
  {
    type: 'transform',
    label: 'Aggregate',
    icon: 'bi-collection',
    color: '#a855f7',
    description: 'Combine multiple inputs',
    category: 'Logic',
    defaultConfig: { aggregationType: 'array' },
  },
  
  // Output
  {
    type: 'output',
    label: 'Output',
    icon: 'bi-box-arrow-right',
    color: '#ef4444',
    description: 'Return result to AI agent',
    category: 'Output',
    defaultConfig: { format: 'json' },
  },
];

const categories = ['Triggers', 'Actions', 'Logic', 'Output'];

export default function NodePalette() {
  const onDragStart = (event: DragEvent, nodeType: NodeType) => {
    event.dataTransfer.setData('application/reactflow', nodeType.type);
    event.dataTransfer.setData('application/nodedata', JSON.stringify({
      label: nodeType.label,
      description: nodeType.description,
      defaultConfig: nodeType.defaultConfig,
    }));
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div className="node-palette">
      <div className="palette-header">
        <h3>
          <i className="bi bi-grid-3x3-gap"></i>
          Nodes
        </h3>
      </div>
      
      <div className="palette-content">
        {categories.map((category) => (
          <div key={category} className="palette-category">
            <div className="category-title">{category}</div>
            <div className="category-nodes">
              {nodeTypes
                .filter((node) => node.category === category)
                .map((node, index) => (
                  <div
                    key={`${node.type}-${index}`}
                    className="palette-node"
                    draggable
                    onDragStart={(e) => onDragStart(e, node)}
                    style={{ '--node-color': node.color } as React.CSSProperties}
                  >
                    <div className="palette-node-icon" style={{ background: node.color }}>
                      <i className={`bi ${node.icon}`}></i>
                    </div>
                    <div className="palette-node-info">
                      <div className="palette-node-label">{node.label}</div>
                      <div className="palette-node-desc">{node.description}</div>
                    </div>
                  </div>
                ))}
            </div>
          </div>
        ))}
      </div>
      
      <div className="palette-footer">
        <p>Drag nodes to the canvas to build your tool flow</p>
      </div>
    </div>
  );
}
