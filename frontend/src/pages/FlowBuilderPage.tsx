import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'react-toastify';
import { Node, Edge } from 'reactflow';
import Editor from '@monaco-editor/react';
import FlowBuilder, { FlowData } from '../components/FlowBuilder';
import {
  createFlow,
  updateFlow,
  getFlow,
  executeFlow,
  convertFlowToTool,
  type Flow,
  type FlowExecutionResult,
} from '../services/api';
import '../components/FlowBuilder/flow-styles.css';
import { sanitizeMcpToolName, validateMcpToolName } from '../utils/mcpToolName';

export default function FlowBuilderPage() {
  const { id: serverId } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const flowId = searchParams.get('flowId');
  const navigate = useNavigate();
  
  const [loading, setLoading] = useState(false);
  const [currentFlow, setCurrentFlow] = useState<Flow | null>(null);
  const [showTestPanel, setShowTestPanel] = useState(false);
  const [showConvertModal, setShowConvertModal] = useState(false);
  const [testInput, setTestInput] = useState('{\n  "query": "test input"\n}');
  const [testResult, setTestResult] = useState<FlowExecutionResult | null>(null);
  const [testing, setTesting] = useState(false);
  const [converting, setConverting] = useState(false);
  const [toolName, setToolName] = useState('');
  const [toolDescription, setToolDescription] = useState('');
  const [flowNodes, setFlowNodes] = useState<Node[]>([]);
  const [flowEdges, setFlowEdges] = useState<Edge[]>([]);

  useEffect(() => {
    if (flowId) {
      loadFlow(flowId);
    } else {
      setCurrentFlow(null);
    }
  }, [flowId]);

  // Show error if no server ID
  if (!serverId) {
    return (
      <div className="page">
        <div className="page-header">
          <div>
            <h1 className="page-title">
              <i className="bi bi-diagram-3 page-title-icon"></i>
              Visual Flow Builder
            </h1>
          </div>
        </div>
        <div style={{
          background: 'var(--card-bg)',
          border: '1px solid var(--card-border)',
          borderRadius: '12px',
          padding: '3rem',
          textAlign: 'center',
        }}>
          <i className="bi bi-exclamation-triangle" style={{ fontSize: '3rem', color: 'var(--warning-color)', marginBottom: '1rem', display: 'block' }}></i>
          <h3 style={{ color: 'var(--text-primary)', marginBottom: '0.5rem' }}>Server Required</h3>
          <p style={{ color: 'var(--text-secondary)', marginBottom: '1.5rem' }}>
            Please select a server first to create flows. Flows must be associated with a server.
          </p>
          <Link to="/" className="btn btn-primary">
            <i className="bi bi-arrow-left"></i>
            Go to Dashboard
          </Link>
        </div>
      </div>
    );
  }

  const loadFlow = async (id: string) => {
    setLoading(true);
    try {
      const flow = await getFlow(id);
      setCurrentFlow(flow);
    } catch {
      toast.error('Failed to load flow');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (flowData: FlowData) => {
    if (!serverId) {
      toast.error('Server ID is required');
      return;
    }

    try {
      const payload = {
        server_id: serverId,
        name: flowData.name || 'Untitled Flow',
        description: flowData.description || '',
        nodes: flowData.nodes.map(n => ({
          id: n.id,
          type: n.type || 'unknown',
          position: n.position,
          data: n.data,
        })),
        edges: flowData.edges.map(e => ({
          id: e.id,
          source: e.source,
          target: e.target,
          sourceHandle: e.sourceHandle || undefined,
          targetHandle: e.targetHandle || undefined,
        })),
      };

      if (currentFlow?.id) {
        const updated = await updateFlow(currentFlow.id, payload);
        setCurrentFlow(updated);
        toast.success('Flow updated successfully!');
      } else {
        const created = await createFlow(payload);
        setCurrentFlow(created);
        toast.success('Flow saved successfully!');
        navigate(`/servers/${serverId}/flow?flowId=${created.id}`, { replace: true });
      }
      
    } catch (err) {
      toast.error('Failed to save flow');
      console.error(err);
    }
  };

  const handleExecute = async () => {
    if (!currentFlow?.id) {
      toast.warning('Please save the flow first before testing');
      setShowTestPanel(true);
      return;
    }
    setShowTestPanel(true);
  };

  const runTest = async () => {
    if (!currentFlow?.id) return;
    
    setTesting(true);
    setTestResult(null);
    
    try {
      let input = {};
      try {
        input = JSON.parse(testInput);
      } catch {
        toast.error('Invalid JSON input');
        setTesting(false);
        return;
      }

      const result = await executeFlow(currentFlow.id, input);
      setTestResult(result);
      
      if (result.success) {
        toast.success(`Flow executed in ${result.duration_ms}ms`);
      } else {
        toast.error(`Flow failed: ${result.error}`);
      }
    } catch (err) {
      toast.error('Failed to execute flow');
      console.error(err);
    } finally {
      setTesting(false);
    }
  };

  const handleConvert = (nodes: Node[], edges: Edge[]) => {
    setFlowNodes(nodes);
    setFlowEdges(edges);
    
    if (!currentFlow?.id) {
      toast.warning('Please save the flow first before converting to a tool');
      return;
    }
    
    setToolName(sanitizeMcpToolName(currentFlow.name.replace(/\s+/g, '_').toLowerCase()));
    setToolDescription(currentFlow.description || `Tool generated from ${currentFlow.name} flow`);
    setShowConvertModal(true);
  };

  const performConvert = async () => {
    if (!currentFlow?.id || !toolName) return;

    const tn = toolName.trim();
    const nameErr = validateMcpToolName(tn);
    if (nameErr) {
      toast.error(nameErr);
      return;
    }

    setConverting(true);
    try {
      await convertFlowToTool(currentFlow.id, tn, toolDescription);
      toast.success('Flow converted to tool successfully!');
      setShowConvertModal(false);
      
      if (serverId) {
        navigate(`/servers/${serverId}`);
      }
    } catch (err) {
      toast.error('Failed to convert flow to tool');
      console.error(err);
    } finally {
      setConverting(false);
    }
  };

  const initialFlow: FlowData | undefined = currentFlow ? {
    id: currentFlow.id,
    name: currentFlow.name,
    description: currentFlow.description,
    nodes: currentFlow.nodes,
    edges: currentFlow.edges,
  } : undefined;

  if (loading) {
    return (
      <div className="page" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '60vh' }}>
        <div className="spinner-border text-primary" role="status">
          <span className="visually-hidden">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <nav className="page-breadcrumb">
            <Link to="/" className="page-breadcrumb-link">
              Dashboard
            </Link>
            <span className="page-breadcrumb-sep">/</span>
            {serverId && (
              <>
                <Link to={`/servers/${serverId}`} className="page-breadcrumb-link">
                  Server
                </Link>
                <span className="page-breadcrumb-sep">/</span>
              </>
            )}
            <span className="page-breadcrumb-current">Flow Builder</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-diagram-3 page-title-icon"></i>
            Visual Flow Builder
          </h1>
          <p className="page-subtitle">
            Build tool pipelines visually by dragging and connecting nodes
          </p>
        </div>
        <Link to={serverId ? `/servers/${serverId}` : '/'} className="btn btn-secondary btn-sm">
          <i className="bi bi-arrow-left" aria-hidden="true" />
          Back to server
        </Link>
      </div>

      <div style={{ display: 'flex', gap: '1rem' }}>
        <div style={{ flex: 1 }}>
          <FlowBuilder
            key={flowId ?? 'draft'}
            flowId={currentFlow?.id}
            initialFlow={initialFlow}
            onSave={handleSave}
            onExecute={handleExecute}
            onConvert={handleConvert}
          />
        </div>
        
        {showTestPanel && (
          <div style={{
            width: '400px',
            background: 'var(--card-bg)',
            border: '1px solid var(--card-border)',
            borderRadius: '12px',
            display: 'flex',
            flexDirection: 'column',
          }}>
            <div style={{ 
              padding: '1rem', 
              borderBottom: '1px solid var(--card-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between'
            }}>
              <h4 style={{ margin: 0, color: 'var(--text-primary)', fontSize: '0.9375rem' }}>
                <i className="bi bi-play-circle" style={{ marginRight: '0.5rem', color: 'var(--success-color)' }}></i>
                Test Flow
              </h4>
              <button className="btn btn-icon btn-sm" onClick={() => setShowTestPanel(false)}>
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            
            <div style={{ padding: '1rem', flex: 1, overflow: 'auto' }}>
              <div className="form-group" style={{ marginBottom: '1rem' }}>
                <label className="form-label">Input JSON</label>
                <div style={{ height: '150px', borderRadius: '8px', overflow: 'hidden', border: '1px solid var(--card-border)' }}>
                  <Editor
                    height="100%"
                    language="json"
                    theme="vs-dark"
                    value={testInput}
                    onChange={(value) => setTestInput(value || '{}')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 12,
                      lineNumbers: 'off',
                      scrollBeyondLastLine: false,
                    }}
                  />
                </div>
              </div>
              
              <button 
                className="btn btn-success" 
                style={{ width: '100%' }}
                onClick={runTest}
                disabled={testing || !currentFlow?.id}
              >
                {testing ? (
                  <>
                    <span className="spinner-border spinner-border-sm" style={{ marginRight: '0.5rem' }}></span>
                    Executing...
                  </>
                ) : (
                  <>
                    <i className="bi bi-play-fill"></i>
                    Run Test
                  </>
                )}
              </button>
              
              {!currentFlow?.id && (
                <p style={{ fontSize: '0.75rem', color: 'var(--warning-color)', marginTop: '0.5rem', textAlign: 'center' }}>
                  Save the flow first to enable testing
                </p>
              )}
              
              {testResult && (
                <div style={{ marginTop: '1rem' }}>
                  <div style={{ 
                    padding: '0.75rem',
                    background: testResult.success ? 'rgba(34, 197, 94, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                    border: `1px solid ${testResult.success ? 'var(--success-color)' : 'var(--danger-color)'}`,
                    borderRadius: '8px',
                    marginBottom: '0.75rem'
                  }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.5rem' }}>
                      <i className={`bi ${testResult.success ? 'bi-check-circle-fill' : 'bi-x-circle-fill'}`} 
                         style={{ color: testResult.success ? 'var(--success-color)' : 'var(--danger-color)' }}></i>
                      <span style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                        {testResult.success ? 'Success' : 'Failed'}
                      </span>
                      <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>
                        {testResult.duration_ms}ms
                      </span>
                    </div>
                    {testResult.error && (
                      <p style={{ color: 'var(--danger-color)', fontSize: '0.8125rem', margin: 0 }}>
                        {testResult.error}
                      </p>
                    )}
                  </div>
                  
                  {testResult.node_results && testResult.node_results.length > 0 && (
                    <div>
                      <h5 style={{ color: 'var(--text-secondary)', fontSize: '0.75rem', marginBottom: '0.5rem', textTransform: 'uppercase' }}>
                        Node Results
                      </h5>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                        {testResult.node_results.map((nr, idx) => (
                          <div key={idx} style={{
                            padding: '0.5rem',
                            background: 'var(--dark-bg)',
                            borderRadius: '6px',
                            fontSize: '0.75rem',
                          }}>
                            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                              <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{nr.node_type}</span>
                              <span style={{ 
                                color: nr.success ? 'var(--success-color)' : 'var(--danger-color)',
                                display: 'flex',
                                alignItems: 'center',
                                gap: '0.25rem'
                              }}>
                                <i className={`bi ${nr.success ? 'bi-check' : 'bi-x'}`}></i>
                                {nr.duration_ms}ms
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                  
                  {testResult.output && (
                    <div style={{ marginTop: '0.75rem' }}>
                      <h5 style={{ color: 'var(--text-secondary)', fontSize: '0.75rem', marginBottom: '0.5rem', textTransform: 'uppercase' }}>
                        Output
                      </h5>
                      <pre style={{
                        background: 'var(--dark-bg)',
                        padding: '0.75rem',
                        borderRadius: '6px',
                        fontSize: '0.6875rem',
                        overflow: 'auto',
                        maxHeight: '150px',
                        color: 'var(--text-secondary)',
                        margin: 0,
                      }}>
                        {JSON.stringify(testResult.output, null, 2)}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
      
      <div style={{ 
        marginTop: '1rem', 
        padding: '1rem', 
        background: 'linear-gradient(135deg, rgba(129, 140, 248, 0.1), rgba(56, 189, 248, 0.05))',
        borderRadius: '12px',
        border: '1px solid rgba(129, 140, 248, 0.2)'
      }}>
        <h4 style={{ color: 'var(--text-primary)', marginBottom: '0.5rem', fontSize: '0.9375rem' }}>
          <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
          Quick Tips
        </h4>
        <ul style={{ color: 'var(--text-secondary)', fontSize: '0.8125rem', margin: 0, paddingLeft: '1.25rem' }}>
          <li>
            Start from the server (<strong>New flow</strong>) or open an existing flow with <strong>Edit flow</strong> on a flow-based tool
          </li>
          <li>Drag nodes from the palette on the left to add them to the canvas</li>
          <li>Connect nodes by dragging from output handles (right) to input handles (left)</li>
          <li>Click a node to configure it in the right panel</li>
          <li>Save your flow first, then use "Test" to execute it with sample input</li>
          <li>
            After <strong>Save flow</strong>, use <strong>Add as tool</strong> if you want an MCP tool on this server (that is separate from saving the graph)
          </li>
        </ul>
      </div>

      {/* Add as tool modal */}
      {showConvertModal && (
        <div className="flow-modal-overlay" onClick={() => setShowConvertModal(false)}>
          <div className="flow-modal" onClick={(e) => e.stopPropagation()}>
            <div className="flow-modal-header">
              <h3>
                <i className="bi bi-magic" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
                Add as MCP tool
              </h3>
              <button type="button" className="btn btn-icon" onClick={() => setShowConvertModal(false)}>
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            <div className="flow-modal-body">
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem' }}>
                Creates a <strong>new tool</strong> on this server that runs this saved flow. This is not the same as <strong>Save flow</strong> (which only stores the graph).
              </p>
              
              <div className="form-group">
                <label className="form-label">Tool Name</label>
                <input
                  type="text"
                  className="form-control"
                  value={toolName}
                  onChange={(e) => {
                    const v = e.target.value;
                    const runes = [...v];
                    setToolName(runes.length > 128 ? runes.slice(0, 128).join('') : v);
                  }}
                  placeholder="my_flow_tool"
                  maxLength={128}
                  autoComplete="off"
                  spellCheck={false}
                />
                <small style={{ color: 'var(--text-muted)' }}>
                  1–128 characters: letters, digits, <code>_</code>, <code>-</code>, <code>.</code> only (case-sensitive, unique per server).
                </small>
              </div>
              
              <div className="form-group">
                <label className="form-label">Description</label>
                <textarea
                  className="form-control"
                  value={toolDescription}
                  onChange={(e) => setToolDescription(e.target.value)}
                  placeholder="What does this tool do?"
                  rows={3}
                />
              </div>
              
              <div style={{ 
                padding: '0.75rem',
                background: 'var(--dark-bg)',
                borderRadius: '8px',
                marginTop: '1rem'
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.5rem' }}>
                  <i className="bi bi-info-circle" style={{ color: 'var(--primary-color)' }}></i>
                  <span style={{ color: 'var(--text-primary)', fontWeight: 500, fontSize: '0.8125rem' }}>Flow Summary</span>
                </div>
                <ul style={{ margin: 0, paddingLeft: '1.25rem', color: 'var(--text-secondary)', fontSize: '0.75rem' }}>
                  <li>{flowNodes.length} nodes in the flow</li>
                  <li>{flowEdges.length} connections</li>
                  <li>Flow ID: {currentFlow?.id?.slice(0, 8)}...</li>
                </ul>
              </div>
            </div>
            <div className="flow-modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowConvertModal(false)}>Cancel</button>
              <button 
                className="btn btn-warning" 
                onClick={performConvert}
                disabled={converting || !toolName}
              >
                {converting ? (
                  <>
                    <span className="spinner-border spinner-border-sm" style={{ marginRight: '0.5rem' }}></span>
                    Adding tool...
                  </>
                ) : (
                  <>
                    <i className="bi bi-magic"></i>
                    Add tool
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
