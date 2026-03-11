import { Link, useLocation } from 'react-router-dom';

export default function Sidebar() {
  const location = useLocation();
  
  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <Link to="/" className="sidebar-logo">
          <div className="sidebar-logo-icon">
            <i className="bi bi-lightning-charge-fill"></i>
          </div>
          <h1>MCP Builder</h1>
        </Link>
      </div>

      <nav className="sidebar-nav">
        <div className="nav-section">
          <div className="nav-section-title">Main</div>
          <Link to="/" className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}>
            <i className="bi bi-grid-3x3-gap-fill"></i>
            <span>Dashboard</span>
          </Link>
          <Link to="/compositions" className={`nav-item ${isActive('/compositions') ? 'active' : ''}`}>
            <i className="bi bi-layers-fill"></i>
            <span>Compositions</span>
          </Link>
        </div>

        <div className="nav-section">
          <div className="nav-section-title">Features</div>
          <div className="nav-item" style={{ opacity: 0.5 }}>
            <i className="bi bi-shield-check"></i>
            <span>Governance</span>
          </div>
          <div className="nav-item" style={{ opacity: 0.5 }}>
            <i className="bi bi-bandaid"></i>
            <span>Self-Healing</span>
          </div>
          <div className="nav-item" style={{ opacity: 0.5 }}>
            <i className="bi bi-person-badge"></i>
            <span>Context Engine</span>
          </div>
        </div>

        <div className="nav-section">
          <div className="nav-section-title">Resources</div>
          <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener noreferrer" className="nav-item">
            <i className="bi bi-book"></i>
            <span>MCP Docs</span>
          </a>
          <a href="#" className="nav-item">
            <i className="bi bi-question-circle"></i>
            <span>Help</span>
          </a>
        </div>
      </nav>

      <div style={{ padding: '1rem', borderTop: '1px solid var(--card-border)' }}>
        <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', textAlign: 'center' }}>
          MCP Server Builder v1.0.0
        </div>
      </div>
    </aside>
  );
}
