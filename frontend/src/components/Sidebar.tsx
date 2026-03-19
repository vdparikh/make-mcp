import { useEffect, useRef, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [openMenu, setOpenMenu] = useState<'build' | 'resources' | 'user' | null>(null);
  const navRef = useRef<HTMLElement | null>(null);
  
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

  useEffect(() => {
    setOpenMenu(null);
  }, [location.pathname]);

  useEffect(() => {
    const onPointerDown = (event: MouseEvent) => {
      if (!navRef.current) return;
      if (navRef.current.contains(event.target as Node)) return;
      setOpenMenu(null);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpenMenu(null);
    };

    document.addEventListener('mousedown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, []);

  const toggleMenu = (menu: 'build' | 'resources' | 'user') => {
    setOpenMenu((prev) => (prev === menu ? null : menu));
  };

  return (
    <header className="top-nav" ref={navRef}>
      <div className="top-nav-inner">
        <Link to="/" className="top-nav-brand" aria-label="Go to dashboard">
          <div className="top-nav-logo-icon rounded-pill" style={{ backgroundColor: 'var(--primary-dark)' }}>
            <i className="bi bi-lightning-charge-fill" />
          </div>
          <span className="top-nav-brand-text">Make MCP</span>
        </Link>

        <nav className="top-nav-links">
          <Link to="/" className={`top-nav-link ${location.pathname === '/' ? 'active' : ''}`}>Dashboard</Link>
          <Link to="/marketplace" className={`top-nav-link ${isActive('/marketplace') ? 'active' : ''}`}>Marketplace</Link>
          <Link to="/observability" className={`top-nav-link ${isActive('/observability') ? 'active' : ''}`}>Observability</Link>
          <Link to="/hosted" className={`top-nav-link ${isActive('/hosted') ? 'active' : ''}`}>Hosted</Link>

          <div className={`top-nav-menu ${openMenu === 'build' ? 'open' : ''}`}>
            <button type="button" className="top-nav-link top-nav-menu-trigger" onClick={() => toggleMenu('build')}>
              Build
              <i className="bi bi-chevron-down" />
            </button>
            <div className="top-nav-dropdown top-nav-mega">
              <div className="top-nav-mega-col">
                <div className="top-nav-mega-title">Create</div>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/')}>
                  <i className="bi bi-plus-lg" />
                  <span>
                    <strong>New Server</strong>
                    <small>Start from scratch with tools, resources, and prompts.</small>
                  </span>
                </button>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/?tab=compositions')}>
                  <i className="bi bi-layers" />
                  <span>
                    <strong>New Composition</strong>
                    <small>Merge multiple servers into one hosted MCP endpoint.</small>
                  </span>
                </button>
              </div>
              <div className="top-nav-mega-col">
                <div className="top-nav-mega-title">Import & Build</div>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/deploy')}>
                  <i className="bi bi-rocket-takeoff" />
                  <span>
                    <strong>Deploy</strong>
                    <small>Open deployment flow and choose what to publish.</small>
                  </span>
                </button>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/import/openapi')}>
                  <i className="bi bi-file-earmark-code" />
                  <span>
                    <strong>Import OpenAPI</strong>
                    <small>Generate tools from an existing API specification.</small>
                  </span>
                </button>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/flow')}>
                  <i className="bi bi-diagram-3" />
                  <span>
                    <strong>Flow Builder</strong>
                    <small>Create orchestrated execution flows between tools.</small>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <div className={`top-nav-menu ${openMenu === 'resources' ? 'open' : ''}`}>
            <button type="button" className="top-nav-link top-nav-menu-trigger" onClick={() => toggleMenu('resources')}>
              Resources
              <i className="bi bi-chevron-down" />
            </button>
            <div className="top-nav-dropdown top-nav-mega top-nav-mega-resources">
              <div className="top-nav-mega-col">
                <div className="top-nav-mega-title">Project</div>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/docs')}>
                  <i className="bi bi-book" />
                  <span>
                    <strong>Documentation</strong>
                    <small>Guides, architecture, and getting started docs.</small>
                  </span>
                </button>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/observability')}>
                  <i className="bi bi-graph-up" />
                  <span>
                    <strong>Observability</strong>
                    <small>Runtime events, health checks, and session controls.</small>
                  </span>
                </button>
                <button type="button" className="top-nav-mega-item" onClick={() => navigate('/hosted')}>
                  <i className="bi bi-hdd-network" />
                  <span>
                    <strong>Hosted Catalog</strong>
                    <small>Browse running hosted servers and install them in one click.</small>
                  </span>
                </button>
              </div>
              <div className="top-nav-mega-col">
                <div className="top-nav-mega-title">External</div>
                <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener noreferrer" className="top-nav-mega-item">
                  <i className="bi bi-box-arrow-up-right" />
                  <span>
                    <strong>MCP Protocol</strong>
                    <small>Official Model Context Protocol reference docs.</small>
                  </span>
                </a>
                <a href="https://github.com/vdparikh/make-mcp" target="_blank" rel="noopener noreferrer" className="top-nav-mega-item">
                  <i className="bi bi-github" />
                  <span>
                    <strong>GitHub Repository</strong>
                    <small>Source code, issues, roadmap, and releases.</small>
                  </span>
                </a>
              </div>
            </div>
          </div>
        </nav>

        <div className={`top-nav-menu top-nav-user-menu ${openMenu === 'user' ? 'open' : ''}`}>
          <button type="button" className="top-nav-user-trigger" onClick={() => toggleMenu('user')} aria-label="User menu">
            <span className="top-nav-user-avatar">{user?.name?.charAt(0).toUpperCase() || 'U'}</span>
            <span className="top-nav-user-name">{user?.name || 'User'}</span>
            <i className="bi bi-chevron-down" />
          </button>
          <div className="top-nav-dropdown top-nav-user-dropdown">
            {user?.email && <div className="top-nav-user-email">{user.email}</div>}
            <button type="button" className="top-nav-dropdown-item" onClick={() => navigate('/hosted/keys')}>
              <i className="bi bi-key" />
              Caller API Keys
            </button>
            <button type="button" className="top-nav-dropdown-item danger" onClick={handleLogout}>
              <i className="bi bi-box-arrow-left" />
              Sign Out
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
