import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import RegisterForm from "@/components/RegisterForm/RegisterForm";
import * as api from "@/lib/api"; // apiモジュールをモックするためにインポート
import { useRouter } from "next/navigation";
import { AuthContext } from "@/context/AuthContext";

// next/navigation をモック
jest.mock("next/navigation", () => ({
  useRouter: jest.fn(),
}));

// apiモジュール全体をモック
jest.mock("@/lib/api", () => ({
  registerUser: jest.fn(),
  loginUser: jest.fn(),
}));

describe("RegisterForm", () => {
  const mockLogin = jest.fn();
  const mockPush = jest.fn();

  // AuthContext Providerのヘルパーコンポーネント
  const renderWithAuthContext = (ui: React.ReactElement) => {
    return render(
      <AuthContext.Provider
        value={{ token: null, login: mockLogin, logout: jest.fn() }}>
        {ui}
      </AuthContext.Provider>
    );
  };

  // 各テストの前にモックをリセット
  beforeEach(() => {
    jest.spyOn(window, "alert").mockImplementation(() => {});
    jest.clearAllMocks(); // 追加: すべてのモックの呼び出し履歴をクリア
    (useRouter as jest.Mock).mockReturnValue({
      push: mockPush,
      replace: jest.fn(),
      refresh: jest.fn(),
      back: jest.fn(),
      forward: jest.fn(),
    });
  });

  afterEach(() => {
    jest.restoreAllMocks(); // 各テスト後にモックをクリーンアップ
  });

  it("フォームが正しくレンダリングされること", () => {
    renderWithAuthContext(<RegisterForm />);
    expect(screen.getByLabelText(/ユーザー名/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/メールアドレス/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/パスワード/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /登録/i })).toBeInTheDocument();
  });

  it("すべてのフィールドが入力されたときに登録に成功すること", async () => {
    const mockRegisterUser = api.registerUser as jest.Mock;
    const mockLoginUser = api.loginUser as jest.Mock;
    mockRegisterUser.mockResolvedValueOnce({
      data: { message: "ユーザー登録が成功しました！", user_id: 1 },
    });
    mockLoginUser.mockResolvedValueOnce({
      data: { token: "fake-jwt-token", user_id: 1, role: "user" },
    });

    renderWithAuthContext(<RegisterForm />);

    await userEvent.type(screen.getByLabelText(/ユーザー名/i), "testuser");
    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "test@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    await waitFor(() => {
      expect(mockRegisterUser).toHaveBeenCalledWith({
        username: "testuser",
        email: "test@example.com",
        password: "password123",
      });
      expect(mockLoginUser).toHaveBeenCalledWith({
        email: "test@example.com",
        password: "password123",
      });
      expect(mockLogin).toHaveBeenCalledWith("fake-jwt-token");
      expect(window.alert).toHaveBeenCalledWith(
        "ユーザー登録が成功しました！自動的にログインします。"
      );
      expect(mockPush).toHaveBeenCalledWith("/");
    });

    // フォームがリセットされていること
    expect(screen.getByLabelText(/ユーザー名/i)).toHaveValue("");
    expect(screen.getByLabelText(/メールアドレス/i)).toHaveValue("");
    expect(screen.getByLabelText(/パスワード/i)).toHaveValue("");
  });

  it("必須フィールドが空の場合、バリデーションエラーを表示しボタンが無効になること", async () => {
    renderWithAuthContext(<RegisterForm />);

    const usernameInput = screen.getByLabelText(/ユーザー名/i);
    const emailInput = screen.getByLabelText(/メールアドレス/i);
    const passwordInput = screen.getByLabelText(/パスワード/i);
    const registerButton = screen.getByRole("button", { name: /登録/i });

    // 各フィールドを明示的にクリア
    await userEvent.clear(usernameInput);
    await userEvent.clear(emailInput);
    await userEvent.clear(passwordInput);

    // 各フィールドからフォーカスを外してバリデーションをトリガー
    await userEvent.tab(); // usernameからフォーカスアウト
    await userEvent.tab(); // emailからフォーカスアウト
    await userEvent.tab(); // passwordからフォーカスアウト

    await waitFor(() => {
      expect(
        screen.getByText(/ユーザー名は3文字以上である必要があります/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/有効なメールアドレスを入力してください/i)
      ).toBeInTheDocument();
      expect(
        screen.getByText(/パスワードは8文字以上である必要があります/i)
      ).toBeInTheDocument();
      expect(registerButton).toBeDisabled(); // ボタンが無効になっていることを確認
    });

    expect(api.registerUser).not.toHaveBeenCalled(); // APIが呼び出されていないこと
  });

  it("メールアドレスが無効な場合、バリデーションエラーを表示すること", async () => {
    renderWithAuthContext(<RegisterForm />);

    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "invalid-email"
    );
    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    // waitFor を使用して、エラーメッセージが DOM に現れるまで待機
    await waitFor(() => {
      expect(
        screen.getByText("有効なメールアドレスを入力してください")
      ).toBeInTheDocument();
    });
    expect(api.registerUser).not.toHaveBeenCalled(); // APIが呼び出されていないこと
  });

  it("APIエラーが発生した場合、エラーメッセージを表示すること", async () => {
    const mockRegisterUser = api.registerUser as jest.Mock;
    mockRegisterUser.mockResolvedValueOnce({
      error: "Username or email already exists",
    });

    renderWithAuthContext(<RegisterForm />);

    await userEvent.type(screen.getByLabelText(/ユーザー名/i), "duplicate");
    await userEvent.type(
      screen.getByLabelText(/メールアドレス/i),
      "duplicate@example.com"
    );
    await userEvent.type(screen.getByLabelText(/パスワード/i), "password123");

    await userEvent.click(screen.getByRole("button", { name: /登録/i }));

    // 修正: `queryByText` や `getAllByText` ではなく、単一のエラーメッセージを期待
    await waitFor(() => {
      expect(
        screen.getByText("Username or email already exists")
      ).toBeInTheDocument();
    });
    expect(api.loginUser).not.toHaveBeenCalled(); // 自動ログインは呼ばれない
    expect(mockLogin).not.toHaveBeenCalled(); // loginは呼ばれない
    expect(mockPush).not.toHaveBeenCalled(); // リダイレクトもされない
  });
});
