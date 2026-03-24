import { useCallback, useRef, useState, useEffect, DragEvent } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ReactFlowProvider,
  ReactFlowInstance,
  BackgroundVariant,
  Panel,
} from 'reactflow';
import 'reactflow/dist/style.css';

import TriggerNode from './nodes/TriggerNode';
import ApiNode from './nodes/ApiNode';
import CliNode from './nodes/CliNode';
import TransformNode from './nodes/TransformNode';
import ConditionNode from './nodes/ConditionNode';
import OutputNode from './nodes/OutputNode';
import NodePalette from './NodePalette';
import NodeConfigPanel from './NodeConfigPanel';
import './flow-styles.css';

const nodeTypes = {
  trigger: TriggerNode,
  api: ApiNode,
  cli: CliNode,
  transform: TransformNode,
  condition: ConditionNode,
  output: OutputNode,
};

export interface FlowData {
  id?: string;
  name?: string;
  description?: string;
  nodes: Node[];
  edges: Edge[];
}

interface FlowBuilderProps {
  flowId?: string;
  initialFlow?: FlowData;
  onSave?: (flowData: FlowData) => Promise<void> | void;
  onExecute?: (nodes: Node[], edges: Edge[]) => void;
  onConvert?: (nodes: Node[], edges: Edge[]) => void;
}

const defaultNodes: Node[] = [
  {
    id: 'trigger-1',
    type: 'trigger',
    position: { x: 100, y: 200 },
    data: { label: 'Tool Input', description: 'Receives input from AI agent' },
  },
];

const defaultEdges: Edge[] = [];

let nodeId = 100;
const getNodeId = () => `node_${nodeId++}`;

export default function FlowBuilder({ 
  flowId,
  initialFlow,
  onSave, 
  onExecute,
  onConvert 
}: FlowBuilderProps) {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState(initialFlow?.nodes || defaultNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialFlow?.edges || defaultEdges);
  const [reactFlowInstance, setReactFlowInstance] = useState<ReactFlowInstance | null>(null);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [flowName, setFlowName] = useState(initialFlow?.name || 'Untitled Flow');
  const [flowDescription, setFlowDescription] = useState(initialFlow?.description || '');
  const [showSaveModal, setShowSaveModal] = useState(false);

  useEffect(() => {
    if (initialFlow) {
      setNodes(initialFlow.nodes);
      setEdges(initialFlow.edges);
      setFlowName(initialFlow.name || 'Untitled Flow');
      setFlowDescription(initialFlow.description || '');
    }
  }, [initialFlow, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge({ ...params, animated: true, style: { stroke: '#818cf8' } }, eds)),
    [setEdges]
  );

  const onDragOver = useCallback((event: DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: DragEvent) => {
      event.preventDefault();

      if (!reactFlowWrapper.current || !reactFlowInstance) return;

      const type = event.dataTransfer.getData('application/reactflow');
      const nodeData = JSON.parse(event.dataTransfer.getData('application/nodedata') || '{}');

      if (!type) return;

      const position = reactFlowInstance.screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      const newNode: Node = {
        id: getNodeId(),
        type,
        position,
        data: { 
          label: nodeData.label || 'New Node',
          description: nodeData.description || '',
          config: nodeData.defaultConfig || {},
        },
      };

      setNodes((nds) => nds.concat(newNode));
    },
    [reactFlowInstance, setNodes]
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const updateNodeData = useCallback((nodeId: string, newData: Record<string, unknown>) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return { ...node, data: { ...node.data, ...newData } };
        }
        return node;
      })
    );
    if (selectedNode && selectedNode.id === nodeId) {
      setSelectedNode((prev) => prev ? { ...prev, data: { ...prev.data, ...newData } } : null);
    }
  }, [setNodes, selectedNode]);

  const deleteNode = useCallback((nodeId: string) => {
    setNodes((nds) => nds.filter((node) => node.id !== nodeId));
    setEdges((eds) => eds.filter((edge) => edge.source !== nodeId && edge.target !== nodeId));
    setSelectedNode(null);
  }, [setNodes, setEdges]);

  const [isSaving, setIsSaving] = useState(false);

  const handleSave = useCallback(async () => {
    if (onSave) {
      setIsSaving(true);
      try {
        await onSave({
          id: flowId,
          name: flowName,
          description: flowDescription,
          nodes,
          edges,
        });
        setShowSaveModal(false);
      } catch (err) {
        console.error('Save error:', err);
      } finally {
        setIsSaving(false);
      }
    } else {
      setShowSaveModal(false);
    }
  }, [flowId, flowName, flowDescription, nodes, edges, onSave]);

  const handleExecute = useCallback(() => {
    if (onExecute) {
      onExecute(nodes, edges);
    }
  }, [nodes, edges, onExecute]);

  const handleConvert = useCallback(() => {
    if (onConvert) {
      onConvert(nodes, edges);
    }
  }, [nodes, edges, onConvert]);

  return (
    <div className="flow-builder">
      <ReactFlowProvider>
        <div className="flow-builder-container">
          <NodePalette />
          
          <div className="flow-canvas" ref={reactFlowWrapper}>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onInit={setReactFlowInstance}
              onDrop={onDrop}
              onDragOver={onDragOver}
              onNodeClick={onNodeClick}
              onPaneClick={onPaneClick}
              nodeTypes={nodeTypes}
              fitView
              snapToGrid
              snapGrid={[15, 15]}
              defaultEdgeOptions={{
                animated: true,
                style: { stroke: '#818cf8', strokeWidth: 2 },
              }}
            >
              <Controls />
              <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="#374151" />
              
              <Panel position="top-left">
                <div style={{ 
                  background: 'var(--card-bg)', 
                  padding: '0.75rem 1rem',
                  borderRadius: '8px',
                  border: '1px solid var(--card-border)'
                }}>
                  <input
                    type="text"
                    value={flowName}
                    onChange={(e) => setFlowName(e.target.value)}
                    style={{
                      background: 'transparent',
                      border: 'none',
                      color: 'var(--text-primary)',
                      fontSize: '1rem',
                      fontWeight: 600,
                      width: '200px',
                    }}
                    placeholder="Flow name..."
                  />
                </div>
              </Panel>
              
              <Panel position="top-right">
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button className="btn btn-secondary btn-sm" onClick={() => { setNodes(defaultNodes); setEdges(defaultEdges); }}>
                    <i className="bi bi-arrow-counterclockwise"></i>
                    Reset
                  </button>
                  {onExecute && (
                    <button className="btn btn-success btn-sm" onClick={handleExecute}>
                      <i className="bi bi-play-fill"></i>
                      Test
                    </button>
                  )}
                  {onConvert && (
                    <button className="btn btn-warning btn-sm" onClick={handleConvert}>
                      <i className="bi bi-magic"></i>
                      Convert to Tool
                    </button>
                  )}
                  <button className="btn btn-primary btn-sm" onClick={() => setShowSaveModal(true)}>
                    <i className="bi bi-save"></i>
                    Save
                  </button>
                </div>
              </Panel>
            </ReactFlow>
          </div>

          {selectedNode && (
            <NodeConfigPanel
              node={selectedNode}
              onUpdate={updateNodeData}
              onDelete={deleteNode}
              onClose={() => setSelectedNode(null)}
            />
          )}
        </div>
      </ReactFlowProvider>

      {/* Save Modal */}
      {showSaveModal && (
        <div className="flow-modal-overlay" onClick={() => !isSaving && setShowSaveModal(false)}>
          <div className="flow-modal" onClick={(e) => e.stopPropagation()}>
            <div className="flow-modal-header">
              <h3>Save Flow</h3>
              <button className="btn btn-icon" onClick={() => setShowSaveModal(false)} disabled={isSaving}>
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            <div className="flow-modal-body">
              <div className="form-group">
                <label className="form-label">Flow Name</label>
                <input
                  type="text"
                  className="form-control"
                  value={flowName}
                  onChange={(e) => setFlowName(e.target.value)}
                  placeholder="My Flow"
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description</label>
                <textarea
                  className="form-control"
                  value={flowDescription}
                  onChange={(e) => setFlowDescription(e.target.value)}
                  placeholder="What does this flow do?"
                  rows={3}
                />
              </div>
              <div style={{ 
                padding: '0.75rem',
                background: 'var(--dark-bg)',
                borderRadius: '8px',
                fontSize: '0.8125rem',
                color: 'var(--text-secondary)'
              }}>
                <i className="bi bi-info-circle" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                This flow has <strong style={{ color: 'var(--text-primary)' }}>{nodes.length}</strong> nodes and <strong style={{ color: 'var(--text-primary)' }}>{edges.length}</strong> connections.
              </div>
            </div>
            <div className="flow-modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowSaveModal(false)} disabled={isSaving}>Cancel</button>
              <button className="btn btn-primary" onClick={handleSave} disabled={isSaving || !flowName.trim()}>
                {isSaving ? (
                  <>
                    <span className="spinner-border spinner-border-sm" style={{ marginRight: '0.5rem' }}></span>
                    Saving...
                  </>
                ) : (
                  <>
                    <i className="bi bi-save"></i>
                    Save Flow
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
