import { useState, useEffect } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  // Auto-collapse only on server edit page; auto-expand when leaving it
  const onServerEditPage = /^\/servers\/[^/]+$/.test(location.pathname);
  const [collapsed, setCollapsed] = useState<boolean>(() => onServerEditPage);

  useEffect(() => {
    setCollapsed(onServerEditPage);
  }, [onServerEditPage]);
  
  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
      <div className="sidebar-header">
        <Link to="/" className="sidebar-logo">
          <div className="sidebar-logo-icon">
            <i className="bi bi-lightning-charge-fill"></i>
          </div>
          <h1>Make MCP</h1>
        </Link>
        <button
          type="button"
          className="sidebar-toggle"
          onClick={() => setCollapsed(!collapsed)}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          <i className={`bi ${collapsed ? 'bi-chevron-right' : 'bi-chevron-left'}`}></i>
        </button>
      </div>

      <nav className="sidebar-nav">
        <div className="nav-section">
          <div className="nav-section-title">Main</div>
          <Link to="/" className={`nav-item ${isActive('/') && location.pathname === '/' ? 'active' : ''}`}>
            <i className="bi bi-grid-3x3-gap-fill"></i>
            <span>Dashboard</span>
          </Link>
          <Link to="/marketplace" className={`nav-item ${isActive('/marketplace') ? 'active' : ''}`}>
            <i className="bi bi-shop"></i>
            <span>Marketplace</span>
          </Link>
          <Link to="/observability" className={`nav-item ${isActive('/observability') ? 'active' : ''}`}>
            <i className="bi bi-graph-up"></i>
            <span>Observability</span>
          </Link>
          <Link to="/compositions" className={`nav-item ${isActive('/compositions') ? 'active' : ''}`}>
            <i className="bi bi-layers-fill"></i>
            <span>Compositions</span>
          </Link>
        </div>

        <div className="nav-section">
          <div className="nav-section-title">Import</div>
          <Link to="/import/openapi" className={`nav-item ${isActive('/import/openapi') ? 'active' : ''}`}>
            <i className="bi bi-file-earmark-code"></i>
            <span>OpenAPI</span>
          </Link>
        </div>


        <div className="nav-section">
          <div className="nav-section-title">Resources</div>
          <Link to="/docs" className={`nav-item ${isActive('/docs') ? 'active' : ''}`}>
            <i className="bi bi-book"></i>
            <span>Documentation</span>
          </Link>
          <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener noreferrer" className="nav-item">
            <i className="bi bi-box-arrow-up-right"></i>
            <span>MCP Protocol</span>
          </a>
        </div>
      </nav>

      {/* User section */}
      <div className="sidebar-user-section" style={{ 
        marginTop: 'auto',
        padding: '1rem', 
        borderTop: '1px solid var(--card-border)',
      }}>
        {user && (
          <div className="sidebar-user-info" style={{ 
            display: 'flex', 
            alignItems: 'center', 
            gap: '0.75rem',
            marginBottom: '0.75rem'
          }}>
            <div style={{
              width: '36px',
              height: '36px',
              flexShrink: 0,
              borderRadius: '50%',
              background: 'var(--primary-light)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--primary-color)',
              fontWeight: 600,
              fontSize: '0.875rem'
            }}>
              {user.name.charAt(0).toUpperCase()}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ 
                fontSize: '0.875rem', 
                fontWeight: 500, 
                color: 'var(--text-primary)',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis'
              }}>
                {user.name}
              </div>
              <div style={{ 
                fontSize: '0.75rem', 
                color: 'var(--text-muted)',
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis'
              }}>
                {user.email}
              </div>
            </div>
          </div>
        )}
        <button 
          onClick={handleLogout}
          className="sidebar-signout-btn"
          style={{
            width: '100%',
            padding: '0.5rem',
            background: 'transparent',
            border: '1px solid var(--card-border)',
            borderRadius: '6px',
            color: 'var(--text-secondary)',
            fontSize: '0.8125rem',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '0.5rem',
            transition: 'all 0.15s'
          }}
          onMouseOver={(e) => {
            e.currentTarget.style.background = 'var(--hover-bg)';
            e.currentTarget.style.color = 'var(--danger-color)';
            e.currentTarget.style.borderColor = 'var(--danger-color)';
          }}
          onMouseOut={(e) => {
            e.currentTarget.style.background = 'transparent';
            e.currentTarget.style.color = 'var(--text-secondary)';
            e.currentTarget.style.borderColor = 'var(--card-border)';
          }}
          aria-label="Sign out"
        >
          <i className="bi bi-box-arrow-left"></i>
          <span className="sidebar-signout-text">Sign Out</span>
        </button>
      </div>
    </aside>
  );
}
