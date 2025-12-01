import { RegisterFormInputs } from "@/app/types/user";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080/api";

interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export const registerUser = async (
  userData: RegisterFormInputs
): Promise<ApiResponse<{ message: string; user_id: number }>> => {
  try {
    const response = await fetch(`${API_BASE_URL}/register`, {
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

// 将来的にログインAPIもここに追加します
// export const loginUser = async (credentials: LoginFormInputs): Promise<ApiResponse<{ token: string }>> => { /* ... */ };
