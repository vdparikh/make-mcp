import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeSlug from 'rehype-slug';
import axios from 'axios';

interface DocItem {
  id: string;
  title: string;
  description: string;
  icon: string;
}

const docs: DocItem[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    description: 'Quick start guide for the MCP Server Builder platform',
    icon: 'bi-rocket-takeoff',
  },
  {
    id: 'creating-servers',
    title: 'Creating Servers',
    description: 'Complete guide to building MCP servers with tools, resources, and prompts',
    icon: 'bi-server',
  },
  {
    id: 'compositions',
    title: 'Server Compositions',
    description: 'How to combine multiple MCP servers into one unified interface',
    icon: 'bi-layers',
  },
  {
    id: 'security-best-practices',
    title: 'Security Best Practices',
    description: 'MCP security practices, in-app security score (SlowMist checklist), and how Make MCP supports them',
    icon: 'bi-shield-lock',
  },
];

const DOC_IDS = new Set(docs.map((d) => d.id));

export default function Docs() {
  const [selectedDoc, setSelectedDoc] = useState<string | null>(null);
  const [content, setContent] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedDoc) {
      setContent('');
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    axios
      .get<{ content: string }>(`/api/docs/${selectedDoc}`)
      .then((res) => {
        if (!cancelled) setContent(res.data.content ?? '');
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err.response?.status === 404 ? 'Document not found.' : 'Failed to load document.');
          setContent('');
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedDoc]);

  const isInternalDocLink = (href: string) => {
    if (!href || href.startsWith('#')) return false;
    const withoutExt = href.replace(/\.md$/, '').replace(/^\.\//, '').split('#')[0];
    return DOC_IDS.has(withoutExt);
  };

  const navigateToDoc = (href: string) => {
    if (!href || href.startsWith('#')) return;
    const withoutExt = href.replace(/\.md$/, '').replace(/^\.\//, '').split('#')[0];
    if (DOC_IDS.has(withoutExt)) setSelectedDoc(withoutExt);
  };

  if (selectedDoc) {
    const docMeta = docs.find((d) => d.id === selectedDoc);
    return (
      <div>
        <div className="page-header">
          <div>
            <nav style={{ marginBottom: '0.5rem' }}>
              <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
                Dashboard
              </Link>
              <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
              <button
                type="button"
                onClick={() => setSelectedDoc(null)}
                style={{
                  color: 'var(--text-muted)',
                  textDecoration: 'none',
                  fontSize: '0.875rem',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                }}
              >
                Documentation
              </button>
              <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
              <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>{docMeta?.title}</span>
            </nav>
            <h1 className="page-title">Documentation</h1>
          </div>
          <button type="button" className="btn btn-secondary" onClick={() => setSelectedDoc(null)}>
            <i className="bi bi-arrow-left"></i>
            Back to Docs
          </button>
        </div>

        <div className="card">
          <div
            className="doc-content"
            style={{
              lineHeight: 1.7,
              color: 'var(--text-primary)',
            }}
          >
            {loading && (
              <p style={{ color: 'var(--text-muted)' }}>
                <span className="spinner" style={{ width: 20, height: 20, borderWidth: 2, marginRight: '0.5rem' }} />
                Loading…
              </p>
            )}
            {error && (
              <p style={{ color: 'var(--danger-color)' }}>{error}</p>
            )}
            {!loading && !error && content && (
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeSlug]}
                components={{
                  a: ({ href, children, ...props }) => {
                    const linkHref = href ?? '';
                    if (linkHref && isInternalDocLink(linkHref)) {
                      return (
                        <button
                          type="button"
                          style={{
                            background: 'none',
                            border: 'none',
                            padding: 0,
                            color: 'var(--primary-color)',
                            cursor: 'pointer',
                            textDecoration: 'none',
                            font: 'inherit',
                          }}
                          onClick={() => navigateToDoc(linkHref)}
                        >
                          {children}
                        </button>
                      );
                    }
                    const isExternal = linkHref.startsWith('http://') || linkHref.startsWith('https://');
                    return (
                      <a
                        href={linkHref}
                        {...(isExternal ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
                        {...props}
                      >
                        {children}
                      </a>
                    );
                  },
                }}
              >
                {content}
              </ReactMarkdown>
            )}
          </div>
        </div>

        <style>{`
          .doc-content h1 { font-size: 1.75rem; margin: 0 0 1rem 0; color: var(--text-primary); }
          .doc-content h2 { font-size: 1.375rem; margin: 1.5rem 0 0.75rem 0; color: var(--text-primary); border-bottom: 1px solid var(--card-border); padding-bottom: 0.5rem; }
          .doc-content h3 { font-size: 1.125rem; margin: 1.25rem 0 0.5rem 0; color: var(--text-primary); }
          .doc-content h4 { font-size: 1rem; margin: 1rem 0 0.5rem 0; color: var(--text-primary); }
          .doc-content p { margin: 0.75rem 0; }
          .doc-content code { background: #1a1a2e; color: #e5e7eb; padding: 0.125rem 0.375rem; border-radius: 4px; font-size: 0.875rem; }
          .doc-content pre { background: #1a1a2e; padding: 1rem; border-radius: 8px; overflow-x: auto; margin: 1rem 0; }
          .doc-content pre code { background: none; padding: 0; color: #a5f3fc; }
          .doc-content table { width: 100%; border-collapse: collapse; margin: 1rem 0; border: 1px solid var(--card-border); border-radius: 8px; overflow: hidden; }
          .doc-content th { padding: 0.75rem 1rem; text-align: left; background: var(--hover-bg); font-weight: 600; color: var(--text-primary); border-bottom: 2px solid var(--card-border); }
          .doc-content td { padding: 0.75rem 1rem; border-bottom: 1px solid var(--card-border); color: var(--text-secondary); }
          .doc-content tbody tr:last-child td { border-bottom: none; }
          .doc-content tbody tr:hover { background: var(--hover-bg); }
          .doc-content a { color: var(--primary-color); text-decoration: none; }
          .doc-content a:hover { text-decoration: underline; }
          .doc-content strong { color: var(--text-primary); }
          .doc-content ul, .doc-content ol { margin: 0.75rem 0; padding-left: 1.5rem; }
          .doc-content li { margin: 0.25rem 0; }
          .doc-content blockquote { border-left: 4px solid var(--card-border); margin: 1rem 0; padding-left: 1rem; color: var(--text-secondary); }
        `}</style>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <nav style={{ marginBottom: '0.5rem' }}>
            <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
              Dashboard
            </Link>
            <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>Documentation</span>
          </nav>
          <h1 className="page-title">Documentation</h1>
          <p className="page-subtitle">Learn how to build and deploy MCP servers</p>
        </div>
        <Link to="/" className="btn btn-secondary">
          <i className="bi bi-arrow-left"></i>
          Back
        </Link>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '1rem' }}>
        {docs.map((doc) => (
          <div
            key={doc.id}
            className="card"
            style={{ cursor: 'pointer', transition: 'all 0.2s' }}
            onClick={() => setSelectedDoc(doc.id)}
            onMouseOver={(e) => {
              e.currentTarget.style.borderColor = 'var(--primary-color)';
              e.currentTarget.style.transform = 'translateY(-2px)';
            }}
            onMouseOut={(e) => {
              e.currentTarget.style.borderColor = 'var(--card-border)';
              e.currentTarget.style.transform = 'translateY(0)';
            }}
          >
            <div style={{ fontSize: '2.5rem', marginBottom: '1rem', color: 'var(--primary-color)' }}>
              <i className={`bi ${doc.icon}`}></i>
            </div>
            <h3 style={{ marginBottom: '0.5rem', color: 'var(--text-primary)' }}>{doc.title}</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', margin: 0 }}>
              {doc.description}
            </p>
          </div>
        ))}
      </div>

      <div className="card" style={{ marginTop: '2rem' }}>
        <h3 className="card-title" style={{ marginBottom: '1rem' }}>
          <i className="bi bi-link-45deg" style={{ marginRight: '0.5rem' }}></i>
          External Resources
        </h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
          <a
            href="https://modelcontextprotocol.io"
            target="_blank"
            rel="noopener noreferrer"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '0.75rem',
              padding: '1rem',
              background: 'var(--hover-bg)',
              borderRadius: '8px',
              textDecoration: 'none',
              color: 'var(--text-primary)',
              transition: 'all 0.2s',
            }}
          >
            <i className="bi bi-book" style={{ fontSize: '1.5rem', color: 'var(--primary-color)' }}></i>
            <div>
              <div style={{ fontWeight: 500 }}>MCP Documentation</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Official protocol docs</div>
            </div>
            <i className="bi bi-box-arrow-up-right" style={{ marginLeft: 'auto', color: 'var(--text-muted)' }}></i>
          </a>

          <a
            href="https://github.com/modelcontextprotocol"
            target="_blank"
            rel="noopener noreferrer"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '0.75rem',
              padding: '1rem',
              background: 'var(--hover-bg)',
              borderRadius: '8px',
              textDecoration: 'none',
              color: 'var(--text-primary)',
              transition: 'all 0.2s',
            }}
          >
            <i className="bi bi-github" style={{ fontSize: '1.5rem' }}></i>
            <div>
              <div style={{ fontWeight: 500 }}>MCP GitHub</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>SDKs and examples</div>
            </div>
            <i className="bi bi-box-arrow-up-right" style={{ marginLeft: 'auto', color: 'var(--text-muted)' }}></i>
          </a>
        </div>
      </div>
    </div>
  );
}
