import { RegisterFormInputs, LoginFormInputs } from "@/app/types/user";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export const registerUser = async (
  userData: RegisterFormInputs
): Promise<ApiResponse<{ message: string; user_id: number }>> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/register`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(userData),
    });

    const data = await response.json();

    if (!response.ok) {
      // バックエンドからのエラーメッセージを返す
      return {
        error: data.error || data.message || "ユーザー登録に失敗しました",
      };
    }

    return { data: data };
  } catch (error) {
    console.error("Registration API error:", error);
    return { error: "ネットワークエラーによりユーザー登録に失敗しました" };
  }
};

export const loginUser = async (
  credentials: LoginFormInputs
): Promise<ApiResponse<{ token: string; user_id: number; role: string }>> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(credentials),
    });

    const data = await response.json();

    if (!response.ok) {
      return {
        error: data.error || data.message || "ログインに失敗しました",
      };
    }

    return {
      data: { token: data.token, user_id: data.user_id, role: data.role },
    };
  } catch (error) {
    console.error("Login API error:", error);
    return { error: "ネットワークエラーによりログインに失敗しました" };
  }
};
