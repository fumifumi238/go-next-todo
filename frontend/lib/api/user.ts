import { User, LoginFormInputs } from "@/app/types/user";
import Cookies from "js-cookie";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/**
 * すべてのユーザーを取得する（管理者専用）
 * @returns ユーザー一覧
 */
export const fetchAllUsers = async (): Promise<User[]> => {
  const token = Cookies.get("token");
  if (!token) {
    throw new Error("認証トークンが見つかりません");
  }

  try {
    const response = await fetch(`${API_BASE_URL}/api/admin/users`, {
      method: "GET",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      if (response.status === 403) {
        throw new Error("管理者権限が必要です");
      }
      throw new Error("ユーザー一覧の取得に失敗しました");
    }

    return response.json();
  } catch (error) {
    if (error instanceof TypeError && error.message.includes("fetch")) {
      throw new Error(
        "ネットワークエラーによりユーザー一覧の取得に失敗しました"
      );
    }
    throw error;
  }
};

/**
 * 管理者ログインステップ1: パスワード認証 + OTP送信
 * @param data ログイン情報
 * @returns user_idとrole
 */
export const adminLoginStep1 = async (data: LoginFormInputs): Promise<{ user_id: number; role: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/admin/login/step1`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "管理者ログインに失敗しました");
    }

    return response.json();
  } catch (error) {
    if (error instanceof TypeError && error.message.includes("fetch")) {
      throw new Error("ネットワークエラーにより管理者ログインに失敗しました");
    }
    throw error;
  }
};

/**
 * 管理者ログインステップ2: OTP検証 + JWT発行
 * @param userId ユーザーID
 * @param otp OTPコード
 * @returns トークン情報
 */
export const adminLoginStep2 = async (userId: number, otp: string): Promise<{ token: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/admin/login/step2`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ user_id: userId, otp }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "OTP検証に失敗しました");
    }

    return response.json();
  } catch (error) {
    if (error instanceof TypeError && error.message.includes("fetch")) {
      throw new Error("ネットワークエラーによりOTP検証に失敗しました");
    }
    throw error;
  }
};
