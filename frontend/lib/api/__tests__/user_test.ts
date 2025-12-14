import { fetchAllUsers } from "@/lib/api/user";
import { User } from "@/app/types/user";

// fetchとCookiesをモック化
global.fetch = jest.fn();
jest.mock("js-cookie", () => ({
  get: jest.fn(),
}));

import Cookies from "js-cookie";

const mockedCookies = Cookies as jest.Mocked<typeof Cookies>;
const mockedGet = mockedCookies.get as jest.Mock;

describe("API Functions - User", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("fetchAllUsers", () => {
    it("管理者ユーザーがユーザー一覧を取得できる", async () => {
      const mockUsers: User[] = [
        {
          id: 1,
          username: "admin",
          email: "admin@example.com",
          role: "admin",
          created_at: "2023-01-01T00:00:00Z",
          updated_at: "2023-01-01T00:00:00Z",
        },
        {
          id: 2,
          username: "user",
          email: "user@example.com",
          role: "user",
          created_at: "2023-01-02T00:00:00Z",
          updated_at: "2023-01-02T00:00:00Z",
        },
      ];

      mockedGet.mockReturnValue("mock-token");

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockUsers,
      });

      const result = await fetchAllUsers();

      expect(mockedGet).toHaveBeenCalledWith("token");
      expect(fetch).toHaveBeenCalledWith(
        "http://localhost:8080/api/admin/users",
        {
          method: "GET",
          headers: {
            Authorization: "Bearer mock-token",
            "Content-Type": "application/json",
          },
        }
      );
      expect(result).toEqual(mockUsers);
    });

    it("トークンがない場合、エラーを投げる", async () => {
      mockedGet.mockReturnValue(undefined);

      await expect(fetchAllUsers()).rejects.toThrow(
        "認証トークンが見つかりません"
      );
    });

    it("管理者権限がない場合、エラーを投げる", async () => {
      mockedGet.mockReturnValue("mock-token");

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 403,
        json: async () => ({ error: "管理者権限が必要です" }),
      });

      await expect(fetchAllUsers()).rejects.toThrow("管理者権限が必要です");
    });

    it("サーバーエラーの場合、エラーを投げる", async () => {
      mockedGet.mockReturnValue("mock-token");

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: "サーバーエラー" }),
      });

      await expect(fetchAllUsers()).rejects.toThrow(
        "ユーザー一覧の取得に失敗しました"
      );
    });

    it("ネットワークエラーの場合、エラーを投げる", async () => {
      mockedGet.mockReturnValue("mock-token");

      (fetch as jest.Mock).mockRejectedValueOnce(
        new TypeError("Failed to fetch")
      );

      await expect(fetchAllUsers()).rejects.toThrow(
        "ネットワークエラーによりユーザー一覧の取得に失敗しました"
      );
    });
  });
});
