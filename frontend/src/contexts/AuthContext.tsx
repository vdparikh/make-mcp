import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import {
  getCurrentUser,
  registerAccount,
  webauthnRegisterBegin,
  webauthnRegisterFinish,
  webauthnLoginBegin,
  webauthnLoginFinish,
} from '../services/api';
import {
  toCreationOptions,
  toRequestOptions,
  credentialCreationResponseToJSON,
  credentialAssertionResponseToJSON,
} from '../utils/webauthn';

interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string) => Promise<void>;
  register: (email: string, name: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const initAuth = async () => {
      const storedToken = localStorage.getItem('token');
      if (storedToken) {
        try {
          const userData = await getCurrentUser();
          setUser(userData);
          setToken(storedToken);
        } catch {
          localStorage.removeItem('token');
          setToken(null);
        }
      }
      setIsLoading(false);
    };

    initAuth();
  }, []);

  const login = async (email: string) => {
    const { session_id, options } = await webauthnLoginBegin(email);
    const requestOptions = toRequestOptions(options.publicKey) as PublicKeyCredentialRequestOptions;
    const cred = await navigator.credentials.get({ publicKey: requestOptions });
    if (!cred || !(cred instanceof PublicKeyCredential)) {
      throw new Error('Passkey sign-in was not completed');
    }
    const responseJson = credentialAssertionResponseToJSON(cred);
    const response = await webauthnLoginFinish(session_id, responseJson);
    localStorage.setItem('token', response.token);
    setToken(response.token);
    setUser(response.user);
  };

  const register = async (email: string, name: string) => {
    await registerAccount(email, name);
    const { session_id, options } = await webauthnRegisterBegin(email);
    const creationOptions = toCreationOptions(options.publicKey) as PublicKeyCredentialCreationOptions;
    const cred = await navigator.credentials.create({ publicKey: creationOptions });
    if (!cred || !(cred instanceof PublicKeyCredential)) {
      throw new Error('Passkey creation was not completed');
    }
    const responseJson = credentialCreationResponseToJSON(cred);
    const response = await webauthnRegisterFinish(session_id, responseJson);
    localStorage.setItem('token', response.token);
    setToken(response.token);
    setUser(response.user);
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isAuthenticated: !!token && !!user,
        isLoading,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
