import { useTryChat } from '../contexts/TryChatContext';
import TryChatModal from './TryChatModal';
import { useLocation } from 'react-router-dom';

export default function TryChatLauncher() {
  const { openTryChat } = useTryChat();
  const location = useLocation();
  const serverMatch = location.pathname.match(/^\/servers\/([^/]+)$/);
  const routeTarget = serverMatch
    ? { type: 'server', id: serverMatch[1], name: 'Server' }
    : undefined;

  return (
    <>
      <button
        type="button"
        className="btn btn-primary"
        onClick={() => openTryChat(routeTarget)}
        style={{
          position: 'fixed',
          right: '1.25rem',
          bottom: '1.25rem',
          zIndex: 1500,
          borderRadius: '9999px',
          boxShadow: '0 10px 22px rgba(0,0,0,0.22)',
          padding: '0.7rem 1rem',
        }}
      >
        <i className="bi bi-stars" style={{ marginRight: '0.4rem' }} />
        Try Chat
      </button>
      <TryChatModal />
    </>
  );
}

