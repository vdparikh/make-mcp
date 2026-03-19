import { Routes, Route, Navigate } from 'react-router-dom';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import Sidebar from './components/Sidebar';
import Dashboard from './pages/Dashboard';
import ServerEditor from './pages/ServerEditor';
import ImportOpenAPI from './pages/ImportOpenAPI';
import FlowBuilderPage from './pages/FlowBuilderPage';
import Marketplace from './pages/Marketplace';
import Observability from './pages/Observability';
import Docs from './pages/Docs';
import DeployFlowPage from './pages/DeployFlowPage';
import HostedCatalogPage from './pages/HostedCatalogPage';
import HostedCallerKeysPage from './pages/HostedCallerKeysPage';
import Login from './pages/Login';
import Register from './pages/Register';
import { TryChatProvider } from './contexts/TryChatContext';
import TryChatModal from './components/TryChatModal';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div style={{ 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'center', 
        minHeight: '100vh',
        background: 'var(--dark-bg)'
      }}>
        <div className="spinner"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function AppLayout() {
  return (
    <div className="app-container">
      <Sidebar />
      <main className="main-content">
        <div className="main-content-container">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/servers/:id" element={<ServerEditor />} />
            <Route path="/servers/:id/flow" element={<FlowBuilderPage />} />
            <Route path="/flow" element={<FlowBuilderPage />} />
            <Route path="/compositions" element={<Navigate to="/?tab=compositions" replace />} />
            <Route path="/marketplace" element={<Marketplace />} />
            <Route path="/observability" element={<Observability />} />
            <Route path="/hosted" element={<HostedCatalogPage />} />
            <Route path="/hosted/keys" element={<HostedCallerKeysPage />} />
            <Route path="/import/openapi" element={<ImportOpenAPI />} />
            <Route path="/docs" element={<Docs />} />
            <Route path="/deploy" element={<DeployFlowPage />} />
          </Routes>
        </div>
      </main>
      <TryChatModal />
    </div>
  );
}

function App() {
  return (
    <AuthProvider>
      <TryChatProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <AppLayout />
              </ProtectedRoute>
            }
          />
        </Routes>
        <ToastContainer
          position="bottom-right"
          autoClose={3000}
          hideProgressBar={false}
          newestOnTop
          closeOnClick
          rtl={false}
          pauseOnFocusLoss
          draggable
          pauseOnHover
          theme="dark"
        />
      </TryChatProvider>
    </AuthProvider>
  );
}

export default App;
