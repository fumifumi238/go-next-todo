"use client";

import React, { createContext, useState, ReactNode, useCallback } from "react";
import Cookies from "js-cookie";

type AuthContextType = {
  token: string | null;
  login: (newToken: string) => void;
  logout: () => void;
};

export const AuthContext = createContext<AuthContextType>({
  token: null,
  login: () => {},
  logout: () => {},
});

type AuthProviderProps = {
  children: ReactNode;
  initialToken: string | null;
};

export const AuthProvider: React.FC<AuthProviderProps> = ({
  children,
  initialToken,
}) => {
  // 初期値を lazy initializer で読み込む → これなら useEffect 不要
  const [token, setToken] = useState<string | null>(initialToken);
  const login = useCallback((newToken: string) => {
    setToken(newToken);
    Cookies.set("token", newToken, { expires: 7 });
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    Cookies.remove("token");
  }, []);

  return (
    <AuthContext.Provider value={{ token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
