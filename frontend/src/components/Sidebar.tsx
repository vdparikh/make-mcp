import { useEffect, useState } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [openMenu, setOpenMenu] = useState<'build' | 'resources' | 'user' | null>(null);
  
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

  const toggleMenu = (menu: 'build' | 'resources' | 'user') => {
    setOpenMenu((prev) => (prev === menu ? null : menu));
  };

  return (
    <header className="top-nav">
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

          <div className={`top-nav-menu ${openMenu === 'build' ? 'open' : ''}`}>
            <button type="button" className="top-nav-link top-nav-menu-trigger" onClick={() => toggleMenu('build')}>
              Build
              <i className="bi bi-chevron-down" />
            </button>
            <div className="top-nav-dropdown">
              <button type="button" className="top-nav-dropdown-item" onClick={() => navigate('/')}>
                <i className="bi bi-plus-lg" />
                New Server
              </button>
              <button type="button" className="top-nav-dropdown-item" onClick={() => navigate('/?tab=compositions')}>
                <i className="bi bi-layers" />
                Compositions
              </button>
              <button type="button" className="top-nav-dropdown-item" onClick={() => navigate('/import/openapi')}>
                <i className="bi bi-file-earmark-code" />
                Import OpenAPI
              </button>
            </div>
          </div>

          <div className={`top-nav-menu ${openMenu === 'resources' ? 'open' : ''}`}>
            <button type="button" className="top-nav-link top-nav-menu-trigger" onClick={() => toggleMenu('resources')}>
              Resources
              <i className="bi bi-chevron-down" />
            </button>
            <div className="top-nav-dropdown">
              <button type="button" className="top-nav-dropdown-item" onClick={() => navigate('/docs')}>
                <i className="bi bi-book" />
                Documentation
              </button>
              <a href="https://modelcontextprotocol.io" target="_blank" rel="noopener noreferrer" className="top-nav-dropdown-item">
                <i className="bi bi-box-arrow-up-right" />
                MCP Protocol
              </a>
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
