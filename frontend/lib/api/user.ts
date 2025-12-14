import { User } from "@/app/types/user";
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
