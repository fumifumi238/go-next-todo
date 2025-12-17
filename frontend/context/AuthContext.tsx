"use client";

import React, { createContext, useState, ReactNode, useCallback } from "react";
import Cookies from "js-cookie";

type AuthContextType = {
  token: string | null;
  role: string | null;
  login: (newToken: string, userRole: string) => void;
  logout: () => void;
};

export const AuthContext = createContext<AuthContextType>({
  token: null,
  role: null,
  login: () => {},
  logout: () => {},
});

type AuthProviderProps = {
  children: ReactNode;
  initialToken: string | null;
  initialRole: string | null;
};

export const AuthProvider: React.FC<AuthProviderProps> = ({
  children,
  initialToken,
  initialRole,
}) => {
  // 初期値を lazy initializer で読み込む
  const [token, setToken] = useState<string | null>(initialToken);
  const [role, setRole] = useState<string | null>(initialRole);
  const login = useCallback((newToken: string, userRole: string) => {
    setToken(newToken);
    setRole(userRole);
    Cookies.set("token", newToken, { expires: 7 });
    localStorage.setItem("role", userRole);
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    setRole(null);
    Cookies.remove("token");
    localStorage.removeItem("role");
  }, []);

  return (
    <AuthContext.Provider value={{ token, role, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
