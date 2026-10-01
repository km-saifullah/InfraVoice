import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { AuthApi } from "../lib/endpoints";
import { configureAuthHook } from "../lib/apiClient";

const REFRESH_TOKEN_KEY = "infravoice.refresh_token";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [accessToken, setAccessToken] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  const accessTokenRef = useRef(null);
  const refreshTokenRef = useRef(localStorage.getItem(REFRESH_TOKEN_KEY));

  const applySession = useCallback((session) => {
    accessTokenRef.current = session.access_token;
    refreshTokenRef.current = session.refresh_token;
    localStorage.setItem(REFRESH_TOKEN_KEY, session.refresh_token);
    setAccessToken(session.access_token);
    setUser(session.user);
  }, []);

  const clearSession = useCallback(() => {
    accessTokenRef.current = null;
    refreshTokenRef.current = null;
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    setAccessToken(null);
    setUser(null);
  }, []);

  const refreshAccessToken = useCallback(async () => {
    const currentRefreshToken = refreshTokenRef.current;

    if (!currentRefreshToken) {
      return null;
    }

    try {
      const session = await AuthApi.refresh(currentRefreshToken);
      applySession(session);
      return session.access_token;
    } catch {
      clearSession();
      return null;
    }
  }, [applySession, clearSession]);

  useEffect(() => {
    configureAuthHook({
      getAccessToken: () => accessTokenRef.current,
      refreshAccessToken,
      onAuthExpired: clearSession,
    });
  }, [refreshAccessToken, clearSession]);

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      if (refreshTokenRef.current) {
        await refreshAccessToken();
      }

      if (!cancelled) {
        setIsLoading(false);
      }
    }

    bootstrap();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const login = useCallback(
    async (email, password) => {
      const session = await AuthApi.login(email, password);
      applySession(session);
      return session.user;
    },
    [applySession],
  );

  const register = useCallback(
    async (name, email, password) => {
      await AuthApi.register(name, email, password);
      return login(email, password);
    },
    [login],
  );

  const logout = useCallback(async () => {
    const currentRefreshToken = refreshTokenRef.current;

    try {
      if (currentRefreshToken) {
        await AuthApi.logout(currentRefreshToken);
      }
    } finally {
      clearSession();
    }
  }, [clearSession]);

  const value = {
    user,
    accessToken,
    isAuthenticated: Boolean(accessToken && user),
    isLoading,
    login,
    register,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }

  return context;
}
