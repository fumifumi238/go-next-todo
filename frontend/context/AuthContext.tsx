// frontend/context/AuthContext.tsx
"use client";

import React, {
  createContext,
  useState,
  useContext,
  useEffect,
  ReactNode,
} from "react";
import Cookies from "js-cookie"; // クッキーを扱うためにインストールします

interface AuthContextType {
  token: string | null;
  userId: number | null;
  role: string | null;
  login: (jwtToken: string, id: number, userRole: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState<string | null>(null);
  const [userId, setUserId] = useState<number | null>(null);
  const [role, setRole] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    // コンポーネントマウント時にクッキーからトークンを読み込む
    const storedToken = Cookies.get("jwt_token");
    const storedUserId = Cookies.get("user_id");
    const storedRole = Cookies.get("user_role");

    if (storedToken && storedUserId && storedRole) {
      setToken(storedToken);
      setUserId(parseInt(storedUserId, 10));
      setRole(storedRole);
      setIsAuthenticated(true);
    }
  }, []);

  const login = (jwtToken: string, id: number, userRole: string) => {
    Cookies.set("jwt_token", jwtToken, { expires: 7 }); // 7日間有効なクッキー
    Cookies.set("user_id", id.toString(), { expires: 7 });
    Cookies.set("user_role", userRole, { expires: 7 });
    setToken(jwtToken);
    setUserId(id);
    setRole(userRole);
    setIsAuthenticated(true);
  };

  const logout = () => {
    Cookies.remove("jwt_token");
    Cookies.remove("user_id");
    Cookies.remove("user_role");
    setToken(null);
    setUserId(null);
    setRole(null);
    setIsAuthenticated(false);
  };

  return (
    <AuthContext.Provider
      value={{ token, userId, role, login, logout, isAuthenticated }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};
