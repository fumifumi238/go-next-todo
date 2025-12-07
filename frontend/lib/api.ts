import { z } from "zod";
import { RegisterFormInputs, LoginFormInputs } from "@/app/types/user";
import { todoSchema } from "@/app/types/todo";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

interface ApiResponse<T> {
  data?: T;
  error?: string;
}

// Zod schemas for API responses
const registerResponseSchema = z.object({
  id: z.number(),
  username: z.string(),
  email: z.string(),
});

const loginResponseSchema = z.object({
  token: z.string(),
  user_id: z.number(),
  role: z.string(),
});

// 使われていません
const todoResponseSchema = z.array(todoSchema);

const errorResponseSchema = z.object({
  error: z.string(),
});

export const registerUser = async (
  userData: RegisterFormInputs
): Promise<ApiResponse<z.infer<typeof registerResponseSchema>>> => {
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
      const errorParsed = errorResponseSchema.safeParse(data);
      return {
        error: errorParsed.success
          ? errorParsed.data.error
          : "ユーザー登録に失敗しました",
      };
    }

    const parsed = registerResponseSchema.safeParse(data.data);
    if (!parsed.success) {
      console.error("Response validation failed:", parsed.error);
      return { error: "レスポンス形式が無効です" };
    }

    return { data: parsed.data };
  } catch (error) {
    console.error("Registration API error:", error);
    return { error: "ネットワークエラーによりユーザー登録に失敗しました" };
  }
};

export const loginUser = async (
  credentials: LoginFormInputs
): Promise<ApiResponse<z.infer<typeof loginResponseSchema>>> => {
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
      const errorParsed = errorResponseSchema.safeParse(data);
      return {
        error: errorParsed.success
          ? errorParsed.data.error
          : "ログインに失敗しました",
      };
    }

    const parsed = loginResponseSchema.safeParse(data.data);
    if (!parsed.success) {
      console.error("Response validation failed:", parsed.error);
      return { error: "レスポンス形式が無効です" };
    }

    return { data: parsed.data };
  } catch (error) {
    console.error("Login API error:", error);
    return { error: "ネットワークエラーによりログインに失敗しました" };
  }
};
