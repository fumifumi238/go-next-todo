import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import AdminUsersPage from "../page";
import { fetchAllUsers } from "@/lib/api/user";
import { User } from "@/app/types/user";

// Next.jsのuseRouterとfetchAllUsersをモック化
const mockPush = jest.fn();
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

jest.mock("@/lib/api/user", () => ({
  fetchAllUsers: jest.fn(),
}));

const mockedFetchAllUsers = fetchAllUsers as jest.MockedFunction<typeof fetchAllUsers>;

describe("AdminUsersPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("ユーザー一覧を正常に表示する", async () => {
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

    mockedFetchAllUsers.mockResolvedValue(mockUsers);

    render(<AdminUsersPage />);

    // ローディング中
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();

    // データ取得完了後
    await waitFor(() => {
      expect(screen.getByText("ユーザー管理")).toBeInTheDocument();
    });

    // テーブルヘッダー
    expect(screen.getByText("ID")).toBeInTheDocument();
    expect(screen.getByText("ユーザー名")).toBeInTheDocument();
    expect(screen.getByText("メールアドレス")).toBeInTheDocument();
    expect(screen.getByText("ロール")).toBeInTheDocument();

    // ユーザー情報
    expect(screen.getByText("admin")).toBeInTheDocument();
    expect(screen.getByText("admin@example.com")).toBeInTheDocument();
    expect(screen.getByText("管理者")).toBeInTheDocument();
    expect(screen.getByText("user")).toBeInTheDocument();
    expect(screen.getByText("user@example.com")).toBeInTheDocument();
    expect(screen.getByText("ユーザー")).toBeInTheDocument();

    // APIが呼ばれたことを確認
    expect(mockedFetchAllUsers).toHaveBeenCalledTimes(1);
  });

  it("APIエラーの場合、エラーメッセージを表示する", async () => {
    mockedFetchAllUsers.mockRejectedValue(new Error("管理者権限が必要です"));

    render(<AdminUsersPage />);

    await waitFor(() => {
      expect(screen.getByText("エラー: 管理者権限が必要です")).toBeInTheDocument();
    });

    // ホームに戻るボタン
    const backButton = screen.getByText("ホームに戻る");
    expect(backButton).toBeInTheDocument();

    // ボタンクリックでルーターが呼ばれる
    await userEvent.click(backButton);
    expect(mockPush).toHaveBeenCalledWith("/");
  });

  it("ユーザーが空の場合、適切なメッセージを表示する", async () => {
    mockedFetchAllUsers.mockResolvedValue([]);

    render(<AdminUsersPage />);

    await waitFor(() => {
      expect(screen.getByText("ユーザーが見つかりません")).toBeInTheDocument();
    });
  });

  it("戻るボタンをクリックするとホームに戻る", async () => {
    const mockUsers: User[] = [
      {
        id: 1,
        username: "admin",
        email: "admin@example.com",
        role: "admin",
        created_at: "2023-01-01T00:00:00Z",
        updated_at: "2023-01-01T00:00:00Z",
      },
    ];

    mockedFetchAllUsers.mockResolvedValue(mockUsers);

    render(<AdminUsersPage />);

    await waitFor(() => {
      expect(screen.getByText("ユーザー管理")).toBeInTheDocument();
    });

    const backButton = screen.getByText("戻る");
    await userEvent.click(backButton);

    expect(mockPush).toHaveBeenCalledWith("/");
  });
});
